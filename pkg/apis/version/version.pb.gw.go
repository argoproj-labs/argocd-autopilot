// Code generated by protoc-gen-grpc-gateway. DO NOT EDIT.
// source: pkg/apis/version/version.proto

/*
Package version is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package version

import (
	"context"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Suppress "imported and not used" errors
var _ codes.Code
var _ io.Reader
var _ status.Status
var _ = runtime.String
var _ = utilities.NewDoubleArray
var _ = metadata.Join

func request_Version_Version_0(ctx context.Context, marshaler runtime.Marshaler, client VersionClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq emptypb.Empty
	var metadata runtime.ServerMetadata

	msg, err := client.Version(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func local_request_Version_Version_0(ctx context.Context, marshaler runtime.Marshaler, server VersionServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq emptypb.Empty
	var metadata runtime.ServerMetadata

	msg, err := server.Version(ctx, &protoReq)
	return msg, metadata, err

}

// RegisterVersionHandlerServer registers the http handlers for service Version to "mux".
// UnaryRPC     :call VersionServer directly.
// StreamingRPC :currently unsupported pending https://github.com/grpc/grpc-go/issues/906.
// Note that using this registration option will cause many gRPC library features to stop working. Consider using RegisterVersionHandlerFromEndpoint instead.
func RegisterVersionHandlerServer(ctx context.Context, mux *runtime.ServeMux, server VersionServer) error {

	mux.Handle("GET", pattern_Version_Version_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/github.com.argoproj.argocd_autopilot.pkg.apis.version.Version/Version")
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_Version_Version_0(rctx, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Version_Version_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

// RegisterVersionHandlerFromEndpoint is same as RegisterVersionHandler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterVersionHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterVersionHandler(ctx, mux, conn)
}

// RegisterVersionHandler registers the http handlers for service Version to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterVersionHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterVersionHandlerClient(ctx, mux, NewVersionClient(conn))
}

// RegisterVersionHandlerClient registers the http handlers for service Version
// to "mux". The handlers forward requests to the grpc endpoint over the given implementation of "VersionClient".
// Note: the gRPC framework executes interceptors within the gRPC handler. If the passed in "VersionClient"
// doesn't go through the normal gRPC flow (creating a gRPC client etc.) then it will be up to the passed in
// "VersionClient" to call the correct interceptors.
func RegisterVersionHandlerClient(ctx context.Context, mux *runtime.ServeMux, client VersionClient) error {

	mux.Handle("GET", pattern_Version_Version_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/github.com.argoproj.argocd_autopilot.pkg.apis.version.Version/Version")
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_Version_Version_0(rctx, inboundMarshaler, client, req, pathParams)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_Version_Version_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

var (
	pattern_Version_Version_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"api", "version"}, ""))
)

var (
	forward_Version_Version_0 = runtime.ForwardResponseMessage
)
