package dirsync

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"google.golang.org/protobuf/proto"
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// We diff two trees instead of a list of items. By diffing trees top-down, we can issue
	// 1 deletes for an entire tree, instead of a list of deletes for each file under a tree.
	// It also allows for the opportunity (future) to make merkle trees to efficiently identify
	// branches in the tree that have changes (not done here).
	sigs, err := sink.GetSignatures(ctx)
	if err != nil {
		return fmt.Errorf("getting signatures from sink: %w", err)
	}

	rootp := typesv1.PathFromString(root)
	err = ComputeTreeDiff(ctx, rootp, src, sigs,
		func(co CreateOp) error {
			return upload(ctx, src, sink, co)
		},
		func(co PatchOp) error {
			return patch(ctx, src, sink, co)
		},
		func(co DeleteOp) error {
			return sink.DeleteFile(ctx, co)
		},
	)
	if err != nil {
		return fmt.Errorf("computing tree diff: %w", err)
	}
	return nil
}

func ComputeTreeDiff(ctx context.Context, root *typesv1.Path, src Source, sinkDir *typesv1.DirSum,
	emitCreate func(CreateOp) error,
	emitPatch func(PatchOp) error,
	emitDelete func(DeleteOp) error,
) error {
	rootp := typesv1.StringFromPath(root)
	srcDir, err := TraceSource(ctx, rootp, src)
	if err != nil {
		return fmt.Errorf("enumerating files on source: %w", err)
	}
	return computeDirDiff(ctx, src, &typesv1.Path{}, srcDir, sinkDir, emitCreate, emitPatch, emitDelete)
}

func computeDirDiff(ctx context.Context, fs fs.FS, path *typesv1.Path, src *SourceDir, sink *typesv1.DirSum,
	emitCreate func(CreateOp) error,
	emitPatch func(PatchOp) error,
	emitDelete func(DeleteOp) error,
) error {

	// 1) handle `Src_dir` and `Sink_dir` first

	for _, srcDir := range src.Dirs {
		// set(Src_dir) - set(Sink_dir)
		sinkDir, found := sinkHasDirNamed(sink, srcDir.Info.Name)
		if !found {
			// the entire dir is missing, so we can stop looking for
			// patches and deletes and just generate a list of creates
			// for this entire dir
			err := emitCreateOpsForDir(fs, path, srcDir, emitCreate)
			if err != nil {
				dirPath := typesv1.PathJoin(path, srcDir.Info.Name)
				return fmt.Errorf("emitting diff for directory %q: %w", dirPath, err)
			}
			continue
		}
		// set(Src_dir) ∩ set(Sink_dir)
		dirPath := typesv1.PathJoin(path, srcDir.Info.Name)
		err := computeDirDiff(ctx, fs, dirPath, srcDir, sinkDir, emitCreate, emitPatch, emitDelete)
		if err != nil {
			return fmt.Errorf("computing diff for directory %q: %w", dirPath, err)
		}

		// check if the dir itself needs a patch too
		if diff := makeDirDiff(srcDir, sinkDir); diff != nil {
			op := patchDirOp(path, src.Info.Name, diff)
			if err := emitPatch(op); err != nil {
				return fmt.Errorf("emitting patch dir: %w", err)
			}
		}
	}
	for _, sinkDir := range sink.Dirs {
		// set(Sink_dir) - set(Src_dir)
		if !srcHasDirNamed(src, sinkDir.Info.Name) {
			op := deleteDirOp(path, sink)
			if err := emitDelete(op); err != nil {
				return fmt.Errorf("emitting delete dir: %w", err)
			}
		}
	}

	// 2) then do the `Src_file` and `Sink_file`
	for _, srcFile := range src.Files {
		// set(Src_file) - set(Sink_file)
		sinkFile, found := sinkHasFileNamed(sink, srcFile.Info.Name)
		if !found {
			op := createFileOp(fs, path, srcFile.Info)
			if err := emitCreate(op); err != nil {
				return fmt.Errorf("emitting create file: %w", err)
			}
			continue
		}
		// set(Src_file) ∩ set(Sink_file)
		if diff, err := makeFileDiff(ctx, fs, path, srcFile, sinkFile); err != nil {
			return fmt.Errorf("computing diff for file %q: %w", srcFile.Info.Name, err)
		} else if diff != nil {
			op := patchFileOp(path, diff)
			if err := emitPatch(op); err != nil {
				return fmt.Errorf("emitting patch file: %w", err)
			}
		}
	}
	for _, sinkFile := range sink.Files {
		// set(Sink_file) - set(Src_file)
		if !srcHasFileNamed(src, sinkFile.Info.Name) {
			op := deleteFileOp(path, sinkFile)
			if err := emitDelete(op); err != nil {
				return fmt.Errorf("emitting delete file: %w", err)
			}
		}
	}
	return nil
}

func emitCreateOpsForDir(
	fs fs.FS,
	path *typesv1.Path,
	dir *SourceDir,
	emitCreate func(CreateOp) error,
) error {
	currentDirOp := createDirOp(path, dir)
	if err := emitCreate(currentDirOp); err != nil {
		return fmt.Errorf("emiting current dir: %w", err)
	}
	currentDir := typesv1.PathJoin(path, dir.Info.Name)
	// create files first, so that if we restart the process, we will
	// have entire directories to transfer at once, which will help expedite
	// the search
	for _, file := range dir.Files {
		op := createFileOp(fs, currentDir, file.Info)
		if err := emitCreate(op); err != nil {
			return fmt.Errorf("emiting opds for file: %w", err)
		}
	}
	for _, dir := range dir.Dirs {
		err := emitCreateOpsForDir(fs, currentDir, dir, emitCreate)
		if err != nil {
			return fmt.Errorf("emiting opds for subdir: %w", err)
		}
	}
	return nil
}

