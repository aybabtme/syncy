package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
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
}

// trace sink with hashed files and trees
// - starting from the root and going deeper into each children, for all file in `src` and not on `sink`, upload & create dirs

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
