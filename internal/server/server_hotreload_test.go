// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestServerHotreloadLifecycleRecreatesWatcherForReuse(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true
	srv := &GRPCWebServer{runtimeScope: runtimeScope}

	if err := srv.startHotreloadLifecycle(); err != nil {
		t.Fatalf("startHotreloadLifecycle() first error = %v", err)
	}
	firstWatcher := srv.hotreloadWatcher()
	if firstWatcher == nil || srv.hotreloadQueue() == nil {
		t.Fatal("expected startHotreloadLifecycle to initialize watcher and queue")
	}
	srv.hotreloadQueue() <- "stale.ts"
	srv.stopHotreloadLifecycle()

	if err := srv.startHotreloadLifecycle(); err != nil {
		t.Fatalf("startHotreloadLifecycle() second error = %v", err)
	}
	defer srv.stopHotreloadLifecycle()
	if srv.hotreloadWatcher() == nil {
		t.Fatal("expected second hotreload lifecycle start to recreate watcher")
	}
	if srv.hotreloadWatcher() == firstWatcher {
		t.Fatal("expected second hotreload lifecycle start to create a new watcher instance")
	}
	select {
	case got := <-srv.hotreloadQueue():
		t.Fatalf("unexpected stale queued watch event after lifecycle restart: %q", got)
	default:
	}
}

