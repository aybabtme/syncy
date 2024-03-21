package syncsvc

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	v1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
)

type Handler struct {
	ll *slog.Logger
}

var _ syncv1connect.SyncServiceHandler = (*Handler)(nil)

func NewHandler(ll *slog.Logger) *Handler {
	return &Handler{ll: ll}
}

func (hdl *Handler) GetRoot(ctx context.Context, req *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error) {
	ll := hdl.ll.WithGroup("GetRoot")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: GetRoot"))
}
func (hdl *Handler) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	ll := hdl.ll.WithGroup("Stat")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: Stat"))
}
func (hdl *Handler) ListDir(ctx context.Context, req *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	ll := hdl.ll.WithGroup("ListDir")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: ListDir"))
}
