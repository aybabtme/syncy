// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: svc/sync00/v1/service.proto

package sync00v1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/aybabtme/syncy/pkg/gen/svc/sync00/v1"
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
	SyncServiceName = "svc.sync00.v1.SyncService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// SyncServiceCreateAccountProcedure is the fully-qualified name of the SyncService's CreateAccount
	// RPC.
	SyncServiceCreateAccountProcedure = "/svc.sync00.v1.SyncService/CreateAccount"
	// SyncServiceCreateProjectProcedure is the fully-qualified name of the SyncService's CreateProject
	// RPC.
	SyncServiceCreateProjectProcedure = "/svc.sync00.v1.SyncService/CreateProject"
	// SyncServiceStatProcedure is the fully-qualified name of the SyncService's Stat RPC.
	SyncServiceStatProcedure = "/svc.sync00.v1.SyncService/Stat"
	// SyncServiceListDirProcedure is the fully-qualified name of the SyncService's ListDir RPC.
	SyncServiceListDirProcedure = "/svc.sync00.v1.SyncService/ListDir"
	// SyncServiceGetSignatureProcedure is the fully-qualified name of the SyncService's GetSignature
	// RPC.
	SyncServiceGetSignatureProcedure = "/svc.sync00.v1.SyncService/GetSignature"
	// SyncServiceGetFileSumProcedure is the fully-qualified name of the SyncService's GetFileSum RPC.
	SyncServiceGetFileSumProcedure = "/svc.sync00.v1.SyncService/GetFileSum"
	// SyncServiceCreateProcedure is the fully-qualified name of the SyncService's Create RPC.
	SyncServiceCreateProcedure = "/svc.sync00.v1.SyncService/Create"
	// SyncServicePatchProcedure is the fully-qualified name of the SyncService's Patch RPC.
	SyncServicePatchProcedure = "/svc.sync00.v1.SyncService/Patch"
	// SyncServiceDeleteProcedure is the fully-qualified name of the SyncService's Delete RPC.
	SyncServiceDeleteProcedure = "/svc.sync00.v1.SyncService/Delete"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	syncServiceServiceDescriptor             = v1.File_svc_sync00_v1_service_proto.Services().ByName("SyncService")
	syncServiceCreateAccountMethodDescriptor = syncServiceServiceDescriptor.Methods().ByName("CreateAccount")
	syncServiceCreateProjectMethodDescriptor = syncServiceServiceDescriptor.Methods().ByName("CreateProject")
	syncServiceStatMethodDescriptor          = syncServiceServiceDescriptor.Methods().ByName("Stat")
	syncServiceListDirMethodDescriptor       = syncServiceServiceDescriptor.Methods().ByName("ListDir")
	syncServiceGetSignatureMethodDescriptor  = syncServiceServiceDescriptor.Methods().ByName("GetSignature")
	syncServiceGetFileSumMethodDescriptor    = syncServiceServiceDescriptor.Methods().ByName("GetFileSum")
	syncServiceCreateMethodDescriptor        = syncServiceServiceDescriptor.Methods().ByName("Create")
	syncServicePatchMethodDescriptor         = syncServiceServiceDescriptor.Methods().ByName("Patch")
	syncServiceDeleteMethodDescriptor        = syncServiceServiceDescriptor.Methods().ByName("Delete")
)

// SyncServiceClient is a client for the svc.sync00.v1.SyncService service.
type SyncServiceClient interface {
	// mgmt
	CreateAccount(context.Context, *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.CreateAccountResponse], error)
	CreateProject(context.Context, *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error)
	// info
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
	// sync
	// TODO: split in a separate service definition
	GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error)
	GetFileSum(context.Context, *connect.Request[v1.GetFileSumRequest]) (*connect.Response[v1.GetFileSumResponse], error)
	Create(context.Context) *connect.ClientStreamForClient[v1.CreateRequest, v1.CreateResponse]
	Patch(context.Context) *connect.ClientStreamForClient[v1.PatchRequest, v1.PatchResponse]
	Delete(context.Context, *connect.Request[v1.DeleteRequest]) (*connect.Response[v1.DeleteResponse], error)
}

