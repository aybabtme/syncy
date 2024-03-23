package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/hashicorp/go-multierror"
)

const enforceAssertions = true // TODO: turn this off when confident enough

func assert(msg string, cond bool) {
	if !enforceAssertions {
		return
	}
	if !cond {
		panic(msg)
	}
}

type Params struct {
	MaxParallelFileStreams int
}

func Sync(ctx context.Context, root string, src Source, sink Sink, params Params) error {
	// We diff two trees instead of a list of items. By diffing trees top-down, we can issue
	// 1 deletes for an entire tree, instead of a list of deletes for each file under a tree.
	// It also allows for the opportunity (future) to make merkle trees to efficiently identify
	// branches in the tree that have changes (not done here).
	sigs, err := sink.GetSignatures(ctx)
	if err != nil {
		return fmt.Errorf("getting signatures from sink: %w", err)
	}
	createOps, deleteOps, patchOps, err := ComputeTreeDiff(ctx, root, src, sigs)
	if err != nil {
		return fmt.Errorf("computing tree diff: %w", err)
	}

	sem := make(chan struct{}, params.MaxParallelFileStreams)

	errc := make(chan error, 3)
	var wg sync.WaitGroup
	for _, createOp := range createOps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			withSem(ctx, sem, func() {
				err := upload(ctx, src, sink, createOp)
				trySendErr(ctx, errc, err)
			})
		}()
	}
	for _, patchOp := range patchOps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			withSem(ctx, sem, func() {
				_ = patchOp
				// err := Rsync(ctx, src, sink, patchOp)
				trySendErr(ctx, errc, err)
			})
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sink.DeleteFiles(ctx, deleteOps); err != nil {
			trySendErr(ctx, errc, err)
		}
	}()
	go func() {
		wg.Wait()
		close(errc)
	}()
	var merr *multierror.Error
	for {
		select {
		case err, ok := <-errc:
			if err != nil {
				merr = multierror.Append(merr, err)
			}
			if !ok {
				return merr
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func trySendErr(ctx context.Context, errc chan error, err error) {
	if err != nil {
		select {
		case errc <- err:
		case <-ctx.Done():
		}
	}
}

func withSem(ctx context.Context, sem chan struct{}, fn func()) {
	select {
	case sem <- struct{}{}: // try to take a semaphore
		fn()
	case <-ctx.Done():
		return // abort
	}
	select {
	case <-sem:
	default:
	}
}

func ComputeTreeDiff(ctx context.Context, root string, src Source, sinkDir *typesv1.DirSum) ([]CreateOp, []DeleteOp, []PatchOp, error) {
	srcDir, err := TraceSource(ctx, root, src)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("enumerating files on source: %w", err)
	}
	return computeDirDiff(ctx, src, root, srcDir, sinkDir)
}

func computeDirDiff(ctx context.Context, fs fs.FS, path string, src *SourceDir, sink *typesv1.DirSum) ([]CreateOp, []DeleteOp, []PatchOp, error) {
	var (
		createOps []CreateOp // set(Src)  -  set(Sink) = set to create
		deleteOps []DeleteOp // set(Src)  ∩  set(Sink) = set to patch
		patchOps  []PatchOp  // set(Sink) -  set(Src)  = set to delete
	)

	// 1) handle `Src_dir` and `Sink_dir` first

	for _, srcDir := range src.Dirs {
		// set(Src_dir) - set(Sink_dir)
		sinkDir, found := sinkHasDirNamed(sink, srcDir.Name)
		if !found {
			// the entire dir is missing, so we can stop looking for
			// patches and deletes and just generate a list of creates
			// for this entire dir
			createDirOps := createOpsForDir(path, srcDir)
			createOps = append(createOps, createDirOps...)
			continue
		}
		// set(Src_dir) ∩ set(Sink_dir)
		dirPath := filepath.Join(path, srcDir.Name)
		cOps, dOps, pOps, err := computeDirDiff(ctx, fs, dirPath, srcDir, sinkDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("computing diff for directory %q: %w", dirPath, err)
		}
		createOps = append(createOps, cOps...)
		deleteOps = append(deleteOps, dOps...)
		patchOps = append(patchOps, pOps...)
		// check if the dir itself needs a patch too
		if diff := makeDirDiff(srcDir, sinkDir); diff != nil {
			op := patchDirOp(path, src.Name, diff)
			patchOps = append(patchOps, op)
		}
	}
	for _, sinkDir := range sink.Dirs {
		// set(Sink_dir) - set(Src_dir)
		if !srcHasDirNamed(src, sinkDir.Name) {
			op := deleteDirOp(path, sink)
			deleteOps = append(deleteOps, op)
		}
	}

	// 2) then do the `Src_file` and `Sink_file`
	for _, srcFile := range src.Files {
		// set(Src_file) - set(Sink_file)
		sinkFile, found := sinkHasFileNamed(sink, srcFile.Name)
		if !found {
			createOps = append(createOps, createFileOp(path, srcFile))
			continue
		}
		// set(Src_file) ∩ set(Sink_file)
		if diff := makeFileDiff(srcFile, sinkFile); diff != nil {
			op := patchFileOp(path, diff)
			patchOps = append(patchOps, op)
		}
	}
	for _, sinkFile := range sink.Files {
		// set(Sink_file) - set(Src_file)
		if !srcHasFileNamed(src, sinkFile.Info.Name) {
			op := deleteFileOp(path, sinkFile)
			deleteOps = append(deleteOps, op)
		}
	}

	return createOps, deleteOps, patchOps, nil
}

func ptr[E any](v E) *E {
	return &v
}

func createOpsForDir(path string, dir *SourceDir) []CreateOp {
	ops := make([]CreateOp, 0, len(dir.Dirs)+len(dir.Files)+1)
	ops = append(ops, createDirOp(path, dir))
	path = filepath.Join(path, dir.Name)
	// create files first, so that if we restart the process, we will
	// have entire directories to transfer at once, which will help expedite
	// the search
	for _, file := range dir.Files {
		fpath := filepath.Join(path, file.Name)
		ops = append(ops, CreateOp{Path: fpath})
	}
	for _, dir := range dir.Dirs {
		ops = append(ops, createOpsForDir(path, dir)...)
	}
	return ops
}

func sinkHasDirNamed(sink *typesv1.DirSum, name string) (*typesv1.DirSum, bool) {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(sink.Dirs, func(a, b *typesv1.DirSum) int {
		return strings.Compare(a.Name, b.Name)
	}))
	i, found := slices.BinarySearchFunc(sink.Dirs, name, func(dir *typesv1.DirSum, name string) int {
		return strings.Compare(dir.Name, name)
	})
	if !found {
		return nil, false
	}
	return sink.Dirs[i], found
}

func sinkHasFileNamed(sink *typesv1.DirSum, name string) (*typesv1.FileSum, bool) {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(sink.Files, func(a, b *typesv1.FileSum) int {
		return strings.Compare(a.Info.Name, b.Info.Name)
	}))
	i, found := slices.BinarySearchFunc(sink.Files, name, func(file *typesv1.FileSum, name string) int {
		return strings.Compare(file.Info.Name, name)
	})
	if !found {
		return nil, false
	}
	return sink.Files[i], found
}

