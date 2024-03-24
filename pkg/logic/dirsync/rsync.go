package dirsync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/silvasur/buzhash"
	"lukechampine.com/blake3"
)

type PatchOp struct {
	Path *typesv1.Path
	Dir  *DirPatchOp
	File *FilePatchOp
}

type DirPatchOp struct {
	DirPatches []DirPatch
}

type DirPatch struct {
	SetMode *uint32
}

type FilePatchOp struct {
	S       *typesv1.FileSum
	S_prime *typesv1.FileBlockPatch
}

func blockSize(filesize int64) uint32 {
	if filesize < 490000 {
		return 700
	}
	blockSize := uint32(math.Sqrt(float64(filesize)))
	if blockSize > 131072 {
		return 131072
	}
	return blockSize
}

func Rsync(ctx context.Context, src io.Reader, dstSum *typesv1.FileSum, patchData func([]byte) (int, error), patchBlockID func(uint32) (int, error)) (int, error) {
	maxBufferSize := 1 << 20 // 1 MiB

	if len(dstSum.SumBlocks) == 0 {
		return scanAndEmitBlocks(ctx, src, maxBufferSize, patchData)
	}

	// TODO: if slow, implement a faster lookup algo:
	// - 32b sig is turned into an index of 16b sig, sorted
	// - colliding 32b sig are appended in the 16b index
	// - lookup with 16b sig, on hit, iterate 32b list to find 32b sig, then confirm with strong sig
	fastSigIndex := make(map[uint32]uint32)
	for i, block := range dstSum.SumBlocks {
		if _, ok := fastSigIndex[block.FastSig]; !ok {
			fastSigIndex[block.FastSig] = uint32(i)
		}
	}

	blockSize := dstSum.BlockSize

	br := bufio.NewReader(src)
	fastHash := buzhash.NewBuzHash(blockSize)

	written := 0
	n := 0
	block := make([]byte, 0, blockSize)
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			// we were accumulating non-matching data, send it before shutting down
			if len(block) > 0 {
				n, err = patchData(block)
				written += n
				if err != nil {
					return written, fmt.Errorf("emiting data block: %w", err)
				}
			}
			return written, nil
		}
		block = append(block, b)
		fastSig := fastHash.HashByte(b)
		blockIdx, ok := fastSigIndex[fastSig]
		if !ok {
			// no match, accumulate and continue hashing
			if len(block) < maxBufferSize {
				continue
			}
			// except if the block is getting too large
			// we were accumulating non-matching data, send it at once
			n, err = patchData(block)
			written += n
			if err != nil {
				return written, fmt.Errorf("emiting data block: %w", err)
			}
			block = block[:0:blockSize]
			fastHash.Reset()
			continue
		}
		// potential match in the last `blockSize` of data
		matchStart := imax(len(block)-int(blockSize), 0)
		matchEnd := len(block)

		bSig := typesv1.Array32ByteFromUint256(dstSum.SumBlocks[blockIdx].StrongSig)
		aSig := blake3.Sum256(block[matchStart:matchEnd])
		if bSig != aSig {
			// not a real match
			continue
		}
		// if the match starts at >0, then we have non-matching data in the head of
		// the block
		if matchStart > 0 {
			// we were accumulating non-matching data, send it at once
			n, err = patchData(block[:matchStart])
			written += n
			if err != nil {
				return written, fmt.Errorf("emiting data block: %w", err)
			}
		}
		// send the matching block index
		n, err = patchBlockID(blockIdx)
		written += n
		if err != nil {
			return written, fmt.Errorf("emiting block id %d: %w", blockIdx, err)
		}
		// reset the block to its original capacity
		block = block[:0:blockSize]
		fastHash.Reset()
	}
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// send in blocks to avoid loading all the file in memory. ideally we could just `io.Copy` without sending it via protobuf
// TODO: replace protobuf for the upload/patch paths
func scanAndEmitBlocks(ctx context.Context, src io.Reader, blockSize int, dst func([]byte) (int, error)) (int, error) {
	more := true
	block := make([]byte, blockSize) // TODO: use sync.Pool
	written := 0
	for i := 0; more; i++ {
		n, err := io.ReadFull(src, block)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			more = false
		case nil:
			// continue
		default:
			return written, fmt.Errorf("reading block %d: %w", i, err)
		}
		if n == 0 {
			return written, nil
		}
		n, err = dst(block[:n])
		if err != nil {
			return written, fmt.Errorf("sending block %d: %w", i, err)
		}
		written += n
	}
	return written, nil
}

type FilePatcher struct {
	original io.ReadSeeker
	target   io.Writer
	sum      *typesv1.FileSum
}

func NewFilePatcher(original io.ReadSeeker, target io.Writer, sum *typesv1.FileSum) *FilePatcher {
	return &FilePatcher{original: original, target: target, sum: sum}
}

func (fp *FilePatcher) WriteBlock(idx uint32) (int, error) {
	if int(idx) >= len(fp.sum.SumBlocks) {
		return -1, fmt.Errorf("invalid patch, block ID %d out of range (max %d)", idx, len(fp.sum.SumBlocks))
	}
	block := fp.sum.SumBlocks[idx]
	fileOffsetStart := int64(fp.sum.BlockSize) * int64(idx)
	_, err := fp.original.Seek(fileOffsetStart, io.SeekStart)
	if err != nil {
		return -1, fmt.Errorf("seeking to %d in old file: %v", fileOffsetStart, err)
	}
	n, err := io.CopyN(fp.target, fp.original, int64(block.Size))
	if err != nil {
		return int(n), fmt.Errorf("copying patch block from old file to new file: %v", err)
	}
	return int(n), nil
}

func (fp *FilePatcher) Copy(r io.Reader) (int, error) {
	n, err := io.Copy(fp.target, r)
	return int(n), err
}

func (fp *FilePatcher) WriteData(p []byte) (int, error) {
	return fp.target.Write(p)
}
