// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"net/http"
	"sync"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc"
)

// pendingSplashService is the host-only HTTP surface for RunModeBootstrap after
// the dual frontend (internal/bootstrap/web + Workspace gRPC) was removed.
// It does not expose /bootstrap or any install RPC — Empty workspaces are
// provisioned by ensureHostProvision before Serve.
type pendingSplashService struct{}

func newPendingSplashService() *pendingSplashService {
	return &pendingSplashService{}
}

// pendingSplashBindingFn builds the bootstrap-mode registration binding.
// Tests override it via setPendingSplashBindingForTest to force failures.
var (
	pendingSplashBindingMu sync.RWMutex
	pendingSplashBindingFn = func() registrationBinding {
		return newPendingSplashService()
	}
)

func pendingSplashBinding() registrationBinding {
	pendingSplashBindingMu.RLock()
	defer pendingSplashBindingMu.RUnlock()
	return pendingSplashBindingFn()
}

func setPendingSplashBindingForTest(fn func() registrationBinding) (restore func()) {
	pendingSplashBindingMu.Lock()
	prev := pendingSplashBindingFn
	pendingSplashBindingFn = fn
	pendingSplashBindingMu.Unlock()
	return func() {
		pendingSplashBindingMu.Lock()
		pendingSplashBindingFn = prev
		pendingSplashBindingMu.Unlock()
	}
}

func (s *pendingSplashService) Name() string { return "bootstrap" }

func (s *pendingSplashService) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	return nil, nil
}

func (s *pendingSplashService) ServiceScripts() []*jsengine.JsScript { return nil }

func (s *pendingSplashService) WebHandlers() (map[string]http.Handler, error) {
	// Exact "/" only (Go 1.22+ {$}); do not claim the "/" tree catch-all so
	// /healthz and other base routes keep mux precedence, and stale
	// /bootstrap/* bookmarks get the mux 404.
	return map[string]http.Handler{
		"/{$}": http.HandlerFunc(servePendingSplash),
	}, nil
}

func servePendingSplash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusServiceUnavailable)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(pendingSplashHTML))
}

const pendingSplashHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Choysum</title>
<style>
  :root { color-scheme: light; }
  body {
    margin: 0;
    min-height: 100vh;
    display: grid;
    place-items: center;
    font-family: "Iowan Old Style", "Palatino Linotype", Palatino, Georgia, serif;
    background:
      radial-gradient(ellipse 80% 50% at 20% 0%, #d9e8f2 0%, transparent 55%),
      radial-gradient(ellipse 70% 45% at 90% 100%, #c5d4c8 0%, transparent 50%),
      #eef2f4;
    color: #152028;
  }
  main { max-width: 28rem; padding: 2rem; text-align: center; }
  h1 { font-size: 1.75rem; font-weight: 600; margin: 0 0 0.75rem; letter-spacing: -0.03em; }
  p { margin: 0; line-height: 1.55; color: #3d4a55; font-family: "Avenir Next", "Segoe UI", sans-serif; font-size: 0.95rem; }
</style>
</head>
<body>
<main>
  <h1>Choysum</h1>
  <p>Application assets are not ready yet. Check the terminal for host provision progress, then restart the server (or re-run <code>choysum run</code>) and reload.</p>
</main>
</body>
</html>
`
