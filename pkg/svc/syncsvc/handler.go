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

func (hdl *Handler) CreateAccount(ctx context.Context, req *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.CreateAccountResponse], error) {
	ll := hdl.ll.WithGroup("CreateAccount")
	ll.InfoContext(ctx, "received req CreateAccount")
	defer ll.InfoContext(ctx, "done req CreateAccount")
	if req.Msg.AccountName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing account name"))
	}
	publicID, err := hdl.db.CreateAccount(ctx, req.Msg.AccountName)
	if err != nil {
		ll.ErrorContext(ctx, "creating account in DB", slog.String("err", err.Error()))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}
	return connect.NewResponse(&v1.CreateAccountResponse{AccountId: publicID}), nil
}
func (hdl *Handler) CreateProject(ctx context.Context, req *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error) {
	ll := hdl.ll.WithGroup("CreateProject")
	ll.InfoContext(ctx, "received CreateProject req")
	defer ll.InfoContext(ctx, "done CreateProject")
	if req.Msg.AccountId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing account ID"))
	}
	if req.Msg.ProjectName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing project name"))
	}
	publicID, err := hdl.db.CreateProject(ctx, req.Msg.AccountId, req.Msg.ProjectName)
	if err != nil {
		ll.ErrorContext(ctx, "creating project in DB", slog.String("err", err.Error()))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}
	return connect.NewResponse(&v1.CreateProjectResponse{ProjectId: publicID}), nil
}

func (hdl *Handler) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	ll := hdl.ll.WithGroup("Stat")
	ll.InfoContext(ctx, "received Stat req",
		slog.Any("path", req.Msg.GetPath()),
	)
	defer ll.InfoContext(ctx, "done Stat")
	// TODO: validate this
	accountPubID, projectID := req.Msg.GetMeta().AccountId, req.Msg.GetMeta().ProjectId
	fi, ok, err := hdl.db.Stat(ctx, accountPubID, projectID, req.Msg.GetPath())
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
	ll.InfoContext(ctx, "received ListDir req")
	defer ll.InfoContext(ctx, "done ListDir")

	// TODO: validate this
	accountPubID, projectID := req.Msg.GetMeta().AccountId, req.Msg.GetMeta().ProjectId
	dirEntries, ok, err := hdl.db.ListDir(ctx, accountPubID, projectID, req.Msg.GetPath())
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
	ll.InfoContext(ctx, "received GetSignature req")
	defer ll.InfoContext(ctx, "done GetSignature")

	accountPubID, projectID := req.Msg.GetMeta().AccountId, req.Msg.GetMeta().ProjectId
	sig, err := hdl.db.GetSignature(ctx, accountPubID, projectID)
	if err != nil {
		ll.ErrorContext(ctx, "getting signature from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}

	return connect.NewResponse(&v1.GetSignatureResponse{
		Root: sig,
	}), nil
}

func (hdl *Handler) GetFileSum(ctx context.Context, req *connect.Request[v1.GetFileSumRequest]) (*connect.Response[v1.GetFileSumResponse], error) {
	ll := hdl.ll.WithGroup("GetFileSum")
	ll.InfoContext(ctx, "received GetFileSum req")
	defer ll.InfoContext(ctx, "done GetFileSum")

	accountPubID, projectID := req.Msg.GetMeta().AccountId, req.Msg.GetMeta().ProjectId
	sig, ok, err := hdl.db.GetFileSum(ctx, accountPubID, projectID, req.Msg.Path)
	if err != nil {
		ll.ErrorContext(ctx, "getting filesum from DB", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("try again later"))
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such file"))
	}

	return connect.NewResponse(&v1.GetFileSumResponse{
		Sum: sig,
	}), nil
}

func (hdl *Handler) Create(ctx context.Context, stream *connect.ClientStream[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	ll := hdl.ll.WithGroup("Create")
	ll.InfoContext(ctx, "received Create req")
	defer ll.InfoContext(ctx, "done Create")

	conn := stream.Conn()

	var req v1.CreateRequest
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
	accountPubID, projectID := req.GetMeta().AccountId, req.GetMeta().ProjectId
	err := hdl.db.CreatePath(ctx, accountPubID, projectID, creating.Path, creating.Info, func(w io.Writer) (blake3_64_256_sum []byte, _ error) {
		tgt := io.MultiWriter(w, h)
		var err error
		for {
			if err = conn.Receive(&req); err != nil {
				ll.Error("receiving step message", slog.Any("err", err))
				return nil, err
			}
			switch step := req.Step.(type) {
			case *v1.CreateRequest_Writing_:
				ll.InfoContext(ctx, "writing file block")
				_, err = tgt.Write(step.Writing.ContentBlock)
				if err != nil {
					ll.Error("writing content to target", slog.Any("err", err))
					return nil, connect.NewError(connect.CodeInternal, errors.New("unable to write to target"))
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
					return nil, connect.NewError(
						connect.CodeFailedPrecondition,
						fmt.Errorf("sent content hashsum of %x but requester announced a sum of %x", gotSum, wantSum),
					)
				}
				return gotSum, nil
			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expecting message of type `writing` or `closing`"))
			}
		}
	})
	if err != nil {
		ll.Error("creating path", slog.Any("err", err))
		return nil, fmt.Errorf("failed to create path")
	}
	return connect.NewResponse(&v1.CreateResponse{}), nil
}

func (hdl *Handler) Patch(ctx context.Context, stream *connect.ClientStream[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	ll := hdl.ll.WithGroup("Patch")
	ll.InfoContext(ctx, "received Patch req")
	defer ll.InfoContext(ctx, "done Patch")

	conn := stream.Conn()

	var req v1.PatchRequest
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
	accountPubID, projectID := req.GetMeta().AccountId, req.GetMeta().ProjectId
	err := hdl.db.PatchPath(ctx, accountPubID, projectID, opening.Path, opening.Info, opening.Sum, func(orig io.ReadSeeker, w io.Writer) (blake3_64_256_sum []byte, _ error) {
		tgt := io.MultiWriter(w, h)

		patcher := dirsync.NewFilePatcher(orig, tgt, opening.Sum)

		var err error
		for {
			if err = conn.Receive(&req); err != nil {
				ll.Error("receiving step message", slog.Any("err", err))
				return nil, err
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
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expecting patch of type `block_id` or `data`"))
				}
				if err != nil {
					ll.Error("writing content to target", slog.Any("err", err))
					return nil, connect.NewError(connect.CodeInternal, errors.New("unable to write to target"))
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
					return nil, connect.NewError(
						connect.CodeFailedPrecondition,
						fmt.Errorf("sent content creates a file with hashsum of %x but requester announced a sum of %x", gotSum, wantSum),
					)
				}
				return gotSum, nil

			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expecting message of type `writing` or `closing`"))
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
	ll.InfoContext(ctx, "received Deletes req")
	defer ll.InfoContext(ctx, "done Deletes")

	accountPubID, projectID := req.Msg.GetMeta().AccountId, req.Msg.GetMeta().ProjectId
	if err := hdl.db.DeletePaths(ctx, accountPubID, projectID, req.Msg.Paths); err != nil {
		ll.ErrorContext(ctx, "couldn't delete paths", slog.Any("err", err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to delete paths"))
	}

	return connect.NewResponse(&v1.DeletesResponse{}), nil
}
