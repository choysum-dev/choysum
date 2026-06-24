// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bootstrapweb "github.com/choysum-dev/choysum/internal/bootstrap/web"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestNewBootstrapServiceWithLockerFactoryUsesInjectedLocker(t *testing.T) {
	base, _ := newFreshnessTestCoordinator(t)
	locker := &coordinatorTestLocker{}

	bootstrapSvc, err := NewBootstrapService(base.runtimeScope,
		WithLockerFactory(func(scope.Scope) statepkg.Locker {
			return locker
		}),
	)
	if err != nil {
		t.Fatalf("NewBootstrapService() error = %v", err)
	}

	workspaceSrv, ok := bootstrapSvc.workspaceServer.(*workspaceServer)
	if !ok {
		t.Fatalf("workspaceServer type = %T, want *workspaceServer", bootstrapSvc.workspaceServer)
	}
	coordinator, ok := workspaceSrv.coordinator.(*coordinator)
	if !ok {
		t.Fatalf("coordinator type = %T, want *coordinator", workspaceSrv.coordinator)
	}

	handle, err := coordinator.defaultAcquireInitLease(context.Background())
	if err != nil {
		t.Fatalf("defaultAcquireInitLease() error = %v", err)
	}
	if handle == nil {
		t.Fatal("expected non-nil lease handle")
	}
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	coordinator.defaultReleaseInitLease(handle)
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}

	if bootstrapSvc.workspaceServer == nil {
		t.Fatal("expected workspaceServer to be initialized")
	}
}

func TestNewBootstrapWebHandlerDoesNotEmitInfoSourceSelectionLog(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")

	var logBuf bytes.Buffer
	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	handler, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		t.Fatalf("newBootstrapWebHandler() error = %v", err)
	}
	if handler == nil {
		t.Fatal("expected bootstrap web handler")
	}
	if logs := logBuf.String(); strings.Contains(logs, "bootstrap web source selected") {
		t.Fatalf("expected bootstrap web source detail log to be hidden at info level, got %q", logs)
	}
}

func TestBootstrapWebHandlerLogsFailedAssetRequests(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")

	var logBuf bytes.Buffer
	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}

	handler, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		t.Fatalf("newBootstrapWebHandler() error = %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/bootstrap/assets/missing.js", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected bootstrap missing asset response: code=%d body=%q", rr.Code, rr.Body.String())
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "web asset request failed") {
		t.Fatalf("expected bootstrap failed asset request log, got %q", logs)
	}
	if !strings.Contains(logs, "status=404") || !strings.Contains(logs, "path=/bootstrap/assets/missing.js") {
		t.Fatalf("expected bootstrap failed asset log attrs, got %q", logs)
	}
}

func TestBootstrapWebHandlerLogsSuccessfulAssetRequestsAtDebug(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")
	distFS, _, _, err := bootstrapweb.LoadDistFS("embed")
	if err != nil {
		t.Fatalf("LoadDistFS(embed) error = %v", err)
	}
	entries, err := fs.ReadDir(distFS, "assets")
	if err != nil {
		t.Fatalf("ReadDir(assets) error = %v", err)
	}
	assetName := ""
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		assetName = entry.Name()
		break
	}
	if assetName == "" {
		t.Fatal("expected embedded bootstrap assets")
	}

	var logBuf bytes.Buffer
	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}

	handler, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		t.Fatalf("newBootstrapWebHandler() error = %v", err)
	}

	rr := httptest.NewRecorder()
	assetPath := "/bootstrap/assets/" + assetName
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, assetPath, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected bootstrap asset response: code=%d body=%q", rr.Code, rr.Body.String())
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "web asset served") {
		t.Fatalf("expected bootstrap successful asset debug log, got %q", logs)
	}
	if !strings.Contains(logs, "status=200") || !strings.Contains(logs, "path="+assetPath) {
		t.Fatalf("expected bootstrap successful asset log attrs, got %q", logs)
	}
}

func TestBootstrapWebHandlerRedirectToCanonicalPath(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")

	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	handler, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		t.Fatalf("newBootstrapWebHandler() error = %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/bootstrap", nil))
	if rr.Code != http.StatusFound {
		t.Fatalf("GET /bootstrap status = %d, want %d", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/bootstrap/" {
		t.Fatalf("GET /bootstrap Location = %q, want /bootstrap/", loc)
	}
}

func TestBootstrapWebHandlerRedirectPreservesQueryParams(t *testing.T) {
	t.Setenv(bootstrapweb.EnvBootstrapWebSource, "embed")

	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	handler, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		t.Fatalf("newBootstrapWebHandler() error = %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/bootstrap?foo=bar&baz=1", nil))
	if rr.Code != http.StatusFound {
		t.Fatalf("GET /bootstrap?foo=bar&baz=1 status = %d, want %d", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/bootstrap/?foo=bar&baz=1" {
		t.Fatalf("GET /bootstrap?foo=bar&baz=1 Location = %q, want /bootstrap/?foo=bar&baz=1", loc)
	}
}

func TestBootstrapServiceWebHandlers(t *testing.T) {
	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	svc, err := NewBootstrapService(runtimeScope)
	if err != nil {
		t.Fatalf("NewBootstrapService() error = %v", err)
	}

	handlers, err := svc.WebHandlers()
	if err != nil {
		t.Fatalf("WebHandlers() error = %v", err)
	}
	if len(handlers) != 3 {
		t.Fatalf("WebHandlers() count = %d, want 3", len(handlers))
	}
	for _, path := range []string{"/", "/bootstrap", "/bootstrap/"} {
		if _, ok := handlers[path]; !ok {
			t.Fatalf("WebHandlers() missing path %q", path)
		}
	}

	rr := httptest.NewRecorder()
	handlers["/"].ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusFound {
		t.Fatalf("GET / status = %d, want %d", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/bootstrap/" {
		t.Fatalf("GET / Location = %q, want /bootstrap/", loc)
	}
}

func TestBootstrapServiceNameAndDescs(t *testing.T) {
	runtimeScope := &freshnessTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	svc, err := NewBootstrapService(runtimeScope)
	if err != nil {
		t.Fatalf("NewBootstrapService() error = %v", err)
	}

	if svc.Name() != "bootstrap" {
		t.Fatalf("Name() = %q, want bootstrap", svc.Name())
	}

	descs, err := svc.ServiceDescs()
	if err != nil {
		t.Fatalf("ServiceDescs() error = %v", err)
	}
	if len(descs) != 1 {
		t.Fatalf("ServiceDescs() count = %d, want 1", len(descs))
	}

	scripts := svc.ServiceScripts()
	if scripts != nil {
		t.Fatalf("ServiceScripts() = %#v, want nil", scripts)
	}
}
