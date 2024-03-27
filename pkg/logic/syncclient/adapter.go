package syncclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"lukechampine.com/blake3"

	"github.com/aybabtme/syncy/pkg/logic/dirsync"
)

var _ dirsync.Sink = (*Sink)(nil)

type Sink struct {
	ll              *slog.Logger
	client          syncv1connect.SyncServiceClient
	createBlockSize uint
	meta            *typesv1.ReqMeta
}

const minCreateBlockSize = 10 * 1 << 10

func ClientAdapter(ll *slog.Logger, client syncv1connect.SyncServiceClient, meta *typesv1.ReqMeta, createBlockSize uint) (*Sink, error) {
	if createBlockSize < minCreateBlockSize {
		return nil, fmt.Errorf("block size must be at least %d", minCreateBlockSize)
	}
	return &Sink{ll: ll, client: client, createBlockSize: createBlockSize, meta: meta}, nil
}

func (sk *Sink) GetSignatures(ctx context.Context) (*typesv1.DirSum, error) {
	res, err := sk.client.GetSignature(ctx, connect.NewRequest(&syncv1.GetSignatureRequest{
		Meta: sk.meta,
	}))
	if err != nil {
		return nil, err
	}
	return res.Msg.GetRoot(), nil
}

func (sk *Sink) CreateFile(ctx context.Context, dir *typesv1.Path, fi *typesv1.FileInfo, r io.Reader) error {
	ll := sk.ll.With(
		slog.String("path", typesv1.StringFromPath(dir)),
		slog.String("file", fi.Name),
	)
	success := false
	ll.InfoContext(ctx, "creating file")
	stream := sk.client.Create(ctx)
	defer func() {
		if !success {
			ll.InfoContext(ctx, "failed, closing and receiving")
			_, _ = stream.CloseAndReceive()
			ll.InfoContext(ctx, "done closing and receiving")
		}
	}()

	hasher := syncv1.Hasher_blake3_64_256
	creating := &syncv1.CreateRequest{
		Meta: sk.meta,
		Step: &syncv1.CreateRequest_Creating_{
			Creating: &syncv1.CreateRequest_Creating{
				Path:   dir,
				Info:   fi,
				Hasher: hasher,
			},
		},
	}
	ll.InfoContext(ctx, "starting step create")
	if err := stream.Send(creating); err != nil {
		return fmt.Errorf("creating file on sink: %w", err)
	}
	ll.InfoContext(ctx, "done step create")

	if fi.IsDir {
		closing := &syncv1.CreateRequest{
			Step: &syncv1.CreateRequest_Closing_{Closing: &syncv1.CreateRequest_Closing{}},
		}
		ll.InfoContext(ctx, "starting step closing")
		if err := stream.Send(closing); err != nil {
			return fmt.Errorf("closing file sink: %w", err)
		}
		ll.InfoContext(ctx, "done step closing")
		success = true

		res, err := stream.CloseAndReceive()
		if err != nil {
			return fmt.Errorf("closing stream: %w", err)
		}
		_ = res
		return nil
	}

	blockSize := sk.createBlockSize
	if blockSize > uint(fi.Size) {
		blockSize = uint(fi.Size)
	}
	if blockSize == 0 {
		blockSize = 1024
	}
	h := blake3.New(64, nil)
	r = io.TeeReader(r, h)

	writingStep := &syncv1.CreateRequest_Writing{}
	writing := &syncv1.CreateRequest{
		Step: &syncv1.CreateRequest_Writing_{
			Writing: writingStep,
		},
	}
	buf := make([]byte, blockSize)
	more := true
	for more {
		ll.InfoContext(ctx, "reading data", slog.Uint64("blocksize", uint64(blockSize)))
		n, err := io.ReadFull(r, buf)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			more = false
		case nil:
			// continue
		default:
			return fmt.Errorf("reading file on source: %w", err)
		}
		if n > 0 {
			writingStep.ContentBlock = buf[:n]
			ll.InfoContext(ctx, "starting step writing")
			if err := stream.Send(writing); err != nil {
				return fmt.Errorf("writing file on sink: %w", err)
			}
			ll.InfoContext(ctx, "done step writing")
		}
	}

	sum := h.Sum(nil)
	closing := &syncv1.CreateRequest{
		Step: &syncv1.CreateRequest_Closing_{
			Closing: &syncv1.CreateRequest_Closing{
				Sum: sum,
			},
		},
	}
	ll.InfoContext(ctx, "starting step closing")
	if err := stream.Send(closing); err != nil {
		return fmt.Errorf("closing file sink: %w", err)
	}
	ll.InfoContext(ctx, "done step closing")
	success = true

	ll.InfoContext(ctx, "success, closing and receiving")
	res, err := stream.CloseAndReceive()
	if err != nil {
		return fmt.Errorf("closing stream: %w", err)
	}
	_ = res
	ll.InfoContext(ctx, "done closing and receiving")

	return err
}

