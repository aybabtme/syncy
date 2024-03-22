package dirsync

import (
	"context"
	"fmt"
	"path/filepath"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
)

type Sink interface {
	GetSignatures(ctx context.Context, blockSize uint32) (*typesv1.DirSum, error)
	CreateFile(context.Context, CreateOp) error
	DeleteFiles(context.Context, []DeleteOp) error
	PatchFile(context.Context, PatchOp) error
}

type SumDB interface {
	// ListDir returns entries in a dir, ordered by name.
	ListDir(ctx context.Context, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetFileSum(ctx context.Context, path *typesv1.Path, name string, blockSize uint32) (*typesv1.FileSum, bool, error)
}

func TraceSink(ctx context.Context, root string, sumDB SumDB, blockSize uint32) (*typesv1.DirSum, error) {
	return trace(ctx, nil, root, sumDB, blockSize)
}

func filepathJoin(parent *typesv1.Path, name string) *typesv1.Path {
	if parent == nil {
		return &typesv1.Path{Elements: []string{name}}
	}
	return &typesv1.Path{Elements: append(parent.Elements, name)}
}

func pathString(path *typesv1.Path) string {
	return filepath.Join(path.Elements...)
}

func trace(ctx context.Context, parent *typesv1.Path, base string, sumDB SumDB, blockSize uint32) (*typesv1.DirSum, error) {
	dir := &typesv1.DirSum{
		Path: parent,
		Name: base,
	}

	path := filepathJoin(parent, base)
	fsEntries, ok, err := sumDB.ListDir(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("no such dir: %q", pathString(path))
	}

	// `fsEntries`` is guaranteed to be sorted, per `sumDB.ListDir`'s contract
	for _, fsEntry := range fsEntries {
		if fsEntry.IsDir {
			path := filepathJoin(parent, base)
			child, err := trace(ctx, path, fsEntry.Name, sumDB, blockSize)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", pathString(path), err)
			}
			dir.Dirs = append(dir.Dirs, child)
			dir.Size += child.Size
		} else {
			path := filepathJoin(parent, base)
			file, ok, err := sumDB.GetFileSum(ctx, path, fsEntry.Name, blockSize)
			if err != nil {
				return nil, fmt.Errorf("looking up filesum for file %q in %q: %w", fsEntry.Name, pathString(path), err)
			}
			if !ok {
				return nil, fmt.Errorf("missing filesum for file %q in %q", fsEntry.Name, pathString(path))
			}

			dir.Files = append(dir.Files, file)
			dir.Size += file.Info.Size
		}
	}
	return dir, nil
}
