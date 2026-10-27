// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestQuietProgressFromContext(t *testing.T) {
	t.Parallel()
	if QuietProgressFromContext(nil) || QuietProgressFromContext(context.Background()) {
		t.Fatal("expected quiet=false without WithQuietProgress")
	}
	ctx := WithQuietProgress(nil)
	if !QuietProgressFromContext(ctx) {
		t.Fatal("expected quiet=true after WithQuietProgress")
	}
}

func TestProgressDetailLogLevel_QuietAndDefault(t *testing.T) {
	t.Parallel()
	if got := ProgressDetailLogLevel(context.Background()); got != slog.LevelInfo {
		t.Fatalf("default detail level = %v, want Info", got)
	}
	if got := ProgressDetailLogLevel(WithQuietProgress(context.Background())); got != slog.LevelDebug {
		t.Fatalf("quiet detail level = %v, want Debug", got)
	}
	// Non-TTY ProgressLine must not demote (CLI pipes / CI).
	line := NewProgressLine(&bytes.Buffer{})
	ctx := WithProgressLine(context.Background(), line)
	if line.IsTTY() {
		t.Fatal("buffer ProgressLine must not report TTY")
	}
	if got := ProgressDetailLogLevel(ctx); got != slog.LevelInfo {
		t.Fatalf("non-TTY detail level = %v, want Info", got)
	}
}

func TestProgressDetailLogLevel_TTYProgressLine(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)
	line := NewProgressLine(&bytes.Buffer{})
	if !line.IsTTY() {
		t.Fatal("expected TTY ProgressLine")
	}
	if got := ProgressDetailLogLevel(WithProgressLine(context.Background(), line)); got != slog.LevelDebug {
		t.Fatalf("TTY detail level = %v, want Debug", got)
	}
}

func TestProgressBoundaryLogLevel(t *testing.T) {
	t.Parallel()
	if got := ProgressBoundaryLogLevel(context.Background()); got != slog.LevelInfo {
		t.Fatalf("default boundary = %v, want Info", got)
	}
	if got := ProgressBoundaryLogLevel(WithQuietProgress(context.Background())); got != slog.LevelDebug {
		t.Fatalf("quiet boundary = %v, want Debug", got)
	}
}
