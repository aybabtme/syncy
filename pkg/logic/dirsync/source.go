package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
)

type Source interface {
	fs.ReadDirFS
	fs.StatFS
}

type CreateOp struct {
	ParentDir *typesv1.Path
	FileInfo  *typesv1.FileInfo
}

type DeleteOp struct {
	Path     *typesv1.Path
	FileInfo *typesv1.FileInfo
}

type SourceDir struct {
	Info  *typesv1.FileInfo
	Dirs  []*SourceDir
	Files []*SourceFile
}

type SourceFile struct {
	Info *typesv1.FileInfo
}

func TraceSource(ctx context.Context, root string, src Source) (*SourceDir, error) {
	dirinfo, err := src.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stating root: %w", err)
	}

	return traceSource(ctx, root, dirinfo, src)
}

func traceSource(ctx context.Context, base string, fi fs.FileInfo, fs Source) (*SourceDir, error) {
	dir := &SourceDir{
		Info: typesv1.FileInfoFromFS(fi),
	}
	if base == "." {
		dir.Info.Name = ""
	}

	fsEntries, err := fs.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}

	// `fsEntries`` is guaranteed to be sorted, per `fs.ReadDir`'s contract
	for _, fsEntry := range fsEntries {
		entryPath := filepath.Join(base, fsEntry.Name())
		fsfi, err := fs.Stat(entryPath)
		if err != nil {
			return nil, fmt.Errorf("stating %q: %w", entryPath, err)
		}

		// TODO: encode symlinks somehow?
		if fsEntry.IsDir() {
			child, err := traceSource(ctx, entryPath, fsfi, fs)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", entryPath, err)
			}
			dir.Dirs = append(dir.Dirs, child)

			dir.Info.Size += child.Info.Size
		} else if fsfi.Mode().IsRegular() {
			file := &SourceFile{
				Info: typesv1.FileInfoFromFS(fsfi),
			}
			dir.Files = append(dir.Files, file)
			dir.Info.Size += file.Info.Size
		}
	}
	return dir, nil
}
