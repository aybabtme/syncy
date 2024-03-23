package blobdb

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
)

type Blob interface {
	Stat(context.Context, *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(context.Context, *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context) (*typesv1.DirSum, error)
}

var _ Blob = (*LocalFS)(nil)

type LocalFS struct {
	fs FS
}

type FS interface {
	fs.ReadDirFS
	fs.StatFS
}

func NewLocalFS(fs FS) *LocalFS {
	return &LocalFS{fs: fs}
}

func (lfs *LocalFS) Stat(ctx context.Context, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	fi, err := lfs.fs.Stat(fpath(path))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("localfs: can't stat, %w", err)
	}
	return typesv1.FileInfoFromFS(fi), true, nil
}

func (lfs *LocalFS) ListDir(ctx context.Context, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	fpath := fpath(path)
	dirs, err := lfs.fs.ReadDir(fpath)
	if os.IsNotExist(err) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("localfs: can't read dir, %w", err)
	}
	out := make([]*typesv1.FileInfo, 0, len(dirs))
	for _, de := range dirs {
		fi, err := de.Info()
		if err != nil {
			return nil, true, fmt.Errorf("localfs: can't get dir entry info, %w", err)
		}
		// TODO: maybe encode symlinks somehow?
		if fi.Mode().IsRegular() || fi.IsDir() {
			out = append(out, typesv1.FileInfoFromFS(fi))
		}
	}
	return out, true, nil
}

func (lfs *LocalFS) GetSignature(ctx context.Context) (*typesv1.DirSum, error) {
	return dirsync.TraceSink(ctx, "", lfs)
}

func (lfs *LocalFS) GetFileSum(ctx context.Context, path *typesv1.Path, name string) (*typesv1.FileSum, bool, error) {
	filepath := joinpath(path, name)
	f, err := lfs.fs.Open(filepath)
	if os.IsNotExist(err) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("localfs: opening %q: %w", filepath, err)
	}
	defer f.Close()
	fs, err := dirsync.ComputeFileSum(ctx, f)
	if err != nil {
		return nil, true, fmt.Errorf("localfs: computing file sum: %w", err)
	}
	return fs, true, nil
}

func fpath(path *typesv1.Path) string {
	return filepath.Join(path.Elements...)
}

func joinpath(parent *typesv1.Path, path string) string {
	return filepath.Join(fpath(parent), path)
}
