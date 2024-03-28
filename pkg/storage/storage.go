package storage

import (
	"context"
	"fmt"
	"regexp"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
)

var (
	ErrAccountDoesntExist   = metadb.ErrAccountDoesntExist
	ErrProjectDoesntExist   = metadb.ErrProjectDoesntExist
	ErrParentDirDoesntExist = metadb.ErrParentDirDoesntExist
)

type DB interface {
	CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error)
	CreateProject(ctx context.Context, accountPublicID, projectName string) (projectPublicID string, err error)
	Stat(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, accountPublicID, projectPublicID string) (*typesv1.DirSum, error)
	GetFileSum(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileSum, bool, error)
	CreatePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, fn blobdb.CreateFunc) error
	PatchPath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn blobdb.PatchFunc) error
	DeletePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) error
}

var _ DB = (*State)(nil)

type State struct {
	// metadata such as FileMode (permissions), ModTime, are saved here
	meta metadb.Metadata
	// only dumb data is saved here, permissions/mode/times are not replicated
	// for safety reason
	blob blobdb.Blob
}

func NewState(meta metadb.Metadata, blob blobdb.Blob) *State {
	return &State{meta: meta, blob: blob}
}

var (
	AccountNameRegexp = regexp.MustCompile(`[a-zA-Z0-9-+_]+`)
	ProjectNameRegexp = regexp.MustCompile(`[a-zA-Z0-9-+_]+`)
)

func (state *State) CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error) {
	if !AccountNameRegexp.MatchString(accountName) {
		return "", fmt.Errorf("invalid account name: doesn't match regexp %s", AccountNameRegexp.String())
	}
	return state.meta.CreateAccount(ctx, accountName)
}

func (state *State) CreateProject(ctx context.Context, accountPublicID, projectName string) (projectPublicID string, err error) {
	if !ProjectNameRegexp.MatchString(projectName) {
		return "", fmt.Errorf("invalid project name: doesn't match regexp %s", ProjectNameRegexp.String())
	}
	return state.meta.CreateProject(ctx, accountPublicID, projectName, func(path string) error {
		return state.blob.CreateProjectRootPath(ctx, path)
	})
}

func (state *State) Stat(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	return state.meta.Stat(ctx, accountPublicID, projectPublicID, path)
}

func (state *State) ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	return state.meta.ListDir(ctx, accountPublicID, projectPublicID, path)
}

func (state *State) GetSignature(ctx context.Context, accountPublicID, projectPublicID string) (*typesv1.DirSum, error) {
	return state.meta.GetSignature(ctx, accountPublicID, projectPublicID, func(projectDir, filename string, fi *typesv1.FileInfo) (*typesv1.FileSum, bool, error) {
		return state.blob.GetFileSum(ctx, projectDir, filename, fi)
	})
}

func (state *State) GetFileSum(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileSum, bool, error) {
	return state.meta.GetFileSum(ctx, accountPublicID, projectPublicID, path, func(projectDir, filename string, fi *typesv1.FileInfo) (*typesv1.FileSum, bool, error) {
		return state.blob.GetFileSum(ctx, projectDir, filename, fi)
	})
}

func (state *State) CreatePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, fn blobdb.CreateFunc) error {
	// todo: do it in a transaction for safe rollback in case of mid-flight failure
	// todo: store all the FileInfo and full Path in metadata, only store filename + data in blobs
	return state.meta.CreatePathTx(ctx, accountPublicID, projectPublicID, path, fi, func(projectDir, filename string) (blake3_64_256_sum []byte, err error) {
		return state.blob.CreatePath(ctx, projectDir, filename, fi.IsDir, fn)
	})
}

func (state *State) PatchPath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn blobdb.PatchFunc) error {
	// lookup by path, fi, and sum.
	// sum in particular ensures that we're not using the wrong file as the original (one with new or stale data)
	return state.meta.PatchPathTx(ctx, accountPublicID, projectPublicID, path, fi, sum, func(projectDir, filename string) (blake3_64_256_sum []byte, err error) {
		return state.blob.PatchPath(ctx, projectDir, filename, fi.IsDir, sum, fn)
	})
}

func (state *State) DeletePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) error {
	return state.meta.DeletePath(ctx, accountPublicID, projectPublicID, path, func(projectDir, filename string) error {
		return state.blob.DeletePath(ctx, projectDir, filename)
	})
}
