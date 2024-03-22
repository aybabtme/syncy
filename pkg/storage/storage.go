package storage

import (
	"context"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
)

type DB interface {
	Stat(context.Context, *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(context.Context, *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, blockSize uint32) (*typesv1.DirSum, error)
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
func (state *State) GetSignature(ctx context.Context, blockSize uint32) (*typesv1.DirSum, error) {
	return state.blob.GetSignature(ctx, blockSize)
}
