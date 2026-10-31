// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/server/runplan"
)

func TestStartRuntimeOwnersReadyHooksFailure(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()
	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "document", "globalThis.__readyHooksStartup = true;")
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)

	srv := NewServer(runtimeScope, WithExecutor(&readyHooksMockExec{
		err: errors.New("HOOK_UNSUPPORTED: ready failed"),
	})).(*GRPCWebServer)
	srv.runState.applicationNames = []string{"document"}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	opts := srv.resolvedRuntimeOptions()
	srv.initStartupState(opts)
	phase, _, _, err := srv.startRuntimeOwners(false, opts)
	if err == nil {
		t.Fatal("expected ready hooks failure")
	}
	if phase != startupPhaseReadyHooks {
		t.Fatalf("phase=%q want %q", phase, startupPhaseReadyHooks)
	}
	if !strings.Contains(err.Error(), "application ready hooks") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunStartupLifecycleFailedPhaseReadyHooks(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()
	writeServerTestAppDist(t, runtimeScope.cfg.DistPath, "document", "globalThis.__readyHooksLifecycle = true;")
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)

	srv := NewServer(runtimeScope, WithExecutor(&readyHooksMockExec{
		err: errors.New("HOOK_UNSUPPORTED: ready failed"),
	})).(*GRPCWebServer)
	srv.runState.applicationNames = []string{"document"}
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	opts := srv.resolvedRuntimeOptions()
	result := srv.runStartupLifecycle(false, opts)
	if result.errorValue() == nil {
		t.Fatal("expected startup failure")
	}
	assertStartupLifecycleStatus(t, result, false, false, startupPhaseReadyHooks, "ready hooks startup failure")
	assertModeRuntimeSummaryFields(t, result.ModeRuntime, runplan.RunModeApplication, 1, 1, true, true, "ready hooks startup failure")
}