func TestServerWatchHelpers(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()

	root := t.TempDir()
	moduleDir := filepath.Join(root, "modules", "demo")
	nestedDir := filepath.Join(moduleDir, "sub")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{filepath.Join(root, "missing"), moduleDir}}
	srv.hotreload = hotreloadState{watcher: watcher}
	runtimeScope.cfg.Server.HotReload = true

	if err := srv.applyRegistrationWatchPlansWithHandler(tracked.watchPlans(), tracked.watchCallback); err != nil {
		t.Fatalf("applyRegistrationWatchPlansWithHandler() error = %v", err)
	}
	watchList := watcher.WatchList()
	if len(watchList) == 0 {
		t.Fatal("expected registerWatchDir to add directories to watcher")
	}
	watchSet := map[string]bool{}
	for _, item := range watchList {
		watchSet[item] = true
	}
	resolvedModuleDir, err := resolveWatchPath(moduleDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(moduleDir) error = %v", err)
	}
	resolvedNestedDir, err := resolveWatchPath(nestedDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(nestedDir) error = %v", err)
	}
	if !watchSet[resolvedModuleDir] || !watchSet[resolvedNestedDir] {
		t.Fatalf("unexpected watch list: %#v", watchList)
	}

	changedFile := filepath.Join(nestedDir, "handler.ts")
	if err := os.WriteFile(changedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := srv.dispatchWatchHandler(changedFile); err != nil {
		t.Fatalf("dispatchWatchHandler() error = %v", err)
	}
	if tracked.callCount() != 1 {
		t.Fatalf("watch handler calls = %d, want 1", tracked.callCount())
	}
	resolvedChangedFile, err := resolveWatchPath(changedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath() error = %v", err)
	}
	firstCall, ok := tracked.firstCall()
	if !ok {
		t.Fatal("expected first watch callback")
	}
	if firstCall.module != "demo" || firstCall.file != resolvedChangedFile {
		t.Fatalf("unexpected watch callback: %#v", firstCall)
	}

	siblingDir := filepath.Join(root, "modules", "demo-sibling")
	if err := os.MkdirAll(siblingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	siblingFile := filepath.Join(siblingDir, "handler.ts")
	if err := os.WriteFile(siblingFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := srv.dispatchWatchHandler(siblingFile); err != nil {
		t.Fatalf("dispatchWatchHandler() sibling path error = %v", err)
	}
	if tracked.callCount() != 1 {
		t.Fatalf("watch handler calls after sibling path = %d, want 1", tracked.callCount())
	}

	tracked.watchErr = errors.New("watch failed")
	if _, err := srv.dispatchWatchHandler(changedFile); err == nil {
		t.Fatal("expected dispatchWatchHandler to propagate watch handler error")
	}

	quietSrv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{watcher: watcher}}
	runtimeScope.cfg.Server.HotReload = false
	if err := quietSrv.applyRegistrationWatchPlansWithHandler((&fakeWatchedService{name: "demo", watchDirs: []string{moduleDir}}).watchPlans(), nil); err != nil {
		t.Fatalf("applyRegistrationWatchPlansWithHandler() with hot reload disabled error = %v", err)
	}
}

func TestIsWatchedPathBoundaries(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "modules", "demo")
	nestedDir := filepath.Join(moduleDir, "sub")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	tests := []struct {
		name      string
		moduleDir string
		file      string
		want      bool
	}{
		{name: "same directory event", moduleDir: moduleDir, file: moduleDir, want: true},
		{name: "parent escape", moduleDir: nestedDir, file: filepath.Join(moduleDir, "handler.ts"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := isWatchedPath(tc.moduleDir, tc.file)
			if err != nil {
				t.Fatalf("isWatchedPath() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("isWatchedPath(%q, %q) = %v, want %v", tc.moduleDir, tc.file, got, tc.want)
			}
		})
	}
}

func TestServerRegisterWatchDirAndDispatchWatchHandlerResolvesSymlinks(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()

	root := t.TempDir()
	logicalModulesDir := filepath.Join(root, "modules")
	if err := os.MkdirAll(logicalModulesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	realModuleDir := filepath.Join(root, "real", "demo")
	realNestedDir := filepath.Join(realModuleDir, "sub")
	if err := os.MkdirAll(realNestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	symlinkModuleDir := filepath.Join(logicalModulesDir, "demo")
	if err := os.Symlink(realModuleDir, symlinkModuleDir); err != nil {
		t.Skipf("Symlink not supported: %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{symlinkModuleDir}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{watcher: watcher}}

	if err := srv.applyRegistrationWatchPlansWithHandler(tracked.watchPlans(), tracked.watchCallback); err != nil {
		t.Fatalf("applyRegistrationWatchPlansWithHandler() error = %v", err)
	}
	watchSet := map[string]bool{}
	for _, item := range watcher.WatchList() {
		watchSet[item] = true
	}
	resolvedModuleDir, err := resolveWatchPath(symlinkModuleDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(module) error = %v", err)
	}
	resolvedNestedDir, err := resolveWatchPath(filepath.Join(symlinkModuleDir, "sub"))
	if err != nil {
		t.Fatalf("resolveWatchPath(nested) error = %v", err)
	}
	if !watchSet[resolvedModuleDir] || !watchSet[resolvedNestedDir] {
		t.Fatalf("unexpected watch list for symlinked module: %#v", watcher.WatchList())
	}

	logicalChangedFile := filepath.Join(symlinkModuleDir, "sub", "handler.ts")
	if err := os.WriteFile(logicalChangedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := srv.dispatchWatchHandler(logicalChangedFile); err != nil {
		t.Fatalf("dispatchWatchHandler() error = %v", err)
	}
	firstCall, ok := tracked.firstCall()
	if !ok {
		t.Fatal("expected watch callback for symlinked module")
	}
	resolvedChangedFile, err := resolveWatchPath(logicalChangedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath(file) error = %v", err)
	}
	if firstCall.module != "demo" || firstCall.file != resolvedChangedFile {
		t.Fatalf("unexpected symlink watch callback: %#v", firstCall)
	}
}

func TestServerEnqueueWatchEventDropsWhenQueueFull(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{queue: make(chan string, 1)}}
	srv.hotreloadQueue() <- "existing.ts"

	if enqueued := srv.enqueueWatchEvent("dropped.ts"); enqueued {
		t.Fatal("expected enqueueWatchEvent to drop events when the queue is full")
	}
	assertHotreloadCounters(t, srv, 1, 0, "enqueueWatchEvent() drops when queue is full")

	select {
	case got := <-srv.hotreloadQueue():
		if got != "existing.ts" {
			t.Fatalf("watch queue retained %q, want %q", got, "existing.ts")
		}
	default:
		t.Fatal("expected existing watch event to remain queued")
	}

	select {
	case got := <-srv.hotreloadQueue():
		t.Fatalf("unexpected dropped watch event in queue: %q", got)
	default:
	}
}

func TestServerEnqueueWatchEventCoalescesWhileReloadPending(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	srv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{queue: make(chan string, 1)}}

	if enqueued := srv.enqueueWatchEvent("first.ts"); !enqueued {
		t.Fatal("expected first watch event to enqueue")
	}
	select {
	case got := <-srv.hotreloadQueue():
		resolvedFirst, err := resolveWatchPath("first.ts")
		if err != nil {
			t.Fatalf("resolveWatchPath(first) error = %v", err)
		}
		// Queue packs "file|module" to avoid enqueue/dequeue mismatch.
		want := resolvedFirst + "|"
		if got != want {
			t.Fatalf("queued watch event = %q, want %q", got, want)
		}
	default:
		t.Fatal("expected first watch event to remain queued")
	}

	if enqueued := srv.enqueueWatchEvent("second.ts"); enqueued {
		t.Fatal("expected second watch event to be coalesced while reload is pending")
	}
	assertHotreloadCounters(t, srv, 0, 1, "enqueueWatchEvent() coalesces while reload is pending")

	srv.finishWatchEvent()
	if enqueued := srv.enqueueWatchEvent("third.ts"); !enqueued {
		t.Fatal("expected third watch event to enqueue after finishing the pending reload")
	}
}

func TestWaitForWatchDebounceHonorsContextCancel(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	ctx, cancel := context.WithCancel(context.Background())
	runtimeScope.ctx = ctx
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	oldWindow := watchDebounceWindow
	watchDebounceWindow = time.Second
	t.Cleanup(func() { watchDebounceWindow = oldWindow })

	cancel()
	if err := srv.waitForWatchDebounce("demo.ts"); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForWatchDebounce() error = %v, want %v", err, context.Canceled)
	}
}

func TestWatchOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "modules", "demo")
	nestedDir := filepath.Join(moduleDir, "sub")
	siblingDir := filepath.Join(root, "modules", "other")
	registeredRoots := map[string]struct{}{moduleDir: {}, nestedDir: {}, siblingDir: {}}

	skip, coveredRoots, err := overlappingWatchRoots(nestedDir, registeredRoots)
	if err != nil {
		t.Fatalf("overlappingWatchRoots(child) error = %v", err)
	}
	if !skip {
		t.Fatal("expected child watch root to be skipped when a parent root already exists")
	}
	if len(coveredRoots) != 0 {
		t.Fatalf("coveredRoots for child = %#v, want empty", coveredRoots)
	}

	parentRoot := filepath.Join(root, "modules")
	skip, coveredRoots, err = overlappingWatchRoots(parentRoot, registeredRoots)
	if err != nil {
		t.Fatalf("overlappingWatchRoots(parent) error = %v", err)
	}
	if skip {
		t.Fatal("expected parent watch root not to be skipped")
	}
	coveredSet := map[string]bool{}
	for _, coveredRoot := range coveredRoots {
		coveredSet[coveredRoot] = true
	}
	if !coveredSet[moduleDir] || !coveredSet[nestedDir] || !coveredSet[siblingDir] {
		t.Fatalf("coveredRoots = %#v, want module+nested+sibling", coveredRoots)
	}
}

func TestServerServeRestartSuccessLogIncludesWatchCounters(t *testing.T) {
	buf := &bytes.Buffer{}
	baseScope := newRichServerTestScope(t)
	baseScope.logger = slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	runtimeScope := (&noSessionServerScope{serverTestScope: baseScope}).WithContext(context.Background()).(*noSessionServerScope)
	runtimeScope.cfg.Auth.Enabled = false
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	watchRoot := t.TempDir()
	changedFile := filepath.Join(watchRoot, "changed.ts")
	if err := os.WriteFile(changedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{watchRoot}}
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

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) error = %v", err)
	}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))
	srv.hotreload.recordDropped()
	srv.hotreload.recordCoalesced()
	if err := srv.handleWatchedFileChange(changedFile); err != nil {
		t.Fatalf("handleWatchedFileChange() error = %v", err)
	}
	if tracked.callCount() != 1 {
		t.Fatalf("watch handler calls = %d, want 1", tracked.callCount())
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "watch reload completed") || !strings.Contains(logOutput, "watch_dropped_count=1") || !strings.Contains(logOutput, "watch_coalesced_count=1") {
		t.Fatalf("unexpected restart success log output: %s", logOutput)
	}
}

