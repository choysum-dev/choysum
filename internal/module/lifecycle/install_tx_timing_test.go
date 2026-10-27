// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogModuleCommitTxHoldEmitsDuration(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	logModuleCommitTxHold(logger, "upgrade", "module_commit", 150*time.Millisecond, nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"upgrade commit transaction hold completed"`,
		`"op":"upgrade"`,
		`"caller":"module_commit"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
	ms := mustParseLogDurationMs(t, logs)
	if ms != 150 {
		t.Fatalf("duration_ms=%d, want 150", ms)
	}
}

func TestLogModuleCommitTxHoldShortHoldIsDebug(t *testing.T) {
	t.Parallel()
	short := moduleCommitTxHoldInfoMinDuration / 2

	var infoBuf bytes.Buffer
	infoLogger := slog.New(slog.NewJSONHandler(&infoBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logModuleCommitTxHold(infoLogger, "upgrade", "module_commit", short, nil)
	if infoBuf.Len() != 0 {
		t.Fatalf("short hold must stay Debug at Info level, got %q", infoBuf.String())
	}

	var debugBuf bytes.Buffer
	debugLogger := slog.New(slog.NewJSONHandler(&debugBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logModuleCommitTxHold(debugLogger, "upgrade", "module_commit", short, nil)
	logs := debugBuf.String()
	if !strings.Contains(logs, `"level":"DEBUG"`) {
		t.Fatalf("expected DEBUG short hold, got %q", logs)
	}
	if !strings.Contains(logs, `"msg":"upgrade commit transaction hold completed"`) {
		t.Fatalf("expected hold message, got %q", logs)
	}
	if ms := mustParseLogDurationMs(t, logs); ms != short.Milliseconds() {
		t.Fatalf("duration_ms=%d, want %d", ms, short.Milliseconds())
	}
}

func TestLogModuleCommitTxHoldIncludesError(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	logModuleCommitTxHold(logger, "uninstall", "module_commit", 10*time.Millisecond, errSample)

	logs := logBuf.String()
	if !strings.Contains(logs, `"op":"uninstall"`) {
		t.Fatalf("expected uninstall op, got %q", logs)
	}
	if !strings.Contains(logs, `"error":"boom"`) {
		t.Fatalf("expected error attr, got %q", logs)
	}
	if !strings.Contains(logs, `"level":"INFO"`) {
		t.Fatalf("error hold must stay Info, got %q", logs)
	}
}

func TestLogInstallOuterTxHoldEmitsDuration(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Deterministic duration path (same helper LogInstallOuterTxHold wraps).
	logModuleCommitTxHold(logger, "install", "bootstrap", 150*time.Millisecond, nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"install commit transaction hold completed"`,
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
	// Debug so the public wrapper smoke check is not coupled to the Info duration floor.
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	LogInstallOuterTxHold(logger, "cli-wrap", time.Now(), nil)
	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"install commit transaction hold completed"`) {
		t.Fatalf("expected install hold message, got %q", logs)
	}
	if !strings.Contains(logs, `"caller":"cli-wrap"`) {
		t.Fatalf("expected caller, got %q", logs)
	}
	if !strings.Contains(logs, `"op":"install"`) {
		t.Fatalf("expected install op, got %q", logs)
	}
}

func TestLogInstallOuterTxHoldIncludesError(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	LogInstallOuterTxHold(logger, "cli", time.Now().Add(-10*time.Millisecond), errSample)

	logs := logBuf.String()
	if !strings.Contains(logs, `"caller":"cli"`) {
		t.Fatalf("expected cli caller, got %q", logs)
	}
	if !strings.Contains(logs, `"error":"boom"`) {
		t.Fatalf("expected error attr, got %q", logs)
	}
}

func TestLogModuleCommitTxHoldZeroStartedNoOp(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	LogModuleCommitTxHold(logger, "upgrade", "module_commit", time.Time{}, nil)
	if buf.Len() != 0 {
		t.Fatalf("zero started must no-op, got %q", buf.String())
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
	if !strings.Contains(logs, `"op":"module"`) {
		t.Fatalf("expected default op=module, got %q", logs)
	}
	if !strings.Contains(logs, `"caller":"unknown"`) {
		t.Fatalf("expected default caller=unknown, got %q", logs)
	}
}

func mustParseLogDurationMs(t *testing.T, logs string) int64 {
	t.Helper()
	var rec struct {
		DurationMs int64 `json:"duration_ms"`
	}
	line := strings.TrimSpace(logs)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("parse log JSON: %v from %q", err, logs)
	}
	return rec.DurationMs
}

var errSample = errString("boom")

type errString string

func (e errString) Error() string { return string(e) }
