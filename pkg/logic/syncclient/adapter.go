package syncclient

import (
	"context"
	"fmt"
	"io"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"lukechampine.com/blake3"

	"github.com/aybabtme/syncy/pkg/logic/dirsync"
)

var _ dirsync.Sink = (*Sink)(nil)

type Sink struct {
	client          syncv1connect.SyncServiceClient
	createBlockSize uint
	meta            *typesv1.ReqMeta
}

const minCreateBlockSize = 10 * 1 << 10

func ClientAdapter(client syncv1connect.SyncServiceClient, meta *typesv1.ReqMeta, createBlockSize uint) (*Sink, error) {
	if createBlockSize < minCreateBlockSize {
		return nil, fmt.Errorf("block size must be at least %d", minCreateBlockSize)
	}
	return &Sink{client: client, createBlockSize: createBlockSize, meta: meta}, nil
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
	success := false
	stream := sk.client.Create(ctx)
	defer func() {
		if !success {
			_, _ = stream.CloseAndReceive()
		}
	}()

	blockSize := sk.createBlockSize
	if blockSize > uint(fi.Size) {
		blockSize = uint(fi.Size)
	}

	hasher := syncv1.Hasher_blake3_64_256
	h := blake3.New(64, nil)
	r = io.TeeReader(r, h)

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
	if err := stream.Send(creating); err != nil {
		return fmt.Errorf("creating file on sink: %w", err)
	}

	writingStep := &syncv1.CreateRequest_Writing{}
	writing := &syncv1.CreateRequest{
		Step: &syncv1.CreateRequest_Writing_{
			Writing: writingStep,
		},
	}
	buf := make([]byte, blockSize)
	more := true
	for more {
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
			if err := stream.Send(writing); err != nil {
				return fmt.Errorf("writing file on sink: %w", err)
			}
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
	if err := stream.Send(closing); err != nil {
		return fmt.Errorf("closing file sink: %w", err)
	}
	success = true

	res, err := stream.CloseAndReceive()
	if err != nil {
		return fmt.Errorf("closing stream: %w", err)
	}
	_ = res

	return err
}

func (sk *Sink) DeleteFiles(ctx context.Context, ops []dirsync.DeleteOp) error {
	_, err := sk.client.Deletes(ctx, connect.NewRequest(&syncv1.DeletesRequest{Meta: sk.meta}))
	return err
}

func (sk *Sink) PatchFile(ctx context.Context, op dirsync.PatchOp) error {
	panic("todo")
	// _, err := sk.client.Patch(ctx, connect.NewRequest(&syncv1.PatchRequest{}))
	// return err
}
