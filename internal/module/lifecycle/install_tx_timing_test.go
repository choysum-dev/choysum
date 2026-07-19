// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogModuleCommitTxHoldEmitsDuration(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	LogModuleCommitTxHold(logger, "upgrade", "module_commit", time.Now().Add(-42*time.Millisecond), nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"upgrade commit transaction hold completed"`,
		`"op":"upgrade"`,
		`"caller":"module_commit"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
}

func TestLogModuleCommitTxHoldIncludesError(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	LogModuleCommitTxHold(logger, "uninstall", "module_commit", time.Now().Add(-10*time.Millisecond), errSample)

	logs := logBuf.String()
	if !strings.Contains(logs, `"op":"uninstall"`) {
		t.Fatalf("expected uninstall op, got %q", logs)
	}
	if !strings.Contains(logs, `"error":"boom"`) {
		t.Fatalf("expected error attr, got %q", logs)
	}
}

func TestLogInstallOuterTxHoldEmitsDuration(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	LogInstallOuterTxHold(logger, "bootstrap", time.Now().Add(-42*time.Millisecond), nil)

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"install commit transaction hold completed"`,
		`"caller":"bootstrap"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected %s in logs, got %q", want, logs)
		}
	}
}

func TestLogInstallOuterTxHoldIncludesError(t *testing.T) {
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

var errSample = errString("boom")

type errString string

func (e errString) Error() string { return string(e) }
