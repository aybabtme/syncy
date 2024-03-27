package dirsync

import (
	"context"
	"fmt"
	"io"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
)

type Sink interface {
	GetSignatures(ctx context.Context) (*typesv1.DirSum, error)
	CreateFile(ctx context.Context, path *typesv1.Path, fi *typesv1.FileInfo, r io.Reader) error
	PatchFile(ctx context.Context, dir *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, r io.Reader) error
	DeleteFiles(context.Context, []DeleteOp) error
}

type SumDB interface {
	Stat(ctx context.Context, path string) (*typesv1.FileInfo, bool, error)
	// ListDir returns entries in a dir, ordered by name.
	ListDir(ctx context.Context, path string) ([]*typesv1.FileInfo, bool, error)
	GetFileSum(ctx context.Context, path string, fi *typesv1.FileInfo) (*typesv1.FileSum, bool, error)
}

func TraceSink(ctx context.Context, root string, sumDB SumDB) (*typesv1.DirSum, error) {
	return trace(ctx, nil, root, sumDB)
}

func trace(ctx context.Context, parent *typesv1.Path, base string, sumDB SumDB) (*typesv1.DirSum, error) {
	path := typesv1.PathJoin(parent, base)
	fi, _, err := sumDB.Stat(ctx, typesv1.StringFromPath(path))
	if err != nil {
		return nil, fmt.Errorf("stating dir: %w", err)
	}

	dir := &typesv1.DirSum{
		Path: parent,
		Info: fi,
	}

	fsEntries, ok, err := sumDB.ListDir(ctx, typesv1.StringFromPath(path))
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("no such dir: %q", typesv1.StringFromPath(path))
	}

	// `fsEntries`` is guaranteed to be sorted, per `sumDB.ListDir`'s contract
	for _, fsEntry := range fsEntries {
		if fsEntry.IsDir {
			path := typesv1.PathJoin(parent, base)
			child, err := trace(ctx, path, fsEntry.Name, sumDB)
			if err != nil {
				return nil, fmt.Errorf("tracing %q, %w", typesv1.StringFromPath(path), err)
			}
			dir.Dirs = append(dir.Dirs, child)
			dir.Info.Size += child.Info.Size
		} else {
			path := typesv1.PathJoin(parent, base)
			file, ok, err := sumDB.GetFileSum(ctx, typesv1.StringFromPath(path), fsEntry)
			if err != nil {
				return nil, fmt.Errorf("looking up filesum for file %q in %q: %w", fsEntry.Name, typesv1.StringFromPath(path), err)
			}
			if !ok {
				return nil, fmt.Errorf("missing filesum for file %q in %q", fsEntry.Name, typesv1.StringFromPath(path))
			}

			dir.Files = append(dir.Files, file)
			dir.Info.Size += file.Info.Size
		}
	}
	return dir, nil
}
