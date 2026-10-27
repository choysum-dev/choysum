// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// UnaryInterceptor logs finished unary RPCs per Options.
func UnaryInterceptor(opts Options, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = ContextWithIdentityCapture(ctx)
		started := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(started)
		code := CodeFromError(err)
		decision := ClassifyGRPC(opts, code, duration)
		if !decision.Log {
			return resp, err
		}
		method := ""
		if info != nil {
			method = info.FullMethod
		}
		LogRequest(ctx, logger, Record{
			Protocol: "grpc",
			Method:   method,
			Code:     code.String(),
			Duration: duration,
			User:     observedUserID(ctx),
			Slow:     decision.Slow,
			Level:    decision.Level,
		})
		return resp, err
	}
}
