package dirsync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/silvasur/buzhash"
	"lukechampine.com/blake3"
)

type Source interface{}
type Sink interface{}

type Dir struct {
	Name  string
	Dirs  []*Dir
	Files []*File
}

type File struct {
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
) (*File, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading fileinfo: %w", err)
	}
	out := &File{
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

// trace sink with hashed files and trees
// - starting from the root and going deeper into each children,
//   for all file in `src` and not on `sink`, upload & create dirs

func Trace(ctx context.Context, root string, fs fs.ReadDirFS) (*Dir, error) {
	return trace(ctx, "", root, fs)
}

func trace(ctx context.Context, parent, base string, fs fs.ReadDirFS) (*Dir, error) {
	dir := &Dir{Name: base}

	fsEntries, err := fs.ReadDir(filepath.Join(parent, base))
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}

	// `fsEntries`` is guaranteed to be sorted, per `fs.ReadDir`'s contract
	for _, fsEntry := range fsEntries {
		if fsEntry.IsDir() {
			path := filepath.Join(parent, base)
			child, err := trace(ctx, path, fsEntry.Name(), fs)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", path, err)
			}
			dir.Dirs = append(dir.Dirs, child)
		} else {
			file := &File{
				Name: fsEntry.Name(),
			}
			dir.Files = append(dir.Files, file)
		}
	}
	return dir, nil
}
