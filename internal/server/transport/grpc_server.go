// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import "google.golang.org/grpc"

type GRPCServerOptions struct {
	MaxRecvMsgSize     int
	MaxSendMsgSize     int
	ServerOptions      []grpc.ServerOption
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
}

func NewGRPCServer(opts GRPCServerOptions) *grpc.Server {
	serverOpts := make([]grpc.ServerOption, 0, len(opts.ServerOptions)+4)
	if opts.MaxRecvMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize))
	}
	if opts.MaxSendMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxSendMsgSize(opts.MaxSendMsgSize))
	}
	serverOpts = append(serverOpts, opts.ServerOptions...)
	if len(opts.UnaryInterceptors) > 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(opts.UnaryInterceptors...))
	}
	if len(opts.StreamInterceptors) > 0 {
		serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(opts.StreamInterceptors...))
	}
	return grpc.NewServer(serverOpts...)
}
