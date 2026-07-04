// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
)

func TestServerServeReturnsErrorForInvalidBundleMode(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "broken"

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
	})

	err := srv.Serve(context.Background())
	if err == nil {
		t.Fatal("expected Serve() to fail for invalid compile bundle mode")
	}
}

func TestServerServeReturnsValidationErrorForMissingWebDist(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
	})

	err := srv.Serve(context.Background(), "web")
	if err == nil {
		t.Fatal("expected Serve() to fail when requested web dist is missing")
	}
	if !strings.Contains(err.Error(), "web dist missing") {
		t.Fatalf("Serve() error = %v, want web dist missing", err)
	}
}

func TestServerServeFallsThroughToServeAfterValidation(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.JsEngineFactory = "missing"

	if err := os.MkdirAll(filepath.Join(runtimeScope.cfg.DistPath, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) error = %v", err)
	}

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
	})

	err := srv.Serve(context.Background(), "web")
	if err == nil {
		t.Fatal("expected Serve() to surface wrapped serve/start failure")
	}
	if !strings.Contains(err.Error(), "failed to start server") {
		t.Fatalf("Serve() error = %v, want wrapped start failure", err)
	}
}

func TestServerServeReturnsWrappedStartError(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	runtimeScope.cfg.Server.JsEngineFactory = "missing"

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	err := srv.serve()
	if err == nil {
		t.Fatal("expected serve() to fail when start() cannot initialize js executor")
	}
	if !strings.Contains(err.Error(), "failed to start server") {
		t.Fatalf("serve() error = %v, want wrapped start failure", err)
	}
}

func TestServerServeSkipsRestartWhenNoModuleMatchesWatchedFile(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		hotreload:    hotreloadState{watcher: mustNewWatcher(t), queue: make(chan string, 1)},
	}
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	runtimeScope.ctx = ctx

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.serve()
	}()

	deadline := time.After(3 * time.Second)
	for !srv.ready.Load() {
		select {
		case err := <-errCh:
			t.Fatalf("serve() returned before ready: %v", err)
		case <-deadline:
			t.Fatal("timed out waiting for server to become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Feed a file that matches no registered watch target.
	// dispatchWatchHandler skips restart when dispatched == 0,
	// so serve() must keep running instead of returning an error.
	srv.hotreloadQueue() <- filepath.Join(t.TempDir(), "unmatched.ts")

	// Give serve() time to process the event.
	time.Sleep(200 * time.Millisecond)

	select {
	case err := <-errCh:
		t.Fatalf("serve() returned unexpectedly after non-matching file: %v", err)
	default:
	}

	// Cancel the runtime context so serve() returns cleanly.
	cancel()
	err := <-errCh
	if err != nil {
		t.Fatalf("serve() returned error after context cancel: %v", err)
	}
}

func TestServerServeReturnsNilWhenWatchHandlerCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtimeScope := &noSessionServerScope{serverTestScope: &serverTestScope{
		ctx: ctx,
		cfg: &config.Config{
			Server:  config.NewDefaultServerConfig(),
			Auth:    config.NewDefaultAuthConfig(),
			Compile: config.NewDefaultCompileConfig(),
			Log:     config.NewDefaultLogConfig(),
			Db:      config.NewDefaultDbConfig(),
		},
	}}
	runtimeScope.cfg.Auth.Enabled = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	watchRoot := t.TempDir()
	changedFile := filepath.Join(watchRoot, "changed.ts")
	if err := os.WriteFile(changedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		hotreload:    hotreloadState{watcher: mustNewWatcher(t), queue: make(chan string, 1)},
	}
	t.Cleanup(func() {
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.serve()
	}()

	deadline := time.After(3 * time.Second)
	for !srv.ready.Load() {
		select {
		case err := <-errCh:
			t.Fatalf("serve() returned before ready: %v", err)
		case <-deadline:
			t.Fatal("timed out waiting for server to become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets((&fakeWatchedService{
		name:      "demo",
		watchDirs: []string{watchRoot},
		watchErr:  context.Canceled,
	}).watchPlans(), func(moduleName string, file string) error {
		return context.Canceled
	}))
	cancel()
	srv.hotreloadQueue() <- changedFile

	if err := <-errCh; err != nil {
		t.Fatalf("serve() error = %v, want nil on canceled watch handler", err)
	}
}

func TestServerServeRestartsOnWatchedFileChangeAndStopsOnContextCancel(t *testing.T) {
	runtimeScope := (&noSessionServerScope{serverTestScope: newRichServerTestScope(t)}).WithContext(context.Background()).(*noSessionServerScope)
	runtimeScope.cfg.Auth.Enabled = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false
	ctx, cancel := context.WithCancel(runtimeScope.Context())
	runtimeScope = runtimeScope.WithContext(ctx).(*noSessionServerScope)

	watchRoot := t.TempDir()
	changedFile := filepath.Join(watchRoot, "changed.ts")
	if err := os.WriteFile(changedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{watchRoot}, callCh: make(chan watchedCall, 1)}
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		hotreload:    hotreloadState{watcher: mustNewWatcher(t), queue: make(chan string, 1)},
	}
	t.Cleanup(func() {
		cancel()
		if watcher := srv.hotreloadWatcher(); watcher != nil {
			_ = watcher.Close()
		}
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.serve()
	}()

	deadline := time.After(3 * time.Second)
	for !srv.ready.Load() {
		select {
		case err := <-errCh:
			t.Fatalf("serve() returned before ready: %v", err)
		case <-deadline:
			t.Fatal("timed out waiting for server to become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	srv.hotreloadQueue() <- changedFile

	restartDeadline := time.After(3 * time.Second)
	var firstCall watchedCall
	gotCall := false
	for !gotCall || !srv.ready.Load() {
		select {
		case firstCall = <-tracked.callCh:
			gotCall = true
		case err := <-errCh:
			t.Fatalf("serve() returned during successful restart path: %v", err)
		case <-restartDeadline:
			t.Fatalf("timed out waiting for serve() to process file change, calls=%d ready=%v", tracked.callCount(), srv.ready.Load())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	if firstCall.module != filepath.Base(watchRoot) {
		t.Fatalf("watch handler module = %q, want %q", firstCall.module, filepath.Base(watchRoot))
	}
	resolvedChangedFile, err := resolveWatchPath(changedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath() error = %v", err)
	}
	if firstCall.file != resolvedChangedFile {
		t.Fatalf("watch handler file = %q, want %q", firstCall.file, resolvedChangedFile)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve() error = %v, want nil after context cancellation", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for serve() to stop after context cancellation")
	}
}
