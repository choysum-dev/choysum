// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

func TestServerRegistersApplicationServicesAndWebHandlers(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	reg := &trackingRegistry{}
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		registry:     reg,
	}
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "demo.Service",
		HandlerType: (*interface{})(nil),
		Methods:     []grpc.MethodDesc{{MethodName: "Get"}},
		Streams:     []grpc.StreamDesc{{StreamName: "Watch"}},
	}
	app := &fakeWatchedService{
		name:     "demo",
		descs:    []*grpc.ServiceDesc{serviceDesc},
		handlers: map[string]http.Handler{"/demo": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })},
		scripts:  []*jsengine.JsScript{{FileName: "demo.js"}},
	}
	batch := srv.registration.beginBatch()

	if err := batch.registerBinding(srv, app); err != nil {
		t.Fatalf("registerBinding() error = %v", err)
	}
	registration := batch.commit()
	assertRegisteredGRPCMethods(t, srv, map[string]struct{}{
		"/demo.Service/Get":   {},
		"/demo.Service/Watch": {},
	}, "registerService() indexes grpc methods")
	if len(reg.registerCalls) != 1 || reg.registerCalls[0] != "demo.Service" {
		t.Fatalf("unexpected registry register calls: %#v", reg.registerCalls)
	}
	assertRegistrationInitScripts(t, registration, []string{"demo.js"}, "registerService() result")
	assertRegistrationBindings(t, registration, []registrationBinding{app}, "registerService() result")
	assertRegisteredBindings(t, srv, []registrationBinding{app}, "registerService() commit")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("mux status = %d, want %d", rec.Code, http.StatusCreated)
	}

	badApp := &fakeWatchedService{name: "broken", descErr: errors.New("desc failed")}
	batch = srv.registration.beginBatch()
	if err := batch.registerBinding(srv, badApp); err == nil {
		t.Fatal("expected registerService to return descriptor error")
	}

	batch = srv.registration.beginBatch()
	if err := batch.registerBinding(srv, &fakeWatchedService{name: "bad-web", webErr: errors.New("web failed")}); err != nil {
		t.Fatalf("registerBinding() error = %v, want nil because web handler discovery is best-effort", err)
	}
}

func TestServerRegisterApplicationBindingDoesNotEmitInfoLog(t *testing.T) {
	var logBuf bytes.Buffer
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		registry:     &trackingRegistry{},
	}
	app := &fakeWatchedService{name: "demo"}
	batch := srv.registration.beginBatch()

	if err := srv.registerApplicationBinding(batch, app); err != nil {
		t.Fatalf("registerApplicationBinding() error = %v", err)
	}

	if strings.Contains(logBuf.String(), "application service registered") {
		t.Fatalf("expected application service registration to be hidden at info level, got %q", logBuf.String())
	}
}

func TestServerRegisterApplicationServicesWithNoTargets(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
	}
	registration, err := srv.registerApplicationServices()
	if err != nil {
		t.Fatalf("registerApplicationServices() error = %v", err)
	}
	assertRegistrationInitScripts(t, registration, []string{}, "registerApplicationServices() with no targets")
	assertRegistrationBindings(t, registration, []registrationBinding{}, "registerApplicationServices() with no targets")
	assertRegisteredBindings(t, srv, []registrationBinding{}, "registerApplicationServices() with no targets")
	assertRegisteredGRPCMethods(t, srv, map[string]struct{}{}, "registerApplicationServices() with no targets")
	assertRegistrationGRPCMethods(t, registration, map[string]struct{}{}, "registerApplicationServices() with no targets")
}

func TestServerRegisterApplicationServicesBuildsWatchDirsAndScripts(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.Server.HotReload = true
	distDir := t.TempDir()
	modulesDir := t.TempDir()
	runtimeScope.cfg.DistPath = distDir
	runtimeScope.cfg.ModulesPath = modulesDir

	appDistDir := filepath.Join(distDir, "apps", "auth")
	if err := os.MkdirAll(appDistDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(appDistDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDistDir, "index.js"), []byte("console.log('auth')"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.js) error = %v", err)
	}
	authModuleDir := filepath.Join(modulesDir, "auth")
	baseModuleDir := filepath.Join(modulesDir, "base")
	for _, dir := range []string{authModuleDir, baseModuleDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		hotreload:    hotreloadState{watcher: watcher},
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		runState: runState{
			applicationNames: []string{"auth"},
			distManifest: &distmanifest.DistManifestV2{Apps: map[string]distmanifest.DistManifestApp{
				"auth": {Dev: distmanifest.DistManifestAppDev{Modules: []string{" auth ", "", "base", "auth"}}},
			}},
		},
	}

	registration, err := srv.registerApplicationServices()
	if err != nil {
		t.Fatalf("registerApplicationServices() error = %v", err)
	}
	assertRegistrationInitScripts(t, registration, []string{filepath.Join(appDistDir, "index.js")}, "registerApplicationServices() init scripts")
	requireSingleRegistrationBinding(t, registration, "registerApplicationServices() result bindings")
	assertRegisteredBindings(t, srv, registration.Bindings, "registerApplicationServices() registered bindings")
	assertRegisteredGRPCMethods(t, srv, map[string]struct{}{}, "registerApplicationServices() registered grpc methods")
	assertRegistrationGRPCMethods(t, registration, map[string]struct{}{}, "registerApplicationServices() grpc methods")

	watchTargets := srv.hotreload.watchTargetsSnapshot()
	if len(watchTargets) != 2 {
		t.Fatalf("watch target len = %d, want 2 (%#v)", len(watchTargets), watchTargets)
	}
	watchRoots := map[string]bool{}
	watchModules := map[string]bool{}
	for _, target := range watchTargets {
		watchRoots[target.root] = true
		watchModules[target.moduleName] = true
	}
	resolvedAuthModuleDir, err := resolveWatchPath(authModuleDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(authModuleDir) error = %v", err)
	}
	resolvedBaseModuleDir, err := resolveWatchPath(baseModuleDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(baseModuleDir) error = %v", err)
	}
	if !watchRoots[resolvedAuthModuleDir] || !watchRoots[resolvedBaseModuleDir] {
		t.Fatalf("unexpected watch target roots: %#v", watchTargets)
	}
	if !watchModules["auth"] || !watchModules["base"] {
		t.Fatalf("unexpected watch target modules: %#v", watchTargets)
	}
	watchList := watcher.WatchList()
	if len(watchList) != 2 {
		t.Fatalf("watcher list len = %d, want 2 (%#v)", len(watchList), watchList)
	}
}

