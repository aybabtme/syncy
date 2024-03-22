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

type PatchOp struct {
	Path string
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
	S       SinkFile
	S_prime SinkFile
}

type SourceDir struct {
	Name  string
	Mode  uint32
	Dirs  []*SourceDir
	Files []*SourceFile
}

type SourceFile struct {
	Name string

	Size    uint64
	ModTime time.Time
	Mode    uint32
}

// trace sink with hashed files and trees
// - starting from the root and going deeper into each children,
//   for all file in `src` and not on `sink`, upload & create dirs

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

		if fsEntry.IsDir() {
			child, err := traceSource(ctx, path, fsEntry.Name(), fsfi, fs)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", path, err)
			}
			dir.Dirs = append(dir.Dirs, child)
		} else {
			file := &SourceFile{
				Name:    fsEntry.Name(),
				Size:    uint64(fsfi.Size()),
				Mode:    uint32(fsfi.Mode()),
				ModTime: fsfi.ModTime(),
			}
			dir.Files = append(dir.Files, file)
		}
	}
	return dir, nil
}