// NewSyncServiceClient constructs a client for the svc.sync00.v1.SyncService service. By default,
// it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and
// sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC()
// or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewSyncServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) SyncServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &syncServiceClient{
		createAccount: connect.NewClient[v1.CreateAccountRequest, v1.CreateAccountResponse](
			httpClient,
			baseURL+SyncServiceCreateAccountProcedure,
			connect.WithSchema(syncServiceCreateAccountMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		createProject: connect.NewClient[v1.CreateProjectRequest, v1.CreateProjectResponse](
			httpClient,
			baseURL+SyncServiceCreateProjectProcedure,
			connect.WithSchema(syncServiceCreateProjectMethodDescriptor),
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
		getFileSum: connect.NewClient[v1.GetFileSumRequest, v1.GetFileSumResponse](
			httpClient,
			baseURL+SyncServiceGetFileSumProcedure,
			connect.WithSchema(syncServiceGetFileSumMethodDescriptor),
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
		delete: connect.NewClient[v1.DeleteRequest, v1.DeleteResponse](
			httpClient,
			baseURL+SyncServiceDeleteProcedure,
			connect.WithSchema(syncServiceDeleteMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// syncServiceClient implements SyncServiceClient.
type syncServiceClient struct {
	createAccount *connect.Client[v1.CreateAccountRequest, v1.CreateAccountResponse]
	createProject *connect.Client[v1.CreateProjectRequest, v1.CreateProjectResponse]
	stat          *connect.Client[v1.StatRequest, v1.StatResponse]
	listDir       *connect.Client[v1.ListDirRequest, v1.ListDirResponse]
	getSignature  *connect.Client[v1.GetSignatureRequest, v1.GetSignatureResponse]
	getFileSum    *connect.Client[v1.GetFileSumRequest, v1.GetFileSumResponse]
	create        *connect.Client[v1.CreateRequest, v1.CreateResponse]
	patch         *connect.Client[v1.PatchRequest, v1.PatchResponse]
	delete        *connect.Client[v1.DeleteRequest, v1.DeleteResponse]
}

// CreateAccount calls svc.sync00.v1.SyncService.CreateAccount.
func (c *syncServiceClient) CreateAccount(ctx context.Context, req *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.CreateAccountResponse], error) {
	return c.createAccount.CallUnary(ctx, req)
}

// CreateProject calls svc.sync00.v1.SyncService.CreateProject.
func (c *syncServiceClient) CreateProject(ctx context.Context, req *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error) {
	return c.createProject.CallUnary(ctx, req)
}

// Stat calls svc.sync00.v1.SyncService.Stat.
func (c *syncServiceClient) Stat(ctx context.Context, req *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	return c.stat.CallUnary(ctx, req)
}

// ListDir calls svc.sync00.v1.SyncService.ListDir.
func (c *syncServiceClient) ListDir(ctx context.Context, req *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	return c.listDir.CallUnary(ctx, req)
}

// GetSignature calls svc.sync00.v1.SyncService.GetSignature.
func (c *syncServiceClient) GetSignature(ctx context.Context, req *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error) {
	return c.getSignature.CallUnary(ctx, req)
}

// GetFileSum calls svc.sync00.v1.SyncService.GetFileSum.
func (c *syncServiceClient) GetFileSum(ctx context.Context, req *connect.Request[v1.GetFileSumRequest]) (*connect.Response[v1.GetFileSumResponse], error) {
	return c.getFileSum.CallUnary(ctx, req)
}

// Create calls svc.sync00.v1.SyncService.Create.
func (c *syncServiceClient) Create(ctx context.Context) *connect.ClientStreamForClient[v1.CreateRequest, v1.CreateResponse] {
	return c.create.CallClientStream(ctx)
}

// Patch calls svc.sync00.v1.SyncService.Patch.
func (c *syncServiceClient) Patch(ctx context.Context) *connect.ClientStreamForClient[v1.PatchRequest, v1.PatchResponse] {
	return c.patch.CallClientStream(ctx)
}

// Delete calls svc.sync00.v1.SyncService.Delete.
func (c *syncServiceClient) Delete(ctx context.Context, req *connect.Request[v1.DeleteRequest]) (*connect.Response[v1.DeleteResponse], error) {
	return c.delete.CallUnary(ctx, req)
}

// SyncServiceHandler is an implementation of the svc.sync00.v1.SyncService service.
type SyncServiceHandler interface {
	// mgmt
	CreateAccount(context.Context, *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.CreateAccountResponse], error)
	CreateProject(context.Context, *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error)
	// info
	Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error)
	ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error)
	// sync
	// TODO: split in a separate service definition
	GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error)
	GetFileSum(context.Context, *connect.Request[v1.GetFileSumRequest]) (*connect.Response[v1.GetFileSumResponse], error)
	Create(context.Context, *connect.ClientStream[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error)
	Patch(context.Context, *connect.ClientStream[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error)
	Delete(context.Context, *connect.Request[v1.DeleteRequest]) (*connect.Response[v1.DeleteResponse], error)
}

// NewSyncServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewSyncServiceHandler(svc SyncServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	syncServiceCreateAccountHandler := connect.NewUnaryHandler(
		SyncServiceCreateAccountProcedure,
		svc.CreateAccount,
		connect.WithSchema(syncServiceCreateAccountMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceCreateProjectHandler := connect.NewUnaryHandler(
		SyncServiceCreateProjectProcedure,
		svc.CreateProject,
		connect.WithSchema(syncServiceCreateProjectMethodDescriptor),
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
	syncServiceGetFileSumHandler := connect.NewUnaryHandler(
		SyncServiceGetFileSumProcedure,
		svc.GetFileSum,
		connect.WithSchema(syncServiceGetFileSumMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceCreateHandler := connect.NewClientStreamHandler(
		SyncServiceCreateProcedure,
		svc.Create,
		connect.WithSchema(syncServiceCreateMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServicePatchHandler := connect.NewClientStreamHandler(
		SyncServicePatchProcedure,
		svc.Patch,
		connect.WithSchema(syncServicePatchMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	syncServiceDeleteHandler := connect.NewUnaryHandler(
		SyncServiceDeleteProcedure,
		svc.Delete,
		connect.WithSchema(syncServiceDeleteMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/svc.sync00.v1.SyncService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case SyncServiceCreateAccountProcedure:
			syncServiceCreateAccountHandler.ServeHTTP(w, r)
		case SyncServiceCreateProjectProcedure:
			syncServiceCreateProjectHandler.ServeHTTP(w, r)
		case SyncServiceStatProcedure:
			syncServiceStatHandler.ServeHTTP(w, r)
		case SyncServiceListDirProcedure:
			syncServiceListDirHandler.ServeHTTP(w, r)
		case SyncServiceGetSignatureProcedure:
			syncServiceGetSignatureHandler.ServeHTTP(w, r)
		case SyncServiceGetFileSumProcedure:
			syncServiceGetFileSumHandler.ServeHTTP(w, r)
		case SyncServiceCreateProcedure:
			syncServiceCreateHandler.ServeHTTP(w, r)
		case SyncServicePatchProcedure:
			syncServicePatchHandler.ServeHTTP(w, r)
		case SyncServiceDeleteProcedure:
			syncServiceDeleteHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedSyncServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedSyncServiceHandler struct{}

func (UnimplementedSyncServiceHandler) CreateAccount(context.Context, *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.CreateAccountResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.CreateAccount is not implemented"))
}

func (UnimplementedSyncServiceHandler) CreateProject(context.Context, *connect.Request[v1.CreateProjectRequest]) (*connect.Response[v1.CreateProjectResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.CreateProject is not implemented"))
}

func (UnimplementedSyncServiceHandler) Stat(context.Context, *connect.Request[v1.StatRequest]) (*connect.Response[v1.StatResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.Stat is not implemented"))
}

func (UnimplementedSyncServiceHandler) ListDir(context.Context, *connect.Request[v1.ListDirRequest]) (*connect.Response[v1.ListDirResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.ListDir is not implemented"))
}

func (UnimplementedSyncServiceHandler) GetSignature(context.Context, *connect.Request[v1.GetSignatureRequest]) (*connect.Response[v1.GetSignatureResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.GetSignature is not implemented"))
}

func (UnimplementedSyncServiceHandler) GetFileSum(context.Context, *connect.Request[v1.GetFileSumRequest]) (*connect.Response[v1.GetFileSumResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.GetFileSum is not implemented"))
}

func (UnimplementedSyncServiceHandler) Create(context.Context, *connect.ClientStream[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.Create is not implemented"))
}

func (UnimplementedSyncServiceHandler) Patch(context.Context, *connect.ClientStream[v1.PatchRequest]) (*connect.Response[v1.PatchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.Patch is not implemented"))
}

func (UnimplementedSyncServiceHandler) Delete(context.Context, *connect.Request[v1.DeleteRequest]) (*connect.Response[v1.DeleteResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("svc.sync00.v1.SyncService.Delete is not implemented"))
}
