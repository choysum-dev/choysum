// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"context"
	"log/slog"
	"time"
)

// Record is one finished request for access logging.
type Record struct {
	Protocol string
	Method   string
	Path     string
	Code     string
	Duration time.Duration
	User     string
	Slow     bool
	Stream   string // "start" | "end" | ""
	Level    slog.Level
}

// LogRequest emits a single structured access line when the caller decides to log.
func LogRequest(ctx context.Context, logger *slog.Logger, rec Record) {
	if logger == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := []any{
		"protocol", rec.Protocol,
		"method", rec.Method,
		"code", rec.Code,
		"duration_ms", rec.Duration.Milliseconds(),
	}
	if rec.Path != "" {
		attrs = append(attrs, "path", rec.Path)
	}
	if rec.User != "" {
		attrs = append(attrs, "user", rec.User)
	}
	if rec.Slow {
		attrs = append(attrs, "slow", true)
	}
	if rec.Stream != "" {
		attrs = append(attrs, "stream", rec.Stream)
	}
	logger.Log(ctx, rec.Level, "request", attrs...)
}

// OptionsFromLogConfig maps config.Log fields into accesslog Options.
func OptionsFromLogConfig(access bool, accessSlowMs int) Options {
	return Options{Access: access, SlowMs: accessSlowMs}
}
