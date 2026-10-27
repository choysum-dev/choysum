// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"context"
	"log/slog"
)

type quietProgressContextKey struct{}

// WithQuietProgress marks ctx so ProgressLine/spinner owns the user-visible
// timeline (hotreload nested upgrade). Progress-detail and nested boundary
// slog rows demote to Debug; failures and outer watch summaries stay Info.
func WithQuietProgress(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, quietProgressContextKey{}, true)
}

// QuietProgressFromContext reports whether WithQuietProgress was applied.
func QuietProgressFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	quiet, _ := ctx.Value(quietProgressContextKey{}).(bool)
	return quiet
}

// ProgressDetailLogLevel is Debug when QuietProgress is set or an interactive
// TTY ProgressLine is present; otherwise Info. Use for per-module / pipeline
// stage rows that duplicate spinner progress.
func ProgressDetailLogLevel(ctx context.Context) slog.Level {
	if QuietProgressFromContext(ctx) {
		return slog.LevelDebug
	}
	if line := ProgressLineFromContext(ctx); line != nil && line.IsTTY() {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// ProgressBoundaryLogLevel is Debug under QuietProgress only so nested
// hotreload upgrades stay quiet; CLI keeps plan/completed at Info.
func ProgressBoundaryLogLevel(ctx context.Context) slog.Level {
	if QuietProgressFromContext(ctx) {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
