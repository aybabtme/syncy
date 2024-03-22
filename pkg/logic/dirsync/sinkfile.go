package dirsync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/silvasur/buzhash"
	"lukechampine.com/blake3"
)

type SinkFile struct {
	Name string

	Size    uint64
	ModTime time.Time
	Blocks  []*Block
}

type Block struct {
	FastSig   uint32   // buzhash
	StrongSig [32]byte // blake3 sum256
	Size      uint32
}

func SumSinkFile(
	ctx context.Context,
	file fs.File,
	blockSize uint32,
) (*SinkFile, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading fileinfo: %w", err)
	}
	out := &SinkFile{
		Name:    fi.Name(),
		Size:    uint64(fi.Size()),
		ModTime: fi.ModTime(),
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

		b := &Block{
			FastSig:   fastSig,
			StrongSig: strongSig,
			Size:      uint32(n),
		}
		out.Blocks = append(out.Blocks, b)
		// reset the reused parts
		buz.Reset()
	}
	return out, nil
}
