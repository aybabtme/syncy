// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: svc/sync/v1/service.proto

package syncv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// SyncServiceName is the fully-qualified name of the SyncService service.
	SyncServiceName = "svc.sync.v1.SyncService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// SyncServiceGetRootProcedure is the fully-qualified name of the SyncService's GetRoot RPC.
	SyncServiceGetRootProcedure = "/svc.sync.v1.SyncService/GetRoot"
	// SyncServiceStatProcedure is the fully-qualified name of the SyncService's Stat RPC.
	SyncServiceStatProcedure = "/svc.sync.v1.SyncService/Stat"
	// SyncServiceListDirProcedure is the fully-qualified name of the SyncService's ListDir RPC.
	SyncServiceListDirProcedure = "/svc.sync.v1.SyncService/ListDir"
	// SyncServiceGetSignatureProcedure is the fully-qualified name of the SyncService's GetSignature
	// RPC.
	SyncServiceGetSignatureProcedure = "/svc.sync.v1.SyncService/GetSignature"
	// SyncServiceCreateProcedure is the fully-qualified name of the SyncService's Create RPC.
	SyncServiceCreateProcedure = "/svc.sync.v1.SyncService/Create"
	// SyncServicePatchProcedure is the fully-qualified name of the SyncService's Patch RPC.
	SyncServicePatchProcedure = "/svc.sync.v1.SyncService/Patch"
	// SyncServiceDeletesProcedure is the fully-qualified name of the SyncService's Deletes RPC.
	SyncServiceDeletesProcedure = "/svc.sync.v1.SyncService/Deletes"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	syncServiceServiceDescriptor            = v1.File_svc_sync_v1_service_proto.Services().ByName("SyncService")
	syncServiceGetRootMethodDescriptor      = syncServiceServiceDescriptor.Methods().ByName("GetRoot")
	syncServiceStatMethodDescriptor         = syncServiceServiceDescriptor.Methods().ByName("Stat")
	syncServiceListDirMethodDescriptor      = syncServiceServiceDescriptor.Methods().ByName("ListDir")
	syncServiceGetSignatureMethodDescriptor = syncServiceServiceDescriptor.Methods().ByName("GetSignature")
	syncServiceCreateMethodDescriptor       = syncServiceServiceDescriptor.Methods().ByName("Create")
	syncServicePatchMethodDescriptor        = syncServiceServiceDescriptor.Methods().ByName("Patch")
	syncServiceDeletesMethodDescriptor      = syncServiceServiceDescriptor.Methods().ByName("Deletes")
)

// SyncServiceClient is a client for the svc.sync.v1.SyncService service.
type SyncServiceClient interface {
	// info
	GetRoot(context.Context, *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error)
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
	// sync
	// TODO: split in a separate service definition
	GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error)
	Create(context.Context, *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error)
	Patch(context.Context, *connect.Request[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error)
	Deletes(context.Context, *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error)
}