func (sk *Sink) PatchFile(ctx context.Context, dir *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, r io.Reader) error {
	ll := sk.ll.With(
		slog.String("path", typesv1.StringFromPath(dir)),
		slog.String("file", fi.Name),
	)
	success := false
	ll.InfoContext(ctx, "patching file")
	stream := sk.client.Patch(ctx)
	defer func() {
		if !success {
			ll.InfoContext(ctx, "failed, closing and receiving")
			_, _ = stream.CloseAndReceive()
			ll.InfoContext(ctx, "done closing and receiving")
		}
	}()

	hasher := syncv1.Hasher_blake3_64_256
	opening := &syncv1.PatchRequest{
		Meta: sk.meta,
		Step: &syncv1.PatchRequest_Opening_{
			Opening: &syncv1.PatchRequest_Opening{
				Path:   dir,
				Info:   fi,
				Hasher: hasher,
				Sum:    sum,
			},
		},
	}
	ll.InfoContext(ctx, "starting step open")
	if err := stream.Send(opening); err != nil {
		return fmt.Errorf("opening file on sink: %w", err)
	}
	ll.InfoContext(ctx, "done step open")

	if fi.IsDir {
		closing := &syncv1.PatchRequest{
			Step: &syncv1.PatchRequest_Closing_{Closing: &syncv1.PatchRequest_Closing{}},
		}
		ll.InfoContext(ctx, "starting step closing")
		if err := stream.Send(closing); err != nil {
			return fmt.Errorf("closing file sink: %w", err)
		}
		ll.InfoContext(ctx, "done step closing")
		success = true

		res, err := stream.CloseAndReceive()
		if err != nil {
			return fmt.Errorf("closing stream: %w", err)
		}
		_ = res
		return nil
	}

	blockSize := sk.createBlockSize
	if blockSize > uint(fi.Size) {
		blockSize = uint(fi.Size)
	}
	if blockSize == 0 {
		blockSize = 1024
	}
	h := blake3.New(64, nil)
	r = io.TeeReader(r, h)

	patch := &typesv1.FileBlockPatch{Patch: nil}
	dataPatch := &typesv1.FileBlockPatch_Data{}
	blockIDPatch := &typesv1.FileBlockPatch_BlockId{}
	patching := &syncv1.PatchRequest{
		Step: &syncv1.PatchRequest_Patching_{
			Patching: &syncv1.PatchRequest_Patching{
				Patch: patch,
			},
		},
	}

	_, err := dirsync.Rsync(ctx, r, sum,
		func(b []byte) (int, error) {
			ll.InfoContext(ctx, "starting block data patching")
			dataPatch.Data = b
			patch.Patch = dataPatch
			err := stream.Send(patching)
			ll.InfoContext(ctx, "done block data patching")
			return len(b), err
		},
		func(u uint32) (int, error) {
			ll.InfoContext(ctx, "starting block id patching")
			blockIDPatch.BlockId = u
			patch.Patch = blockIDPatch
			err := stream.Send(patching)
			ll.InfoContext(ctx, "done block id patching")
			return 4, err
		},
	)
	if err != nil {
		return fmt.Errorf("sending block patches: %w", err)
	}

	blake3_64_256_sum := h.Sum(nil)
	closing := &syncv1.PatchRequest{
		Step: &syncv1.PatchRequest_Closing_{
			Closing: &syncv1.PatchRequest_Closing{
				Sum: blake3_64_256_sum,
			},
		},
	}
	ll.InfoContext(ctx, "starting step closing")
	if err := stream.Send(closing); err != nil {
		return fmt.Errorf("closing file sink: %w", err)
	}
	ll.InfoContext(ctx, "done step closing")
	success = true

	ll.InfoContext(ctx, "success, closing and receiving")
	res, err := stream.CloseAndReceive()
	if err != nil {
		return fmt.Errorf("closing stream: %w", err)
	}
	_ = res
	ll.InfoContext(ctx, "done closing and receiving")

	return err
}

func (sk *Sink) DeleteFiles(ctx context.Context, ops []dirsync.DeleteOp) error {
	_, err := sk.client.Deletes(ctx, connect.NewRequest(&syncv1.DeletesRequest{Meta: sk.meta}))
	return err
}