func TestServerRegisterApplicationServicesReturnsCreateServiceError(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Compile.BundleMode = "application"
	runtimeScope.cfg.DistPath = t.TempDir()
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		runState:     runState{applicationNames: []string{"missing"}},
	}

	if _, err := srv.registerApplicationServices(); err == nil {
		t.Fatal("expected registerApplicationServices to fail when application dist path is missing")
	}
}

func TestServerRegisterApplicationServicesBuildsWebWatchTargetWhenHasWeb(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true
	distDir := t.TempDir()
	modulesDir := t.TempDir()
	runtimeScope.cfg.DistPath = distDir
	runtimeScope.cfg.ModulesPath = modulesDir

	webDistDir := filepath.Join(distDir, "web")
	if err := os.MkdirAll(webDistDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(webDistDir) error = %v", err)
	}
	webModuleDir := filepath.Join(modulesDir, "web")
	if err := os.MkdirAll(webModuleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(webModuleDir) error = %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		hotreload:    hotreloadState{watcher: watcher},
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		runState: runState{
			applicationNames: []string{"web"},
			distManifest: &distmanifest.DistManifestV2{
				HasWeb: true,
				Apps:   map[string]distmanifest.DistManifestApp{},
			},
		},
	}

	registration, err := srv.registerApplicationServices()
	if err != nil {
		t.Fatalf("registerApplicationServices() error = %v", err)
	}
	assertRegistrationInitScripts(t, registration, []string{}, "registerApplicationServices() web init scripts")
	requireSingleRegistrationBinding(t, registration, "registerApplicationServices() web result bindings")

	watchTargets := srv.hotreload.watchTargetsSnapshot()
	if len(watchTargets) != 1 {
		t.Fatalf("watch target len = %d, want 1 (%#v)", len(watchTargets), watchTargets)
	}
	resolvedWebModuleDir, err := resolveWatchPath(webModuleDir)
	if err != nil {
		t.Fatalf("resolveWatchPath(webModuleDir) error = %v", err)
	}
	if watchTargets[0].moduleName != "web" || watchTargets[0].root != resolvedWebModuleDir {
		t.Fatalf("unexpected web watch target: %#v", watchTargets)
	}
	watchList := watcher.WatchList()
	if len(watchList) != 1 {
		t.Fatalf("watcher list len = %d, want 1 (%#v)", len(watchList), watchList)
	}
}

func TestServerRegisterApplicationServicesSkipsWebWatchTargetWhenHasWebFalse(t *testing.T) {
	runtimeScope := newRichServerTestScope(t)
	runtimeScope.cfg.Server.HotReload = true
	distDir := t.TempDir()
	modulesDir := t.TempDir()
	runtimeScope.cfg.DistPath = distDir
	runtimeScope.cfg.ModulesPath = modulesDir

	webDistDir := filepath.Join(distDir, "web")
	if err := os.MkdirAll(webDistDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(webDistDir) error = %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Close()

	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		server:       grpc.NewServer(),
		mux:          http.NewServeMux(),
		hotreload:    hotreloadState{watcher: watcher},
		address:      &resolver.Address{Addr: "127.0.0.1:9527"},
		runState: runState{
			applicationNames: []string{"web"},
			distManifest: &distmanifest.DistManifestV2{
				HasWeb: false,
				Apps:   map[string]distmanifest.DistManifestApp{},
			},
		},
	}

	if _, err := srv.registerApplicationServices(); err != nil {
		t.Fatalf("registerApplicationServices() error = %v", err)
	}
	watchTargets := srv.hotreload.watchTargetsSnapshot()
	if len(watchTargets) != 0 {
		t.Fatalf("watch target len = %d, want 0 (%#v)", len(watchTargets), watchTargets)
	}
	watchList := watcher.WatchList()
	if len(watchList) != 0 {
		t.Fatalf("watcher list len = %d, want 0 (%#v)", len(watchList), watchList)
	}
}