func srcHasDirNamed(src *SourceDir, name string) bool {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(src.Dirs, func(a, b *SourceDir) int {
		return strings.Compare(a.Name, b.Name)
	}))
	_, found := slices.BinarySearchFunc(src.Dirs, name, func(dir *SourceDir, name string) int {
		return strings.Compare(dir.Name, name)
	})
	return found
}

func srcHasFileNamed(src *SourceDir, name string) bool {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(src.Files, func(a, b *SourceFile) int {
		return strings.Compare(a.Name, b.Name)
	}))
	_, found := slices.BinarySearchFunc(src.Files, name, func(file *SourceFile, name string) int {
		return strings.Compare(file.Name, name)
	})
	return found
}

func createDirOp(path string, dir *SourceDir) CreateOp {
	return CreateOp{
		Path:  filepath.Join(path, dir.Name),
		IsDir: true,
		Mode:  dir.Mode,
	}
}

func deleteDirOp(path string, sink *typesv1.DirSum) DeleteOp {
	return DeleteOp{
		Path: filepath.Join(path, sink.Name),
	}
}

func patchDirOp(path string, dirname string, diff *DirPatchOp) PatchOp {
	return PatchOp{
		Path: filepath.Join(path, dirname),
		Dir:  diff,
	}
}

func makeDirDiff(src *SourceDir, sink *typesv1.DirSum) *DirPatchOp {
	out := &DirPatchOp{}
	return out
}

func createFileOp(path string, file *SourceFile) CreateOp {
	return CreateOp{
		Path: filepath.Join(path, file.Name),
		Mode: file.Mode,
	}
}

func deleteFileOp(path string, sink *typesv1.FileSum) DeleteOp {
	return DeleteOp{
		Path: filepath.Join(path, sink.Info.Name),
	}
}

func patchFileOp(path string, diff *FilePatchOp) PatchOp {
	return PatchOp{
		Path: path,
		File: diff,
	}
}

func makeFileDiff(src *SourceFile, sink *typesv1.FileSum) *FilePatchOp {
	out := &FilePatchOp{}
	return out
}

func upload(ctx context.Context, A Source, B Sink, createOp CreateOp) error {
	panic("todo")
}
