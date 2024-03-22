package dirsync

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/silvasur/buzhash"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lukechampine.com/blake3"
)

// type SinkFile struct {
// 	Name string

// 	Size    uint64
// 	ModTime time.Time
// 	Mode    uint32
// 	Blocks  []*Block
// }

// type Block struct {
// 	FastSig   uint32   // buzhash
// 	StrongSig [32]byte // blake3 sum256
// 	Size      uint32
// }

func ComputeFileSum(
	ctx context.Context,
	file fs.File,
	blockSize uint32,
) (*typesv1.FileSum, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading fileinfo: %w", err)
	}
	out := &typesv1.FileSum{
		Info: &typesv1.FileInfo{
			Name:    fi.Name(),
			Size:    uint64(fi.Size()),
			ModTime: timestamppb.New(fi.ModTime()),
			Mode:    uint32(fi.Mode()),
		},
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
			StrongSig: strongSig[:],
			Size:      uint32(n),
		}
		out.SumBlocks = append(out.SumBlocks, b)
		// reset the reused parts
		buz.Reset()
	}
	return out, nil
}
