package storage

import (
	"context"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
)

type DB interface {
	CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error)
	CreateProject(ctx context.Context, accountPublicID, projectName string) error
	Stat(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, accountPublicID, projectName string) (*typesv1.DirSum, error)
	CreatePath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, fn blobdb.CreateFunc) error
	PatchPath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn blobdb.PatchFunc) error
	DeletePaths(ctx context.Context, accountPublicID, projectName string, paths []*typesv1.Path) error
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

func (state *State) CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error) {
	return state.meta.CreateAccount(ctx, accountName)
}
func (state *State) CreateProject(ctx context.Context, accountPublicID, projectName string) error {
	return state.meta.CreateProject(ctx, accountPublicID, projectName)
}

func (state *State) Stat(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	return state.meta.Stat(ctx, accountPublicID, projectName, path)
}
func (state *State) ListDir(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	return state.meta.ListDir(ctx, accountPublicID, projectName, path)
}
func (state *State) GetSignature(ctx context.Context, accountPublicID, projectName string) (*typesv1.DirSum, error) {
	return state.meta.GetSignature(ctx, accountPublicID, projectName)
}

func (state *State) CreatePath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, fn blobdb.CreateFunc) error {
	// todo: do it in a transaction for safe rollback in case of mid-flight failure
	// todo: store all the FileInfo and full Path in metadata, only store filename + data in blobs
	return state.meta.CreatePathTx(ctx, accountPublicID, projectName, path, fi, func(filename string) (blake3_64_256_sum []byte, err error) {
		return state.blob.CreatePath(ctx, filename, fi.IsDir, fn)
	})
}

func (state *State) PatchPath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn blobdb.PatchFunc) error {
	// lookup by path, fi, and sum.
	// sum in particular ensures that we're not using the wrong file as the original (one with new or stale data)
	return state.meta.PatchPathTx(ctx, accountPublicID, projectName, path, fi, sum, func(filename string) (blake3_64_256_sum []byte, err error) {
		return state.blob.PatchPath(ctx, filename, fi.IsDir, sum, fn)
	})
}

func (state *State) DeletePaths(ctx context.Context, accountPublicID, projectName string, paths []*typesv1.Path) error {
	panic("todo")
}
