package dirsync

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/silvasur/buzhash"
	"lukechampine.com/blake3"
)

func ComputeFileSum(ctx context.Context, file fs.File) (*typesv1.FileSum, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading fileinfo: %w", err)
	}
	pfi := typesv1.FileInfoFromFS(fi)
	return computeFileSum(ctx, file, pfi, blockSize(fi.Size()))
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
