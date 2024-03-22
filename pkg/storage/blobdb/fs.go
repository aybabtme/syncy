package blobdb

import "io/fs"

type Blob interface {
}

var _ Blob = (*LocalFS)(nil)

type LocalFS struct {
	fs fs.FS
}

func NewLocalFS(fs fs.FS) *LocalFS {
	return &LocalFS{fs: fs}
}
