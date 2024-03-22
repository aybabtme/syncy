package storage

import (
	"context"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
)

type DB interface {
	GetRoot(context.Context) (*typesv1.Dir, error)
	Stat(context.Context, *typesv1.Path) (*typesv1.FileInfo, error)
	ListDir(context.Context, *typesv1.Path) ([]*typesv1.FileInfo, error)
	GetSignature(context.Context) (*typesv1.DirSum, error)
}

type State struct {
	meta metadb.Metadata
	blob blobdb.Blob
}

func NewState(meta metadb.Metadata, blob blobdb.Blob) *State {
	return &State{meta: meta, blob: blob}
}

func (state *State) GetRoot(ctx context.Context) (*typesv1.Dir, error) {
	panic("todo")
}
func (state *State) Stat(ctx context.Context, path *typesv1.Path) (*typesv1.FileInfo, error) {
	panic("todo")
}
func (state *State) ListDir(ctx context.Context, path *typesv1.Path) ([]*typesv1.FileInfo, error) {
	panic("todo")
}
func (state *State) GetSignature(ctx context.Context) (*typesv1.DirSum, error) {
	panic("todo")
}
