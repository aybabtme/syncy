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
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	syncServiceServiceDescriptor       = v1.File_svc_sync_v1_service_proto.Services().ByName("SyncService")
	syncServiceGetRootMethodDescriptor = syncServiceServiceDescriptor.Methods().ByName("GetRoot")
	syncServiceStatMethodDescriptor    = syncServiceServiceDescriptor.Methods().ByName("Stat")
	syncServiceListDirMethodDescriptor = syncServiceServiceDescriptor.Methods().ByName("ListDir")
)

// SyncServiceClient is a client for the svc.sync.v1.SyncService service.
type SyncServiceClient interface {
	GetRoot(context.Context, *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error)
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
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
	}
}

// syncServiceClient implements SyncServiceClient.
type syncServiceClient struct {
	getRoot *connect.Client[v1.GetRootRequest, v1.GetRootResponse]
	stat    *connect.Client[v1.StatRequest, v1.StatResponse]
	listDir *connect.Client[v1.ListDirRequest, v1.ListDirResponse]
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

// SyncServiceHandler is an implementation of the svc.sync.v1.SyncService service.
type SyncServiceHandler interface {
	GetRoot(context.Context, *connect.Request[v1.GetRootRequest]) (*connect.Response[v1.GetRootResponse], error)
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
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
	return "/svc.sync.v1.SyncService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case SyncServiceGetRootProcedure:
			syncServiceGetRootHandler.ServeHTTP(w, r)
		case SyncServiceStatProcedure:
			syncServiceStatHandler.ServeHTTP(w, r)
		case SyncServiceListDirProcedure:
			syncServiceListDirHandler.ServeHTTP(w, r)
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
