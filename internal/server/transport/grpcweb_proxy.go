// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/mwitkow/grpc-proxy/proxy"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCWebProxyOptions struct {
	Registry       registry.Registry
	Address        string
	EnableTLS      bool
	TLSCaFile      string
	TLSServerName  string
	Logger         *slog.Logger
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

func NewGRPCWebProxy(opts GRPCWebProxyOptions) *grpcweb.WrappedGrpcServer {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	grpc.EnableTracing = true

	director := func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
		if strings.Contains(fullMethodName, ".TaskWorker/") {
			return nil, nil, status.Error(codes.Unimplemented, "TaskWorker is not available via gRPC-Web")
		}
		md, _ := metadata.FromIncomingContext(ctx)
		outCtx := ctx
		mdCopy := md.Copy()
		delete(mdCopy, "user-agent")
		delete(mdCopy, "connection")
		outCtx = metadata.NewOutgoingContext(outCtx, mdCopy)

		var backendCreds credentials.TransportCredentials
		var tlsErr error
		if opts.EnableTLS {
			backendCreds, tlsErr = credentials.NewClientTLSFromFile(opts.TLSCaFile, opts.TLSServerName)
			if tlsErr != nil {
				logger.Error("grpc-web proxy tls credentials creation failed", "error", tlsErr)
				os.Exit(1)
			}
		} else {
			backendCreds = insecure.NewCredentials()
		}

		var backendClientConn *grpc.ClientConn
		var dialErr error
		if opts.Registry != nil {
			target := strings.Split(fullMethodName, "/")[1]
			target = fmt.Sprintf("%s:///%s", opts.Registry.Scheme(), target)
			backendClientConn, dialErr = grpc.NewClient(target,
				grpc.WithTransportCredentials(backendCreds),
				grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
				grpc.WithResolvers(opts.Registry.Resolver()),
				grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			)
		} else {
			backendClientConn, dialErr = grpc.NewClient(opts.Address,
				grpc.WithTransportCredentials(backendCreds),
				grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
			)
		}
		if dialErr != nil {
			logger.Error("grpc-web proxy backend dial failed", "error", dialErr)
			os.Exit(1)
		}

		return outCtx, backendClientConn, nil
	}

	grpcServer := grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
		grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(opts.MaxSendMsgSize),
	)

	return grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithAllowedRequestHeaders([]string{
			"traceparent",
			"tracestate",
			"baggage",
			"authorization",
			"x-xsrf-token",
			"content-type",
		}),
	)
}