func sinkHasDirNamed(sink *typesv1.DirSum, name string) (*typesv1.DirSum, bool) {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(sink.Dirs, func(a, b *typesv1.DirSum) int {
		return strings.Compare(a.Info.Name, b.Info.Name)
	}))
	i, found := slices.BinarySearchFunc(sink.Dirs, name, func(dir *typesv1.DirSum, name string) int {
		return strings.Compare(dir.Info.Name, name)
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
		return strings.Compare(a.Info.Name, b.Info.Name)
	}))
	_, found := slices.BinarySearchFunc(src.Dirs, name, func(dir *SourceDir, name string) int {
		return strings.Compare(dir.Info.Name, name)
	})
	return found
}

func srcHasFileNamed(src *SourceDir, name string) bool {
	// relies on the fact that entries are sorted
	assert("must be sorted", slices.IsSortedFunc(src.Files, func(a, b *SourceFile) int {
		return strings.Compare(a.Info.Name, b.Info.Name)
	}))
	_, found := slices.BinarySearchFunc(src.Files, name, func(file *SourceFile, name string) int {
		return strings.Compare(file.Info.Name, name)
	})
	return found
}

func createDirOp(path *typesv1.Path, dir *SourceDir) CreateOp {
	return CreateOp{
		ParentDir: path,
		FileInfo:  dir.Info,
	}
}

func deleteDirOp(path *typesv1.Path, sink *typesv1.DirSum) DeleteOp {
	return DeleteOp{
		Path: typesv1.PathJoin(path, sink.Info.Name),
	}
}

func patchDirOp(path *typesv1.Path, dirname string, diff *DirPatchOp) PatchOp {
	return PatchOp{
		Path: typesv1.PathJoin(path, dirname),
		Dir:  diff,
	}
}

func makeDirDiff(src *SourceDir, sink *typesv1.DirSum) *DirPatchOp {
	if !proto.Equal(src.Info, sink.Info) {
		return &DirPatchOp{}
	}
	return nil
}

func createFileOp(fs fs.FS, path *typesv1.Path, fi *typesv1.FileInfo) CreateOp {
	op := CreateOp{
		ParentDir: path,
		FileInfo:  fi,
	}
	p := typesv1.StringFromPath(typesv1.PathJoin(path, fi.Name))
	if f, err := fs.Open(p); err != nil {
		println(err)
	} else {
		f.Close()
	}
	return op
}

func deleteFileOp(path *typesv1.Path, sink *typesv1.FileSum) DeleteOp {
	return DeleteOp{
		Path: typesv1.PathJoin(path, sink.Info.Name),
	}
}

func patchFileOp(path *typesv1.Path, diff *FilePatchOp) PatchOp {
	return PatchOp{
		Path: path,
		File: diff,
	}
}

func makeFileDiff(ctx context.Context, fs fs.FS, path *typesv1.Path, src *SourceFile, sink *typesv1.FileSum) (*FilePatchOp, error) {
	// compute mod time, size
	if !proto.Equal(src.Info, sink) {
		// obviously changed, we don't need to sum the content to figure as such
		return &FilePatchOp{
			Sum: sink,
		}, nil
	}
	filepath := typesv1.PathJoin(path, src.Info.Name)
	f, err := fs.Open(typesv1.StringFromPath(filepath))
	if err != nil {
		return nil, fmt.Errorf("opening source file: %w", err)
	}
	defer f.Close()

	matches, err := FileMatchesFileSum(ctx, sink, f, src.Info.Size)
	if err != nil {
		return nil, fmt.Errorf("opening source file: %w", err)
	}
	if !matches {
		return &FilePatchOp{Sum: sink}, nil
	}
	return nil, nil
}

func upload(ctx context.Context, src Source, sink Sink, createOp CreateOp) error {
	path := typesv1.StringFromPath(typesv1.PathJoin(createOp.ParentDir, createOp.FileInfo.Name))
	f, err := src.Open(path)
	if err != nil {
		_, _ = src.Open(path)
		return fmt.Errorf("opening %q on source for upload: %w", path, err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stating %q on source: %w", path, err)
	}

	err = sink.CreateFile(ctx, createOp.ParentDir, typesv1.FileInfoFromFS(fi), f)
	if err != nil {
		return fmt.Errorf("creating file on sink: %w", err)
	}
	return nil
}

func patch(ctx context.Context, src Source, sink Sink, patchOp PatchOp) error {
	if patchOp.Dir != nil {
		// patch a dir
		// panic("todo")
		return nil
	}

	path := typesv1.PathJoin(patchOp.Path, patchOp.File.Sum.Info.Name)
	fileDiff := patchOp.File

	f, err := src.Open(typesv1.StringFromPath(path))
	if err != nil {
		return fmt.Errorf("opening %q on source: %w", path, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stating %q on source: %w", path, err)
	}

	return sink.PatchFile(ctx, patchOp.Path, typesv1.FileInfoFromFS(fi), fileDiff.Sum, f)
}
