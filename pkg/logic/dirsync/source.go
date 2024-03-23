package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"
)

type Source interface {
	fs.ReadDirFS
	fs.StatFS
}

type CreateOp struct {
	Path  string
	IsDir bool
	Mode  uint32
}

type DeleteOp struct {
	Path string
}

type SourceDir struct {
	Name  string
	Mode  uint32
	Size  uint64
	Dirs  []*SourceDir
	Files []*SourceFile
}

type SourceFile struct {
	Name string

	Size    uint64
	ModTime time.Time
	Mode    uint32
}

func TraceSource(ctx context.Context, root string, src Source) (*SourceDir, error) {
	dirinfo, err := src.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stating root: %w", err)
	}

	return traceSource(ctx, "", root, dirinfo, src)
}

func traceSource(ctx context.Context, parent, base string, fi fs.FileInfo, fs Source) (*SourceDir, error) {
	dir := &SourceDir{
		Name: base,
		Mode: uint32(fi.Mode()),
	}

	fsEntries, err := fs.ReadDir(filepath.Join(parent, base))
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}

	// `fsEntries`` is guaranteed to be sorted, per `fs.ReadDir`'s contract
	for _, fsEntry := range fsEntries {
		path := filepath.Join(parent, base)
		fsfi, err := fs.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stating %q: %w", path, err)
		}

		// TODO: encode symlinks somehow?
		if fsEntry.IsDir() {
			child, err := traceSource(ctx, path, fsEntry.Name(), fsfi, fs)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", path, err)
			}
			dir.Dirs = append(dir.Dirs, child)
			dir.Size += child.Size
		} else if fsfi.Mode().IsRegular() {
			file := &SourceFile{
				Name:    fsEntry.Name(),
				Size:    uint64(fsfi.Size()),
				Mode:    uint32(fsfi.Mode()),
				ModTime: fsfi.ModTime(),
			}
			dir.Files = append(dir.Files, file)
			dir.Size += file.Size
		}
	}
	return dir, nil
}
