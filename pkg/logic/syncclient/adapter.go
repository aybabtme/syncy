package syncclient

import (
	"context"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"

	"github.com/aybabtme/syncy/pkg/logic/dirsync"
)

var _ dirsync.Sink = (*Sink)(nil)

type Sink struct {
	client syncv1connect.SyncServiceClient
}

func ClientAdapter(client syncv1connect.SyncServiceClient) *Sink {
	return &Sink{client: client}
}

func (sk *Sink) GetSignatures(ctx context.Context) (*typesv1.DirSum, error) {
	res, err := sk.client.GetSignature(ctx, connect.NewRequest(&syncv1.GetSignatureRequest{}))
	if err != nil {
		return nil, err
	}
	return res.Msg.GetRoot(), nil
}

func (sk *Sink) CreateFile(ctx context.Context, op dirsync.CreateOp) error {
	_, err := sk.client.Create(ctx, connect.NewRequest(&syncv1.CreateRequest{}))
	return err
}

func (sk *Sink) DeleteFiles(ctx context.Context, ops []dirsync.DeleteOp) error {
	_, err := sk.client.Deletes(ctx, connect.NewRequest(&syncv1.DeletesRequest{}))
	return err
}

func (sk *Sink) PatchFile(ctx context.Context, op dirsync.PatchOp) error {
	_, err := sk.client.Patch(ctx, connect.NewRequest(&syncv1.PatchRequest{}))
	return err
}
