// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultengine/quickjsruntime"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
)

func TestStopJSExecutorLogsProcessHostStopError(t *testing.T) {
	quickjsruntime.ResetProcessModuleManagementHostForTest()
	t.Cleanup(quickjsruntime.ResetProcessModuleManagementHostForTest)

	quickjsruntime.SetProcessModuleManagementHostForTest(func() error {
		return errors.New("process stop boom")
	})

	var buf bytes.Buffer
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		jsExecutor:   jsexecutortest.NewUninitializedExecutor(),
	}
	if err := srv.stopJSExecutor(true); err != nil {
		t.Fatalf("reload stop: %v", err)
	}
	if !strings.Contains(buf.String(), "stop process module management host") {
		t.Fatalf("expected stop warn log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "process stop boom") {
		t.Fatalf("expected stop error in log, got %q", buf.String())
	}

	// Nil runtimeScope must not panic when the process host stop fails.
	quickjsruntime.SetProcessModuleManagementHostForTest(func() error {
		return errors.New("process stop boom again")
	})
	nilScopeSrv := &GRPCWebServer{runtimeScope: nil}
	if err := nilScopeSrv.stopJSExecutor(true); err != nil {
		t.Fatalf("nil-scope stop: %v", err)
	}

	// Non-reload path surfaces jsExecutor.Stop errors.
	srv.jsExecutor = jsexecutortest.NewUninitializedExecutor()
	if err := srv.stopJSExecutor(false); err == nil || !strings.Contains(err.Error(), "Failed to stop js executor") {
		t.Fatalf("expected js stop error, got %v", err)
	}
}
