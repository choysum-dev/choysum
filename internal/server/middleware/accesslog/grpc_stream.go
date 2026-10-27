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
// Start lines omit user (auth runs inside); end lines include the captured user.
// When Access is on, start/end use INFO (including client cancel) so the pair is
// visible at the default info log level.
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
				Stream:   "start",
				Level:    slog.LevelInfo,
			})
		}
		started := time.Now()
		err := handler(srv, wrapped)
		duration := time.Since(started)
		code := CodeFromError(err)
		decision := ClassifyGRPC(opts, code, duration)
		if !decision.Log && code == codes.Canceled && opts.Access {
			decision = Decision{Log: true, Level: slog.LevelInfo}
		}
		if !decision.Log {
			return err
		}
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
		return err
	}
}

type streamContextWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *streamContextWrapper) Context() context.Context {
	return s.ctx
}
