package metadb

import (
	"context"
	"database/sql"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	_ "github.com/go-sql-driver/mysql"
)

type Metadata interface {
	Stat(context.Context, *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(context.Context, *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, blockSize uint32) (*typesv1.DirSum, error)
}

var _ Metadata = (*MySQL)(nil)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(db *sql.DB) *MySQL {
	return &MySQL{db: db}
}

func (ms *MySQL) Stat(ctx context.Context, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	panic("todo")
}

func (ms *MySQL) ListDir(ctx context.Context, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	panic("todo")
}

func (ms *MySQL) GetSignature(ctx context.Context, blockSize uint32) (*typesv1.DirSum, error) {
	panic("todo")
}
