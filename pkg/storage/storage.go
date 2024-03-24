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
	meta metadb.Metadata
	blob blobdb.Blob
}

func NewState(meta metadb.Metadata, blob blobdb.Blob) *State {
	return &State{meta: meta, blob: blob}
}

func (state *State) Stat(ctx context.Context, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	return state.blob.Stat(ctx, path)
	// return state.meta.Stat(ctx, path)
}
func (state *State) ListDir(ctx context.Context, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	return state.blob.ListDir(ctx, path)
	// return state.meta.ListDir(ctx, path)
}
func (state *State) GetSignature(ctx context.Context) (*typesv1.DirSum, error) {
	return state.blob.GetSignature(ctx)
}

func (state *State) CreatePath(ctx context.Context, path *typesv1.Path, fi *typesv1.FileInfo, fn func(w io.Writer) error) error {
	panic("todo")
}

func (state *State) PatchPath(ctx context.Context, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn func(orig io.ReadSeeker, w io.Writer) error) error {
	// lookup by path, fi, and sum.
	// sum in particular ensures that we're not using the wrong file as the original (one with new or stale data)
	panic("todo")
}

func (state *State) DeletePaths(ctx context.Context, paths []*typesv1.Path) error {
	panic("todo")
}
