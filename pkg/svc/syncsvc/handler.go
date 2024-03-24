package syncsvc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"

	"connectrpc.com/connect"
	v1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	"github.com/aybabtme/syncy/pkg/storage"
	"lukechampine.com/blake3"
)

type Handler struct {
	ll *slog.Logger

	db storage.DB
}

var _ syncv1connect.SyncServiceHandler = (*Handler)(nil)

func NewHandler(ll *slog.Logger, db storage.DB) *Handler {
	return &Handler{ll: ll, db: db}
}

func (hdl *Handler) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	ll := hdl.ll.WithGroup("Stat")
	ll.InfoContext(ctx, "received req")

	fi, ok, err := hdl.db.Stat(ctx, req.Msg.GetPath())
	if err != nil {
		ll.ErrorContext(ctx, "getting stat from DB", slog.String("err", err.Error()))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such file"))
	}

	return connect.NewResponse(&v1.StatResponse{
		Info: fi,
	}), nil
}

func (hdl *Handler) ListDir(ctx context.Context, req *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	ll := hdl.ll.WithGroup("ListDir")
	ll.InfoContext(ctx, "received req")

	dirEntries, ok, err := hdl.db.ListDir(ctx, req.Msg.GetPath())
	if err != nil {
		ll.ErrorContext(ctx, "getting listdir from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such directory"))
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

func (hdl *Handler) Create(ctx context.Context, stream *connect.ClientStream[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	ll := hdl.ll.WithGroup("Create")
	ll.InfoContext(ctx, "received req")

	conn := stream.Conn()

	req := new(v1.CreateRequest)
	if err := conn.Receive(&req); err != nil {
		ll.Error("receiving step message `creating`", slog.Any("err", err))
		return nil, err
	}
	creating := req.GetCreating()
	if creating == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("first message should be of type `creating`"))
	}
	var h hash.Hash
	switch creating.Hasher {
	case v1.Hasher_blake3_64_256:
		h = blake3.New(64, nil)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown hasher: %s", creating.Hasher.String()))
	}
	ll.InfoContext(ctx, "creating path")
	err := hdl.db.CreatePath(ctx, creating.Path, creating.Info, func(w io.Writer) error {
		tgt := io.MultiWriter(w, h)
		var err error
		for {
			if err = conn.Receive(&req); err != nil {
				ll.Error("receiving step message", slog.Any("err", err))
				return err
			}
			switch step := req.Step.(type) {
			case *v1.CreateRequest_Writing_:
				ll.InfoContext(ctx, "writing file block")
				_, err = tgt.Write(step.Writing.ContentBlock)
				if err != nil {
					ll.Error("writing content to target", slog.Any("err", err))
					return connect.NewError(connect.CodeInternal, errors.New("unable to write to target"))
				}
			case *v1.CreateRequest_Closing_:
				ll.InfoContext(ctx, "closing file")
				gotSum := h.Sum(nil)
				wantSum := req.GetClosing().Sum
				if !bytes.Equal(gotSum, wantSum) {
					ll.ErrorContext(ctx, "hashsum mismatch",
						slog.String("want", hex.EncodeToString(wantSum)),
						slog.String("got", hex.EncodeToString(wantSum)),
					)
					return connect.NewError(
						connect.CodeFailedPrecondition,
						fmt.Errorf("sent content hashsum of %x but requester announced a sum of %x", gotSum, wantSum),
					)
				}
				return nil
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.New("expecting message of type `writing` or `closing`"))
			}
		}
	})
	if err != nil {
		ll.Error("creating path", slog.Any("err", err))
		return nil, err
	}
	return connect.NewResponse(&v1.CreateResponse{}), nil
}

func (hdl *Handler) Patch(ctx context.Context, stream *connect.ClientStream[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	ll := hdl.ll.WithGroup("Patch")
	ll.InfoContext(ctx, "received req")

	conn := stream.Conn()

	req := new(v1.PatchRequest)
	if err := conn.Receive(&req); err != nil {
		ll.Error("receiving step message `opening`", slog.Any("err", err))
		return nil, err
	}
	opening := req.GetOpening()
	if opening == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("first message should be of type `opening`"))
	}
	var h hash.Hash
	switch opening.Hasher {
	case v1.Hasher_blake3_64_256:
		h = blake3.New(64, nil)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown hasher: %s", opening.Hasher.String()))
	}
	ll.InfoContext(ctx, "opening path for patching")
	err := hdl.db.PatchPath(ctx, opening.Path, opening.Info, opening.Sum, func(orig io.ReadSeeker, w io.Writer) error {
		tgt := io.MultiWriter(w, h)

		patcher := dirsync.NewFilePatcher(orig, tgt, opening.Sum)

		var err error
		for {
			if err = conn.Receive(&req); err != nil {
				ll.Error("receiving step message", slog.Any("err", err))
				return err
			}
			switch step := req.Step.(type) {
			case *v1.PatchRequest_Patching_:
				ll.InfoContext(ctx, "applying patch")
				switch p := step.Patching.Patch.Patch.(type) {
				case *typesv1.FileBlockPatch_BlockId:
					_, err = patcher.WriteBlock(p.BlockId)
				case *typesv1.FileBlockPatch_Data:
					_, err = patcher.WriteData(p.Data)
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.New("expecting patch of type `block_id` or `data`"))
				}
				if err != nil {
					ll.Error("writing content to target", slog.Any("err", err))
					return connect.NewError(connect.CodeInternal, errors.New("unable to write to target"))
				}

			case *v1.PatchRequest_Closing_:
				ll.InfoContext(ctx, "closing file")
				gotSum := h.Sum(nil)
				wantSum := req.GetClosing().Sum
				if !bytes.Equal(gotSum, wantSum) {
					ll.ErrorContext(ctx, "hashsum mismatch",
						slog.String("want", hex.EncodeToString(wantSum)),
						slog.String("got", hex.EncodeToString(wantSum)),
					)
					return connect.NewError(
						connect.CodeFailedPrecondition,
						fmt.Errorf("sent content creates a file with hashsum of %x but requester announced a sum of %x", gotSum, wantSum),
					)
				}
				return nil

			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.New("expecting message of type `writing` or `closing`"))
			}

		}
	})
	if err != nil {
		ll.Error("patching path", slog.Any("err", err))
		return nil, err
	}
	return connect.NewResponse(&v1.PatchResponse{}), nil
}

func (hdl *Handler) Deletes(ctx context.Context, req *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error) {
	ll := hdl.ll.WithGroup("Deletes")
	ll.InfoContext(ctx, "received req")

	if err := hdl.db.DeletePaths(ctx, req.Msg.Paths); err != nil {
		ll.ErrorContext(ctx, "couldn't delete paths", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to delete paths"))
	}

	return connect.NewResponse(&v1.DeletesResponse{}), nil
}
