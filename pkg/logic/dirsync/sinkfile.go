package dirsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/silvasur/buzhash"
	"lukechampine.com/blake3"
)

func ComputeFileSum(ctx context.Context, file fs.File, fi *typesv1.FileInfo) (*typesv1.FileSum, error) {
	return computeFileSum(ctx, file, fi, blockSize(fi.Size))
}

func computeFileSum(
	ctx context.Context,
	file io.Reader,
	fi *typesv1.FileInfo,
	blockSize uint32,
) (*typesv1.FileSum, error) {
	out := &typesv1.FileSum{
		Info:      fi,
		BlockSize: blockSize,
	}
	buz := buzhash.NewBuzHash(blockSize)
	more := true
	block := make([]byte, blockSize) // TODO: use sync.Pool
loop:
	for i := 0; more; i++ {
		n, err := io.ReadFull(file, block)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			more = false
		case nil:
			// continue
		default:
			return nil, fmt.Errorf("reading block %d: %w", i, err)
		}
		if n == 0 {
			break loop
		}
		_, _ = buz.Write(block[:n])
		fastSig := buz.Sum32()
		strongSig := blake3.Sum256(block[:n])

		b := &typesv1.FileSumBlock{
			FastSig:   fastSig,
			StrongSig: typesv1.Uint256FromArray32Byte(strongSig),
			Size:      uint32(n),
		}
		out.SumBlocks = append(out.SumBlocks, b)
		// reset the reused parts
		buz.Reset()
	}
	return out, nil
}

func FileMatchesFileSum(ctx context.Context, sum *typesv1.FileSum, file fs.File, size uint64) (bool, error) {
	return fileMatchesFileSum(ctx, sum, file, blockSize(size))
}

func fileMatchesFileSum(
	ctx context.Context,
	sum *typesv1.FileSum,
	file io.Reader,
	blockSize uint32,
) (bool, error) {
	buz := buzhash.NewBuzHash(blockSize)
	more := true
	block := make([]byte, blockSize) // TODO: use sync.Pool
loop:
	for i := 0; more; i++ {

		if i >= len(sum.SumBlocks) {
			return false, nil // new file has more blocks, so clearly it's not equal
		}

		n, err := io.ReadFull(file, block)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			more = false
		case nil:
			// continue
		default:
			return false, fmt.Errorf("reading block %d: %w", i, err)
		}
		if n == 0 {
			break loop
		}

		expectBlock := sum.SumBlocks[i]

		_, _ = buz.Write(block[:n])

		wantFastSig := expectBlock.FastSig
		gotFastSig := buz.Sum32()
		if wantFastSig != gotFastSig {
			return false, nil // fast sig didn't match
		}
		wantStrongSig := typesv1.Array32ByteFromUint256(expectBlock.StrongSig)
		gotStrongSig := blake3.Sum256(block[:n])
		if !bytes.Equal(wantStrongSig[:], gotStrongSig[:]) {
			return false, nil // fast sig didn't match
		}
		// reset the reused parts
		buz.Reset()
	}
	return true, nil
}