func TestWatchEventLogsIncludeStructuredFields(t *testing.T) {
	buf := &bytes.Buffer{}
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.logger = slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dropSrv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{queue: make(chan string, 1)}}
	dropSrv.hotreloadQueue() <- "existing.ts"
	if enqueued := dropSrv.enqueueWatchEvent("dropped.ts"); enqueued {
		t.Fatal("expected dropped watch event")
	}

	coalesceSrv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{queue: make(chan string, 1)}}
	if enqueued := coalesceSrv.enqueueWatchEvent("first.ts"); !enqueued {
		t.Fatal("expected first watch event to enqueue")
	}
	if enqueued := coalesceSrv.enqueueWatchEvent("second.ts"); enqueued {
		t.Fatal("expected second watch event to be coalesced")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "watch event dropped") || !strings.Contains(logOutput, "reason=queue_full") || !strings.Contains(logOutput, "dropped_count=1") {
		t.Fatalf("unexpected dropped watch log output: %s", logOutput)
	}
	if !strings.Contains(logOutput, "watch event coalesced") || !strings.Contains(logOutput, "coalesced_count=1") {
		t.Fatalf("unexpected coalesced watch log output: %s", logOutput)
	}
}

func TestServerDispatchWatchHandlerResolvesSymlinkedRemovedFile(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	root := t.TempDir()
	logicalModulesDir := filepath.Join(root, "modules")
	if err := os.MkdirAll(logicalModulesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	realModuleDir := filepath.Join(root, "real", "demo")
	realNestedDir := filepath.Join(realModuleDir, "sub")
	if err := os.MkdirAll(realNestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	symlinkModuleDir := filepath.Join(logicalModulesDir, "demo")
	if err := os.Symlink(realModuleDir, symlinkModuleDir); err != nil {
		t.Skipf("Symlink not supported: %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{symlinkModuleDir}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	logicalRemovedFile := filepath.Join(symlinkModuleDir, "sub", "removed.ts")
	if err := os.WriteFile(logicalRemovedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Remove(logicalRemovedFile); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := srv.dispatchWatchHandler(logicalRemovedFile); err != nil {
		t.Fatalf("dispatchWatchHandler() error = %v", err)
	}
	firstCall, ok := tracked.firstCall()
	if !ok {
		t.Fatal("expected watch callback for symlinked removed file")
	}
	resolvedRemovedFile, err := resolveWatchPath(logicalRemovedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath(removed file) error = %v", err)
	}
	if firstCall.module != "demo" || firstCall.file != resolvedRemovedFile {
		t.Fatalf("unexpected symlink remove callback: %#v", firstCall)
	}
}

func TestServerDispatchWatchHandlerResolvesSymlinkedRenamedOldFile(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	root := t.TempDir()
	logicalModulesDir := filepath.Join(root, "modules")
	if err := os.MkdirAll(logicalModulesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	realModuleDir := filepath.Join(root, "real", "demo")
	realNestedDir := filepath.Join(realModuleDir, "sub")
	if err := os.MkdirAll(realNestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	symlinkModuleDir := filepath.Join(logicalModulesDir, "demo")
	if err := os.Symlink(realModuleDir, symlinkModuleDir); err != nil {
		t.Skipf("Symlink not supported: %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{symlinkModuleDir}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	logicalOldFile := filepath.Join(symlinkModuleDir, "sub", "rename-old.ts")
	logicalNewFile := filepath.Join(symlinkModuleDir, "sub", "rename-new.ts")
	if err := os.WriteFile(logicalOldFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Rename(logicalOldFile, logicalNewFile); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if _, err := srv.dispatchWatchHandler(logicalOldFile); err != nil {
		t.Fatalf("dispatchWatchHandler() error = %v", err)
	}
	firstCall, ok := tracked.firstCall()
	if !ok {
		t.Fatal("expected watch callback for symlinked renamed file")
	}
	resolvedOldFile, err := resolveWatchPath(logicalOldFile)
	if err != nil {
		t.Fatalf("resolveWatchPath(old file) error = %v", err)
	}
	if firstCall.module != "demo" || firstCall.file != resolvedOldFile {
		t.Fatalf("unexpected symlink rename callback: %#v", firstCall)
	}
}

func TestServerRegisterWatchDirSkipsWalkErrors(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	if err := watcher.Close(); err != nil {
		t.Fatalf("watcher.Close() error = %v", err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	srv := &GRPCWebServer{runtimeScope: runtimeScope, hotreload: hotreloadState{watcher: watcher}}
	app := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	if err := srv.applyRegistrationWatchPlansWithHandler(app.watchPlans(), app.watchCallback); err != nil {
		t.Fatalf("applyRegistrationWatchPlansWithHandler() error = %v, want nil on walk/add failure", err)
	}
}

func TestServerWatchForwardsEventsAndStops(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		hotreload:    hotreloadState{watcher: watcher, queue: make(chan string, 8)},
	}
	if err := srv.startHotreloadLifecycle(); err != nil {
		t.Fatalf("startHotreloadLifecycle() error = %v", err)
	}
	t.Cleanup(func() {
		srv.stopHotreloadLifecycle()
	})

	watchDir := t.TempDir()
	if err := watcher.Add(watchDir); err != nil {
		t.Fatalf("watcher.Add() error = %v", err)
	}
	created := filepath.Join(watchDir, "created.txt")
	if err := os.WriteFile(created, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	select {
	case got := <-srv.hotreloadQueue():
		resolvedCreated, err := resolveWatchPath(created)
		if err != nil {
			t.Fatalf("resolveWatchPath() error = %v", err)
		}
		// Queue packs "file|module" to avoid enqueue/dequeue mismatch.
		want := resolvedCreated + "|"
		if got != want {
			t.Fatalf("watch forwarded path = %q, want %q", got, want)
		}
		srv.finishWatchEvent()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for fsnotify event")
	}

	srv.stopHotreloadLifecycle()
}

func TestServerWatchForwardsSyntheticEvents(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		hotreload:    hotreloadState{watcher: watcher, queue: make(chan string, 8)},
	}
	if err := srv.startHotreloadLifecycle(); err != nil {
		t.Fatalf("startHotreloadLifecycle() error = %v", err)
	}
	t.Cleanup(func() {
		srv.stopHotreloadLifecycle()
	})

	root := t.TempDir()
	removeFile := filepath.Join(root, "remove.ts")
	renameOld := filepath.Join(root, "rename.ts")
	renameNew := filepath.Join(root, "rename-next.ts")
	if err := os.WriteFile(removeFile, []byte("export const removeMe = true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(remove.ts) error = %v", err)
	}
	if err := os.WriteFile(renameOld, []byte("export const renameMe = true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(rename.ts) error = %v", err)
	}
	if err := watcher.Add(root); err != nil {
		t.Fatalf("watcher.Add() error = %v", err)
	}
	waitForExpectedEvent := func(expected ...string) string {
		t.Helper()
		deadline := time.After(3 * time.Second)
		for {
			select {
			case eventInfo := <-srv.hotreloadQueue():
				srv.finishWatchEvent()
				// Strip module suffix from packed "file|module" format.
				file, _, _ := strings.Cut(eventInfo, "|")
				base := filepath.Base(file)
				for _, want := range expected {
					if base == want {
						return base
					}
				}
			case <-deadline:
				t.Fatalf("timed out waiting for watch event, want one of %#v", expected)
			}
		}
	}
	waitForQueueIdle := func(idleFor time.Duration) {
		t.Helper()
		idle := time.NewTimer(idleFor)
		defer idle.Stop()
		for {
			select {
			case <-idle.C:
				return
			case <-srv.hotreloadQueue():
				srv.finishWatchEvent()
				if !idle.Stop() {
					select {
					case <-idle.C:
					default:
					}
				}
				idle.Reset(idleFor)
			}
		}
	}
	createFile := filepath.Join(root, "create.ts")
	if err := os.WriteFile(createFile, []byte("export const created = true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(create.ts) error = %v", err)
	}
	if got := waitForExpectedEvent("create.ts"); got != "create.ts" {
		t.Fatalf("watch forwarded create event = %q, want %q", got, "create.ts")
	}
	// Drain trailing create/write/chmod events so the next operation is not coalesced away.
	waitForQueueIdle(150 * time.Millisecond)
	if err := os.Remove(removeFile); err != nil {
		t.Fatalf("Remove(remove.ts) error = %v", err)
	}
	if got := waitForExpectedEvent("remove.ts"); got != "remove.ts" {
		t.Fatalf("watch forwarded remove event = %q, want %q", got, "remove.ts")
	}
	waitForQueueIdle(150 * time.Millisecond)
	if err := os.Rename(renameOld, renameNew); err != nil {
		t.Fatalf("Rename(rename.ts) error = %v", err)
	}
	gotRename := waitForExpectedEvent("rename.ts", "rename-next.ts")
	if gotRename != "rename.ts" && gotRename != "rename-next.ts" {
		t.Fatalf("watch forwarded rename event = %q, want %q or %q", gotRename, "rename.ts", "rename-next.ts")
	}

	srv.stopHotreloadLifecycle()
}

func TestContentChanged_SkipsNoOpWrite(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "unchanged.ts")
	content := []byte("export const v = 1;\n")

	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	if !hs.contentChanged(file) {
		t.Fatal("expected first write to be treated as changed (no cache entry)")
	}
	if hs.contentChanged(file) {
		t.Fatal("expected identical write to be skipped")
	}
}

func TestContentChanged_DetectsContentChange(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "changed.ts")

	if err := os.WriteFile(file, []byte("v1"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	if !hs.contentChanged(file) {
		t.Fatal("expected first write to be treated as changed")
	}
	if err := os.WriteFile(file, []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !hs.contentChanged(file) {
		t.Fatal("expected second write with different content to be treated as changed")
	}
}

func TestContentChanged_LargeFileSkipsNoOpWrite(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "large.bin")
	content := bytes.Repeat([]byte("a"), 2<<20)

	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	if !hs.contentChanged(file) {
		t.Fatal("expected first write to be treated as changed")
	}
	if hs.contentChanged(file) {
		t.Fatal("expected identical large-file write to be skipped")
	}

	changed := append([]byte(nil), content...)
	changed[len(changed)/2] = 'b'
	if err := os.WriteFile(file, changed, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !hs.contentChanged(file) {
		t.Fatal("expected changed large-file write to be treated as changed")
	}
}

func TestContentChanged_MissingFileTreatedAsChanged(t *testing.T) {
	hs := &hotreloadState{}
	if !hs.contentChanged("/nonexistent/path.ts") {
		t.Fatal("expected missing file to be treated as changed")
	}
}

func TestContentChanged_DirectoryTreatedAsChanged(t *testing.T) {
	root := t.TempDir()
	hs := &hotreloadState{}
	if !hs.contentChanged(root) {
		t.Fatal("expected directory to be treated as changed")
	}
}

func TestContentChanged_ClearFingerprintsResetsState(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "reset.ts")

	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	hs.contentChanged(file) // prime cache
	if hs.contentChanged(file) {
		t.Fatal("expected identical write to be skipped after priming")
	}
	hs.clearFingerprints()
	if !hs.contentChanged(file) {
		t.Fatal("expected write after clearing fingerprints to be treated as changed")
	}
}

func TestApplyRegistrationWatchPlansPrimesFingerprintsForFirstNoOpSave(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true

	root := t.TempDir()
	changedFile := filepath.Join(root, "OLayout.vue")
	content := []byte("<template><div /></template>\n")
	if err := os.WriteFile(changedFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.applyRegistrationWatchPlansWithHandler(tracked.watchPlans(), tracked.watchCallback); err != nil {
		t.Fatalf("applyRegistrationWatchPlansWithHandler() error = %v", err)
	}

	// No-op first save should be ignored because watch registration already
	// primed the initial fingerprint baseline.
	if err := os.WriteFile(changedFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := srv.handleWatchedFileChange(changedFile); err != nil {
		t.Fatalf("handleWatchedFileChange() error = %v", err)
	}
	if tracked.callCount() != 0 {
		t.Fatalf("watch handler calls = %d, want 0 (first no-op save)", tracked.callCount())
	}

	if err := os.WriteFile(changedFile, []byte("<template><section /></template>\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !srv.hotreload.contentChanged(changedFile) {
		t.Fatal("expected contentChanged() to detect real content change after baseline priming")
	}
}

func TestHandleWatchedFileChangeSkipsNoOpWrite(t *testing.T) {
	// The content fingerprint check in handleWatchedFileChange returns nil
	// before dispatch when the file content hasn't changed.
	runtimeScope := newRichServerTestScope(t)

	root := t.TempDir()
	changedFile := filepath.Join(root, "handler.ts")

	if err := os.WriteFile(changedFile, []byte("export const v = 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	// Prime fingerprint.
	srv.hotreload.contentChanged(changedFile)

	// Identical content → handleWatchedFileChange returns nil early.
	if err := os.WriteFile(changedFile, []byte("export const v = 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := srv.handleWatchedFileChange(changedFile); err != nil {
		t.Fatalf("handleWatchedFileChange() error = %v", err)
	}
	if tracked.callCount() != 0 {
		t.Fatalf("watch handler calls = %d, want 0 (no-op save)", tracked.callCount())
	}

	// Changed content → dispatchWatchHandler invokes callback.
	if err := os.WriteFile(changedFile, []byte("export const v = 2;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !srv.hotreload.contentChanged(changedFile) {
		t.Fatal("expected contentChanged to return true after content change")
	}
	if _, err := srv.dispatchWatchHandler(changedFile); err != nil {
		t.Fatalf("dispatchWatchHandler() error = %v", err)
	}
	if tracked.callCount() != 1 {
		t.Fatalf("watch handler calls = %d, want 1 (real change)", tracked.callCount())
	}
}

func TestHandleWatchedFileChangeSkipsNoOpAtomicSave(t *testing.T) {
	// VS Code atomic save (temp + rename → target). Content unchanged.
	runtimeScope := newRichServerTestScope(t)

	root := t.TempDir()
	changedFile := filepath.Join(root, "OLayout.vue")
	content := []byte("export default {};\n")

	if err := os.WriteFile(changedFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))
	srv.hotreload.contentChanged(changedFile)

	tmpFile := changedFile + ".tmp"
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile(tmp) error = %v", err)
	}
	if err := os.Rename(tmpFile, changedFile); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if err := srv.handleWatchedFileChange(changedFile); err != nil {
		t.Fatalf("handleWatchedFileChange() error = %v", err)
	}
	if tracked.callCount() != 0 {
		t.Fatalf("watch handler calls = %d, want 0 (no-op atomic save)", tracked.callCount())
	}
}

func TestPrimeFingerprintsForTargetsDedupRoots(t *testing.T) {
	root := t.TempDir()
	shared := filepath.Join(root, "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := []byte("shared content\n")
	if err := os.WriteFile(filepath.Join(shared, "a.ts"), content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	// Two targets sharing the same root — second should be dedup'd.
	hs.primeFingerprintsForTargets([]registeredWatchTarget{
		{root: shared},
		{root: shared},
	})

	// Both files should have been fingerprinted exactly once.
	if !hs.hasFingerprint(filepath.Join(shared, "a.ts")) {
		t.Fatal("expected shared-root file to have fingerprint after primeFingerprintsForTargets")
	}
}

func TestPrimeFingerprintsForRoots(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	if err := os.MkdirAll(dirA, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(dirB, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "x.ts"), []byte("a\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "y.ts"), []byte("b\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	hs.primeFingerprintsForRoots([]string{dirA, dirB})

	if !hs.hasFingerprint(filepath.Join(dirA, "x.ts")) {
		t.Fatal("expected dirA file to have fingerprint")
	}
	if !hs.hasFingerprint(filepath.Join(dirB, "y.ts")) {
		t.Fatal("expected dirB file to have fingerprint")
	}

	// Duplicate roots should not cause issues.
	hs.primeFingerprintsForRoots([]string{dirA, dirA})
}

func TestPrimeFingerprintsForRootSkipsNoiseDirs(t *testing.T) {
	root := t.TempDir()
	for _, noiseDir := range []string{"node_modules", ".git", "dist", "build"} {
		deep := filepath.Join(root, noiseDir, "deep")
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", noiseDir, err)
		}
		if err := os.WriteFile(filepath.Join(deep, "should-skip.ts"), []byte("skip\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", noiseDir, err)
		}
	}
	legit := filepath.Join(root, "src", "real.ts")
	if err := os.MkdirAll(filepath.Dir(legit), 0o755); err != nil {
		t.Fatalf("MkdirAll(src) error = %v", err)
	}
	if err := os.WriteFile(legit, []byte("real\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(real.ts) error = %v", err)
	}

	hs := &hotreloadState{}
	hs.primeFingerprintsForRoot(root)

	if !hs.hasFingerprint(legit) {
		t.Fatal("expected non-noise file to have fingerprint")
	}
	for _, noiseDir := range []string{"node_modules", ".git", "dist", "build"} {
		noiseFile := filepath.Join(root, noiseDir, "deep", "should-skip.ts")
		if hs.hasFingerprint(noiseFile) {
			t.Fatalf("expected noise-dir file %q to be skipped", noiseFile)
		}
	}
}

func TestHasFingerprintReturnsFalseForNilMap(t *testing.T) {
	hs := &hotreloadState{}
	if hs.hasFingerprint("any.ts") {
		t.Fatal("expected hasFingerprint to return false when fingerprints is nil")
	}
}

func TestContentChangedResolvedRemovesDeletedFingerprint(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "temp.ts")
	if err := os.WriteFile(file, []byte("temp\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hs := &hotreloadState{}
	// Prime the cache.
	if !hs.contentChangedResolved(file) {
		t.Fatal("expected first write to be treated as changed")
	}
	if !hs.hasFingerprint(file) {
		t.Fatal("expected fingerprint to be cached after first write")
	}

	// Delete the file — contentChangedResolved should return true and
	// clean up the stale fingerprint entry.
	if err := os.Remove(file); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if !hs.contentChangedResolved(file) {
		t.Fatal("expected deleted file to be treated as changed")
	}
	if hs.hasFingerprint(file) {
		t.Fatal("expected deleted file fingerprint to be removed from cache")
	}
}

func TestResolveWatchModuleWithResolvedPath(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	root := t.TempDir()
	moduleDir := filepath.Join(root, "modules", "demo")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{moduleDir}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	resolvedFile, err := resolveWatchPath(filepath.Join(moduleDir, "a.ts"))
	if err != nil {
		t.Fatalf("resolveWatchPath() error = %v", err)
	}

	if got := srv.resolveWatchModule(resolvedFile); got != "demo" {
		t.Fatalf("resolveWatchModule() = %q, want %q", got, "demo")
	}

	// File outside any watch root.
	outside := filepath.Join(root, "other", "x.ts")
	if err := os.MkdirAll(filepath.Dir(outside), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	resolvedOutside, err := resolveWatchPath(outside)
	if err != nil {
		t.Fatalf("resolveWatchPath(outside) error = %v", err)
	}
	if got := srv.resolveWatchModule(resolvedOutside); got != "" {
		t.Fatalf("resolveWatchModule(outside) = %q, want \"\"", got)
	}

	// Nil server.
	if got := (*GRPCWebServer)(nil).resolveWatchModule("any.ts"); got != "" {
		t.Fatalf("resolveWatchModule(nil server) = %q, want \"\"", got)
	}
}

func TestDispatchWatchHandlerResolvedFastPath(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	root := t.TempDir()
	moduleDir := filepath.Join(root, "modules", "demo")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	changedFile := filepath.Join(moduleDir, "handler.ts")
	if err := os.WriteFile(changedFile, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{moduleDir}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))

	// Use dispatchWatchHandlerResolved with already-resolved path.
	resolvedFile, err := resolveWatchPath(changedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath() error = %v", err)
	}
	if dispatched, err := srv.dispatchWatchHandlerResolved(resolvedFile); err != nil {
		t.Fatalf("dispatchWatchHandlerResolved() error = %v", err)
	} else if dispatched != 1 {
		t.Fatalf("dispatchWatchHandlerResolved() dispatched = %d, want 1", dispatched)
	}
	if tracked.callCount() != 1 {
		t.Fatalf("watch handler calls = %d, want 1", tracked.callCount())
	}
}

func TestHandleWatchedFileChangeResolvedProgressLineNil(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = false
	root := t.TempDir()
	changedFile := filepath.Join(root, "handler.ts")
	content := []byte("export const v = 1;\n")
	if err := os.WriteFile(changedFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))
	// Prime fingerprint so identical content does not reload.
	srv.hotreload.contentChanged(changedFile)
	if err := os.WriteFile(changedFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolvedFile, err := resolveWatchPath(changedFile)
	if err != nil {
		t.Fatalf("resolveWatchPath() error = %v", err)
	}

	// No progressLine set — the nil branches in handleWatchedFileChangeResolved
	// should be exercised without panicking.
	if err := srv.handleWatchedFileChangeResolved(resolvedFile); err != nil {
		t.Fatalf("handleWatchedFileChangeResolved() error = %v", err)
	}
	// No-op save: content unchanged => should not dispatch.
	if tracked.callCount() != 0 {
		t.Fatalf("watch handler calls = %d, want 0 (no-op save with nil progressLine)", tracked.callCount())
	}
}

func TestHandleWatchedFileChangeResolvesPath(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	root := t.TempDir()
	changedFile := filepath.Join(root, "handler.ts")
	if err := os.WriteFile(changedFile, []byte("export const v = 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tracked := &fakeWatchedService{name: "demo", watchDirs: []string{root}}
	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	srv.hotreload.storeWatchTargets(srv.buildRegisteredWatchTargets(tracked.watchPlans(), tracked.watchCallback))
	srv.hotreload.contentChanged(changedFile)

	// Unchanged content — handleWatchedFileChange should return nil.
	if err := os.WriteFile(changedFile, []byte("export const v = 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := srv.handleWatchedFileChange(changedFile); err != nil {
		t.Fatalf("handleWatchedFileChange() error = %v", err)
	}
	if tracked.callCount() != 0 {
		t.Fatalf("watch handler calls = %d, want 0", tracked.callCount())
	}
}
