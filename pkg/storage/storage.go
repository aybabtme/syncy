package storage

import (
	"context"
	"io"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
)

type DB interface {
	Stat(context.Context, *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(context.Context, *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context) (*typesv1.DirSum, error)

	CreatePath(ctx context.Context, path *typesv1.Path, fi *typesv1.FileInfo, fn func(w io.Writer) error) error
	PatchPath(ctx context.Context, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn func(orig io.ReadSeeker, w io.Writer) error) error
	DeletePaths(ctx context.Context, paths []*typesv1.Path) error
}

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

func (state *State) Stat(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	return state.meta.Stat(ctx, accountPublicID, projectName, path)
}
func (state *State) ListDir(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	return state.meta.ListDir(ctx, accountPublicID, projectName, path)
}
func (state *State) GetSignature(ctx context.Context, accountPublicID, projectName string) (*typesv1.DirSum, error) {
	return state.meta.GetSignature(ctx, accountPublicID, projectName)
}

func (state *State) CreatePath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, fn func(w io.Writer) error) error {
	// todo: do it in a transaction for safe rollback in case of mid-flight failure
	// todo: store all the FileInfo and full Path in metadata, only store filename + data in blobs
	return state.meta.CreatePathTx(ctx, path, fi, func() error {
		state.blob.CreatePath(ctx, accountID, projectID)
	})
}

func (state *State) PatchPath(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn func(orig io.ReadSeeker, w io.Writer) error) error {
	// lookup by path, fi, and sum.
	// sum in particular ensures that we're not using the wrong file as the original (one with new or stale data)
	panic("todo")
}

func (state *State) DeletePaths(ctx context.Context, accountPublicID, projectName string, paths []*typesv1.Path) error {
	panic("todo")
}
