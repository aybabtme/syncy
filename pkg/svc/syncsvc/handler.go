package syncsvc

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	v1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	"github.com/aybabtme/syncy/pkg/storage"
)

type Handler struct {
	ll *slog.Logger

	db storage.DB
}

var _ syncv1connect.SyncServiceHandler = (*Handler)(nil)

func NewHandler(ll *slog.Logger, db storage.DB) *Handler {
	return &Handler{ll: ll, db: db}
}

func (hdl *Handler) GetRoot(ctx context.Context, req *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error) {
	ll := hdl.ll.WithGroup("GetRoot")
	ll.InfoContext(ctx, "received req")

	root, err := hdl.db.GetRoot(ctx)
	if err != nil {
		ll.ErrorContext(ctx, "getting root from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}

	return connect.NewResponse(&v1.GetRootResponse{
		Root: root,
	}), nil
}

func (hdl *Handler) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	ll := hdl.ll.WithGroup("Stat")
	ll.InfoContext(ctx, "received req")

	fi, err := hdl.db.Stat(ctx, req.Msg.GetPath())
	if err != nil {
		ll.ErrorContext(ctx, "getting stat from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}

	return connect.NewResponse(&v1.StatResponse{
		Info: fi,
	}), nil
}

func (hdl *Handler) ListDir(ctx context.Context, req *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	ll := hdl.ll.WithGroup("ListDir")
	ll.InfoContext(ctx, "received req")

	dirEntries, err := hdl.db.ListDir(ctx, req.Msg.GetPath())
	if err != nil {
		ll.ErrorContext(ctx, "getting listdir from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}

	return connect.NewResponse(&v1.ListDirResponse{
		DirEntries: dirEntries,
	}), nil
}

func (hdl *Handler) GetSignature(ctx context.Context, req *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error) {
	ll := hdl.ll.WithGroup("GetSignature")
	ll.InfoContext(ctx, "received req")

	sig, err := hdl.db.GetSignature(ctx)
	if err != nil {
		ll.ErrorContext(ctx, "getting signature from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}

	return connect.NewResponse(&v1.GetSignatureResponse{
		Root: sig,
	}), nil
}

func (hdl *Handler) Create(ctx context.Context, req *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	ll := hdl.ll.WithGroup("Create")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: Create"))
}

func (hdl *Handler) Patch(ctx context.Context, req *connect.Request[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	ll := hdl.ll.WithGroup("Patch")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: Patch"))
}

func (hdl *Handler) Deletes(ctx context.Context, req *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error) {
	ll := hdl.ll.WithGroup("Deletes")
	ll.InfoContext(ctx, "received req")
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("todo: Deletes"))
}
