// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"context"

	"google.golang.org/grpc"
)

type contextKey int

const (
	serviceDialerKey contextKey = iota
)

// ServiceDialer dials a logical gRPC service by service name (e.g. "auth.User")
// and returns a shared ClientConn managed by the host server.
//
// Implementations must be safe for concurrent use.
//
// Strict mode: callers should treat missing dialer as a configuration/runtime
// precondition failure.
//
// The returned connection must not be closed by the caller.
// Its lifecycle is managed by the host server.
type ServiceDialer func(ctx context.Context, serviceName string) (*grpc.ClientConn, error)

func ContextWithServiceDialer(ctx context.Context, dialer ServiceDialer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if dialer == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceDialerKey, dialer)
}

func ServiceDialerFromContext(ctx context.Context) (ServiceDialer, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(serviceDialerKey)
	d, ok := v.(ServiceDialer)
	return d, ok && d != nil
}
