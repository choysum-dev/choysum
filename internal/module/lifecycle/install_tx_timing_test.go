// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"encoding/json"
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
		`"level":"ERROR"`,
		`"msg":"uninstall commit transaction hold failed"`,
		`"error":"commit failed"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
}

func TestLogModuleCommitTxHoldPassesErrorValueForStack(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	// Capture attrs via a thin handler so we can assert the error is not stringified.
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelError,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "error" {
				if a.Value.Kind() != slog.KindAny {
					t.Fatalf("error attr must remain KindAny for stack expansion, got kind=%v", a.Value.Kind())
				}
				errVal, ok := a.Value.Any().(error)
				if !ok || errVal == nil || !strings.Contains(errVal.Error(), "commit failed") {
					t.Fatalf("error attr must remain the original error value, got %#v", a.Value.Any())
				}
			}
			return a
		},
	}))
	errSample := errors.New("commit failed")
	logModuleCommitTxHold(logger, "install", "module_commit", 10*time.Millisecond, errSample)
	if !strings.Contains(logBuf.String(), "commit failed") {
		t.Fatalf("expected error text in logs, got %q", logBuf.String())
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
	if !strings.Contains(logs, `"level":"ERROR"`) {
		t.Fatalf("expected ERROR level, got %q", logs)
	}
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
	for _, line := range strings.Split(logs, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var payload struct {
			DurationMs int64 `json:"duration_ms"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("unmarshal log line %q: %v", line, err)
		}
		if payload.DurationMs > 0 || strings.Contains(line, `"duration_ms"`) {
			return payload.DurationMs
		}
	}
	t.Fatalf("duration_ms missing in %q", logs)
	return 0
}
