// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogModuleCommitTxHoldEmitsDuration(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logModuleCommitTxHold(logger, "upgrade", "module_commit", 150*time.Millisecond, nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"upgrade commit transaction hold completed"`,
		`"caller":"module_commit"`,
		`"op":"upgrade"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
	if !strings.Contains(logs, `"level":"DEBUG"`) {
		t.Fatalf("successful hold must be Debug, got %q", logs)
	}
	if ms := mustParseLogDurationMs(t, logs); ms != 150 {
		t.Fatalf("duration_ms=%d, want 150", ms)
	}
}

func TestLogModuleCommitTxHoldSuccessIsDebug(t *testing.T) {
	t.Parallel()
	var infoBuf bytes.Buffer
	infoLogger := slog.New(slog.NewJSONHandler(&infoBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logModuleCommitTxHold(infoLogger, "upgrade", "module_commit", 500*time.Millisecond, nil)
	if logs := infoBuf.String(); logs != "" {
		t.Fatalf("successful hold must not emit at Info, got %q", logs)
	}

	var debugBuf bytes.Buffer
	debugLogger := slog.New(slog.NewJSONHandler(&debugBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logModuleCommitTxHold(debugLogger, "upgrade", "module_commit", 500*time.Millisecond, nil)
	logs := debugBuf.String()
	if !strings.Contains(logs, `"level":"DEBUG"`) {
		t.Fatalf("expected DEBUG hold log, got %q", logs)
	}
	if ms := mustParseLogDurationMs(t, logs); ms != 500 {
		t.Fatalf("duration_ms=%d, want 500", ms)
	}
}

func TestLogModuleCommitTxHoldIncludesError(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	errSample := errors.New("commit failed")

	logModuleCommitTxHold(logger, "uninstall", "module_commit", 10*time.Millisecond, errSample)

	logs := logBuf.String()
	for _, want := range []string{
		`"level":"INFO"`,
		`"msg":"uninstall commit transaction hold completed"`,
		`"error":"commit failed"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
}

func TestLogInstallOuterTxHoldEmitsDuration(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Deterministic duration path (same helper LogInstallOuterTxHold wraps).
	logModuleCommitTxHold(logger, "install", "bootstrap", 150*time.Millisecond, nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"install commit transaction hold completed"`,
		`"op":"install"`,
		`"caller":"bootstrap"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
	if ms := mustParseLogDurationMs(t, logs); ms != 150 {
		t.Fatalf("duration_ms=%d, want 150", ms)
	}
}

func TestLogInstallOuterTxHoldDelegatesToModuleHold(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	LogInstallOuterTxHold(logger, "cli-wrap", time.Now(), nil)

	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"install commit transaction hold completed"`) {
		t.Fatalf("expected install hold message, got %q", logs)
	}
	if !strings.Contains(logs, `"caller":"cli-wrap"`) {
		t.Fatalf("expected caller=cli-wrap, got %q", logs)
	}
}

func TestLogInstallOuterTxHoldIncludesError(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	errSample := errors.New("outer failed")

	LogInstallOuterTxHold(logger, "cli", time.Now().Add(-10*time.Millisecond), errSample)

	logs := logBuf.String()
	if !strings.Contains(logs, `"error":"outer failed"`) {
		t.Fatalf("expected error attr, got %q", logs)
	}
}

func TestLogModuleCommitTxHoldZeroStartedNoOp(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	LogModuleCommitTxHold(logger, "upgrade", "module_commit", time.Time{}, nil)
	if logs := logBuf.String(); logs != "" {
		t.Fatalf("zero started must no-op, got %q", logs)
	}
}

func TestLogModuleCommitTxHoldNilLoggerNoOp(t *testing.T) {
	t.Parallel()
	LogModuleCommitTxHold(nil, "upgrade", "module_commit", time.Now(), nil)
	logModuleCommitTxHold(nil, "upgrade", "module_commit", time.Millisecond, nil)
}

func TestLogModuleCommitTxHoldDefaultOpAndCaller(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logModuleCommitTxHold(logger, "  ", " \t ", 10*time.Millisecond, nil)
	logs := buf.String()
	if !strings.Contains(logs, `"op":"module"`) || !strings.Contains(logs, `"caller":"unknown"`) {
		t.Fatalf("expected default op/caller, got %q", logs)
	}
}

func mustParseLogDurationMs(t *testing.T, logs string) int64 {
	t.Helper()
	const key = `"duration_ms":`
	idx := strings.Index(logs, key)
	if idx < 0 {
		t.Fatalf("duration_ms missing in %q", logs)
	}
	rest := logs[idx+len(key):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		t.Fatalf("duration_ms value missing in %q", logs)
	}
	var n int64
	for i := 0; i < end; i++ {
		n = n*10 + int64(rest[i]-'0')
	}
	return n
}