// NewSyncServiceClient constructs a client for the svc.sync.v1.SyncService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewSyncServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) SyncServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &syncServiceClient{
		getRoot: connect.NewClient[v1.GetRootRequest, v1.GetRootResponse](
			httpClient,
			baseURL+SyncServiceGetRootProcedure,
			connect.WithSchema(syncServiceGetRootMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		stat: connect.NewClient[v1.StatRequest, v1.StatResponse](
			httpClient,
			baseURL+SyncServiceStatProcedure,
			connect.WithSchema(syncServiceStatMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		listDir: connect.NewClient[v1.ListDirRequest, v1.ListDirResponse](
			httpClient,
			baseURL+SyncServiceListDirProcedure,
			connect.WithSchema(syncServiceListDirMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getSignature: connect.NewClient[v1.GetSignatureRequest, v1.GetSignatureResponse](
			httpClient,
			baseURL+SyncServiceGetSignatureProcedure,
			connect.WithSchema(syncServiceGetSignatureMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		create: connect.NewClient[v1.CreateRequest, v1.CreateResponse](
			httpClient,
			baseURL+SyncServiceCreateProcedure,
			connect.WithSchema(syncServiceCreateMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		patch: connect.NewClient[v1.PatchRequest, v1.PatchResponse](
			httpClient,
			baseURL+SyncServicePatchProcedure,
			connect.WithSchema(syncServicePatchMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		deletes: connect.NewClient[v1.DeletesRequest, v1.DeletesResponse](
			httpClient,
			baseURL+SyncServiceDeletesProcedure,
			connect.WithSchema(syncServiceDeletesMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// syncServiceClient implements SyncServiceClient.
type syncServiceClient struct {
	getRoot      *connect.Client[v1.GetRootRequest, v1.GetRootResponse]
	stat         *connect.Client[v1.StatRequest, v1.StatResponse]
	listDir      *connect.Client[v1.ListDirRequest, v1.ListDirResponse]
	getSignature *connect.Client[v1.GetSignatureRequest, v1.GetSignatureResponse]
	create       *connect.Client[v1.CreateRequest, v1.CreateResponse]
	patch        *connect.Client[v1.PatchRequest, v1.PatchResponse]
	deletes      *connect.Client[v1.DeletesRequest, v1.DeletesResponse]
}

// GetRoot calls svc.sync.v1.SyncService.GetRoot.
func (c *syncServiceClient) GetRoot(ctx context.Context, req *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error) {
	return c.getRoot.CallUnary(ctx, req)
}

// Stat calls svc.sync.v1.SyncService.Stat.
func (c *syncServiceClient) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	return c.stat.CallUnary(ctx, req)
}

// ListDir calls svc.sync.v1.SyncService.ListDir.
func (c *syncServiceClient) ListDir(ctx context.Context, req *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	return c.listDir.CallUnary(ctx, req)
}

// GetSignature calls svc.sync.v1.SyncService.GetSignature.
func (c *syncServiceClient) GetSignature(ctx context.Context, req *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error) {
	return c.getSignature.CallUnary(ctx, req)
}

// Create calls svc.sync.v1.SyncService.Create.
func (c *syncServiceClient) Create(ctx context.Context, req *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	return c.create.CallUnary(ctx, req)
}

// Patch calls svc.sync.v1.SyncService.Patch.
func (c *syncServiceClient) Patch(ctx context.Context, req *connect.Request[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	return c.patch.CallUnary(ctx, req)
}

// Deletes calls svc.sync.v1.SyncService.Deletes.
func (c *syncServiceClient) Deletes(ctx context.Context, req *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error) {
	return c.deletes.CallUnary(ctx, req)
}

// SyncServiceHandler is an implementation of the svc.sync.v1.SyncService service.
type SyncServiceHandler interface {
	// info
	GetRoot(context.Context, *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error)
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
	// sync
	// TODO: split in a separate service definition
	GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error)
	Create(context.Context, *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error)
	Patch(context.Context, *connect.Request[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error)
	Deletes(context.Context, *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error)
}

// NewSyncServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewSyncServiceHandler(svc SyncServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	syncServiceGetRootHandler := connect.NewUnaryHandler(
		SyncServiceGetRootProcedure,
		svc.GetRoot,
		connect.WithSchema(syncServiceGetRootMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceStatHandler := connect.NewUnaryHandler(
		SyncServiceStatProcedure,
		svc.Stat,
		connect.WithSchema(syncServiceStatMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceListDirHandler := connect.NewUnaryHandler(
		SyncServiceListDirProcedure,
		svc.ListDir,
		connect.WithSchema(syncServiceListDirMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceGetSignatureHandler := connect.NewUnaryHandler(
		SyncServiceGetSignatureProcedure,
		svc.GetSignature,
		connect.WithSchema(syncServiceGetSignatureMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceCreateHandler := connect.NewUnaryHandler(
		SyncServiceCreateProcedure,
		svc.Create,
		connect.WithSchema(syncServiceCreateMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServicePatchHandler := connect.NewUnaryHandler(
		SyncServicePatchProcedure,
		svc.Patch,
		connect.WithSchema(syncServicePatchMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceDeletesHandler := connect.NewUnaryHandler(
		SyncServiceDeletesProcedure,
		svc.Deletes,
		connect.WithSchema(syncServiceDeletesMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/svc.sync.v1.SyncService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case SyncServiceGetRootProcedure:
			syncServiceGetRootHandler.ServeHTTP(w, r)
		case SyncServiceStatProcedure:
			syncServiceStatHandler.ServeHTTP(w, r)
		case SyncServiceListDirProcedure:
			syncServiceListDirHandler.ServeHTTP(w, r)
		case SyncServiceGetSignatureProcedure:
			syncServiceGetSignatureHandler.ServeHTTP(w, r)
		case SyncServiceCreateProcedure:
			syncServiceCreateHandler.ServeHTTP(w, r)
		case SyncServicePatchProcedure:
			syncServicePatchHandler.ServeHTTP(w, r)
		case SyncServiceDeletesProcedure:
			syncServiceDeletesHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedSyncServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedSyncServiceHandler struct{}

func (UnimplementedSyncServiceHandler) GetRoot(context.Context, *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.GetRoot is not implemented"))
}

func (UnimplementedSyncServiceHandler) Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.Stat is not implemented"))
}

func (UnimplementedSyncServiceHandler) ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.ListDir is not implemented"))
}

func (UnimplementedSyncServiceHandler) GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.GetSignature is not implemented"))
}

func (UnimplementedSyncServiceHandler) Create(context.Context, *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.Create is not implemented"))
}

func (UnimplementedSyncServiceHandler) Patch(context.Context, *connect.Request[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.Patch is not implemented"))
}

func (UnimplementedSyncServiceHandler) Deletes(context.Context, *connect.Request[v1.DeletesRequest]) (*connect.Response[v1.DeletesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync.v1.SyncService.Deletes is not implemented"))
}
