// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// StreamInterceptor logs stream establish/end boundaries per Options.
func StreamInterceptor(opts Options, logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ContextWithIdentityCapture(ss.Context())
		wrapped := &streamContextWrapper{ServerStream: ss, ctx: ctx}
		method := ""
		if info != nil {
			method = info.FullMethod
		}
		if opts.Access {
			LogRequest(ctx, logger, Record{
				Protocol: "grpc",
				Method:   method,
				Code:     codes.OK.String(),
				Duration: 0,
				User:     observedUserID(ctx),
				Stream:   "start",
				Level:    slog.LevelInfo,
			})
		}
		started := time.Now()
		err := handler(srv, wrapped)
		duration := time.Since(started)
		code := CodeFromError(err)
		if code == codes.Canceled {
			return err
		}
		decision := ClassifyGRPC(opts, code, duration)
		if opts.Access {
			level := slog.LevelInfo
			if code != codes.OK {
				level = grpcFailureLevel(code)
			} else if decision.Slow {
				level = slog.LevelWarn
			}
			LogRequest(ctx, logger, Record{
				Protocol: "grpc",
				Method:   method,
				Code:     code.String(),
				Duration: duration,
				User:     observedUserID(ctx),
				Slow:     decision.Slow,
				Stream:   "end",
				Level:    level,
			})
			return err
		}
		if decision.Log {
			LogRequest(ctx, logger, Record{
				Protocol: "grpc",
				Method:   method,
				Code:     code.String(),
				Duration: duration,
				User:     observedUserID(ctx),
				Slow:     decision.Slow,
				Stream:   "end",
				Level:    decision.Level,
			})
		}
		return err
	}
}

type streamContextWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *streamContextWrapper) Context() context.Context {
	return w.ctx
}
