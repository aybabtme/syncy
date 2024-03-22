package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

type Sink interface {
	GetSignatures(context.Context) (*SinkDir, error)
	CreateFile(context.Context, CreateOp) error
	DeleteFiles(context.Context, []DeleteOp) error
	PatchFile(context.Context, PatchOp) error
}

type SinkDir struct {
	Name  string
	Dirs  []*SinkDir
	Files []*SinkFile
}

func TraceSink(ctx context.Context, root string, fs fs.ReadDirFS) (*SinkDir, error) {
	return trace(ctx, "", root, fs)
}

func trace(ctx context.Context, parent, base string, fs fs.ReadDirFS) (*SinkDir, error) {
	dir := &SinkDir{Name: base}

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
			file := &SinkFile{
				Name: fsEntry.Name(),
			}
			dir.Files = append(dir.Files, file)
		}
	}
	return dir, nil
}
