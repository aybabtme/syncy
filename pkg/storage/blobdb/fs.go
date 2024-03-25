package blobdb

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	"google.golang.org/protobuf/proto"
)

type Blob interface {
	Stat(context.Context, string) (*typesv1.FileInfo, bool, error)
	ListDir(context.Context, string) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context) (*typesv1.DirSum, error)
	CreatePath(ctx context.Context, filename string, isDir bool, fn func(w io.Writer) error) error
	PatchPath(ctx context.Context, filename string, isDir bool, sum *typesv1.FileSum, fn func(orig io.ReadSeeker, w io.Writer) error) error
	DeletePaths(ctx context.Context, filenames []string) error
}

var _ Blob = (*LocalFS)(nil)

type LocalFS struct {
	root    string
	scratch string

	mu    sync.Mutex
	locks map[string]struct{}
}

type FS interface {
	fs.ReadDirFS
	fs.StatFS
}

// NewLocalFS creates a `Blob` that works against a local filesystem.
// `root` is where files are stored.
// `scratch` is where files being constructed are stored.
func NewLocalFS(root, scratch string) *LocalFS {
	return &LocalFS{root: root, scratch: scratch, locks: make(map[string]struct{})}
}

func (lfs *LocalFS) Stat(ctx context.Context, path string) (*typesv1.FileInfo, bool, error) {
	filename := filepath.Join(lfs.root, path)
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("localfs: can't stat, %w", err)
	}
	return typesv1.FileInfoFromFS(fi), true, nil
}

func (lfs *LocalFS) ListDir(ctx context.Context, path string) ([]*typesv1.FileInfo, bool, error) {
	filename := filepath.Join(lfs.root, path)
	dirs, err := os.ReadDir(filename)
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

func (lfs *LocalFS) GetFileSum(ctx context.Context, filename string) (*typesv1.FileSum, bool, error) {
	filepath := filepath.Join(lfs.root, filename)
	f, err := os.Open(filepath)
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

func (lfs *LocalFS) CreatePath(ctx context.Context, path string, isDir bool, fn func(w io.Writer) error) error {
	endPath := filepath.Join(lfs.root, path)

	unlock, locked := lfs.takeLock(path)
	if locked {
		return fmt.Errorf("path is already locked by another request, try again later")
	}
	defer unlock()

	if isDir {
		// we don't use `fi` because we don't want to let
		// requester dictate our local file access policies
		err := os.Mkdir(endPath, 0755)
		if err != nil {
			return fmt.Errorf("creating dir: %w", err)
		}
		return nil
	}
	return lfs.withAtomicFileSwap(path, fn)
}

func (lfs *LocalFS) PatchPath(ctx context.Context, path string, isDir bool, wantSum *typesv1.FileSum, fn func(orig io.ReadSeeker, w io.Writer) error) error {
	endPath := filepath.Join(lfs.root, path)

	unlock, locked := lfs.takeLock(path)
	if locked {
		return fmt.Errorf("path is already locked by another request, try again later")
	}
	defer unlock()

	if isDir {
		// we don't use `fi` because we don't want to let
		// requester dictate our local file access policies
		err := os.Mkdir(endPath, 0755)
		if err == os.ErrExist {
		} else if err != nil {
			return fmt.Errorf("creating dir: %w", err)
		}
		return nil
	}

	origf, err := os.Open(endPath)
	if err != nil {
		return fmt.Errorf("opening original file: %w", err)
	}
	defer origf.Close()
	gotSum, err := dirsync.ComputeFileSum(ctx, origf)
	if err != nil {
		return fmt.Errorf("computing file sum: %w", err)
	}
	if !proto.Equal(wantSum, gotSum) {
		return fmt.Errorf("file sum mismatch, the file you're trying to patch is not the same, or has changed, since computing the submitted filesum")
	}

	_, err = origf.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking back to begining of original file: %w", err)
	}
	return lfs.withAtomicFileSwap(path, func(w io.Writer) error {
		return fn(origf, w)
	})
}

func (lfs *LocalFS) withAtomicFileSwap(filename string, fn func(w io.Writer) error) error {
	endPath := filepath.Join(lfs.root, filename)
	tmpFile, err := os.CreateTemp(lfs.scratch, filename)
	if err != nil {
		return fmt.Errorf("creating temp file in scratch location: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}
	}()
	if err := fn(tmpFile); err != nil {
		return fmt.Errorf("writing to scratch file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("flushing scratch file: %w", err)
	}
	if err := os.Rename(tmpFile.Name(), endPath); err != nil {
		return fmt.Errorf("atomic swap of old file with new file: %w", err)
	}
	return nil
}

func (lfs *LocalFS) DeletePaths(ctx context.Context, paths []string) error {
	for _, path := range paths {
		lfs.deletePath(ctx, path)
	}
	return nil
}

func (lfs *LocalFS) deletePath(ctx context.Context, path string) error {
	unlock, locked := lfs.takeLock(path)

	if locked {
		return fmt.Errorf("path is already locked by another request, try again later")
	}
	defer unlock()

	endPath := filepath.Join(lfs.root, path)
	return os.Remove(endPath)
}

func (lfs *LocalFS) takeLock(path string) (func(), bool) {
	lfs.mu.Lock()
	_, locked := lfs.locks[path]
	if locked {
		lfs.mu.Unlock()
		return nil, false
	}
	lfs.locks[path] = struct{}{}
	lfs.mu.Unlock()
	return func() {
		lfs.mu.Lock()
		delete(lfs.locks, path)
		lfs.mu.Unlock()
	}, true
}
