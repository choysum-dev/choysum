// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Options configures request access logging.
type Options struct {
	Access bool
	SlowMs int // 0 disables slow WARN
}

// Decision is the outcome of Classify.
type Decision struct {
	Log   bool
	Level slog.Level
	Slow  bool
}

// ClassifyGRPC decides whether/how to log a finished gRPC call.
func ClassifyGRPC(opts Options, code codes.Code, duration time.Duration) Decision {
	if code == codes.Canceled {
		return Decision{}
	}
	if code == codes.OK {
		if opts.SlowMs > 0 && duration >= time.Duration(opts.SlowMs)*time.Millisecond {
			return Decision{Log: true, Level: slog.LevelWarn, Slow: true}
		}
		if opts.Access {
			return Decision{Log: true, Level: slog.LevelInfo}
		}
		return Decision{}
	}
	return Decision{Log: true, Level: grpcFailureLevel(code)}
}

// ClassifyHTTP decides whether/how to log a finished HTTP call.
func ClassifyHTTP(opts Options, statusCode int, duration time.Duration) Decision {
	if statusCode >= 500 {
		return Decision{Log: true, Level: slog.LevelError}
	}
	if statusCode >= 400 {
		return Decision{Log: true, Level: slog.LevelWarn}
	}
	if statusCode >= 200 && statusCode < 400 {
		if opts.SlowMs > 0 && duration >= time.Duration(opts.SlowMs)*time.Millisecond {
			return Decision{Log: true, Level: slog.LevelWarn, Slow: true}
		}
		if opts.Access {
			return Decision{Log: true, Level: slog.LevelInfo}
		}
	}
	return Decision{}
}

func grpcFailureLevel(code codes.Code) slog.Level {
	switch code {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
		codes.OutOfRange, codes.Aborted:
		return slog.LevelWarn
	case codes.DeadlineExceeded, codes.ResourceExhausted, codes.Internal,
		codes.Unavailable, codes.DataLoss, codes.Unknown:
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

// CodeFromError maps a gRPC handler error to a status code.
func CodeFromError(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	if st, ok := status.FromError(err); ok {
		return st.Code()
	}
	return codes.Unknown
}

// ShouldSkipHTTP reports paths that never emit access logs.
// Static skips are limited to /web/assets/* and /favicon.ico (not global extensions).
func ShouldSkipHTTP(rawPath string) bool {
	p := strings.TrimSpace(rawPath)
	if p == "" {
		return true
	}
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	switch p {
	case "/health", "/healthz", "/readyz", "/favicon.ico":
		return true
	}
	return strings.HasPrefix(p, "/web/assets/")
}
