package controlplaneclient

import (
	"context"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

var (
	grpcOpts = []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
)

// TODO: add timeout?
func call[Request any, Response any](ctx context.Context, md metadata.MD, fn func(context.Context, Request, ...grpc.CallOption) (Response, error), req Request) (Response, error) {
	return fn(metadata.NewOutgoingContext(ctx, md), req, grpcOpts...)
}
