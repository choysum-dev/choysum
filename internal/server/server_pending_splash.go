// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"net/http"
	"sync"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc"
)

// pendingSplashService is the host-only HTTP surface for RunModeBootstrap.
// It does not expose /bootstrap or any install RPC — Empty workspaces show this
// splash while deferredHostProvision installs the minimal shell.
type pendingSplashService struct {
	server *GRPCWebServer
}

func newPendingSplashService(server *GRPCWebServer) *pendingSplashService {
	return &pendingSplashService{server: server}
}

// pendingSplashBindingFn builds the bootstrap-mode registration binding.
// Tests override it via setPendingSplashBindingForTest to force failures.
var (
	pendingSplashBindingMu sync.RWMutex
	pendingSplashBindingFn = func(server *GRPCWebServer) registrationBinding {
		return newPendingSplashService(server)
	}
)

func pendingSplashBinding(server *GRPCWebServer) registrationBinding {
	pendingSplashBindingMu.RLock()
	defer pendingSplashBindingMu.RUnlock()
	return pendingSplashBindingFn(server)
}

func setPendingSplashBindingForTest(fn func(*GRPCWebServer) registrationBinding) (restore func()) {
	pendingSplashBindingMu.Lock()
	prev := pendingSplashBindingFn
	if fn == nil {
		pendingSplashBindingFn = func(server *GRPCWebServer) registrationBinding {
			return newPendingSplashService(server)
		}
	} else {
		pendingSplashBindingFn = fn
	}
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
	// /healthz, /-/setup/status, and other base routes keep precedence.
	return map[string]http.Handler{
		"/{$}": http.HandlerFunc(s.servePendingSplash),
	}, nil
}

func (s *pendingSplashService) servePendingSplash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	status := http.StatusOK
	payload := setupStatusResponse{
		Phase:   "starting",
		Message: setupStatusMsgStarting,
	}
	if s != nil && s.server != nil {
		payload = s.server.setupStatusPayload()
	}
	if payload.Phase == "failed" {
		status = http.StatusServiceUnavailable
	}
	w.WriteHeader(status)
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
    /* Near shell white + faint emerald wash (product primary hue ~160). */
    background:
      radial-gradient(ellipse 70% 45% at 15% 0%, rgba(16, 185, 129, 0.08) 0%, transparent 55%),
      radial-gradient(ellipse 60% 40% at 95% 100%, rgba(16, 185, 129, 0.06) 0%, transparent 50%),
      #ffffff;
    color: #152028;
    transition: opacity 220ms ease;
  }
  body.is-leaving { opacity: 0; }
  main { max-width: 28rem; padding: 2rem; text-align: center; }
  h1 {
    font-size: 1.75rem;
    font-weight: 600;
    margin: 0 0 0.75rem;
    letter-spacing: -0.03em;
  }
  #status {
    margin: 0 0 1.25rem;
    line-height: 1.55;
    color: #3d4a55;
    font-family: "Avenir Next", "Segoe UI", sans-serif;
    font-size: 0.95rem;
  }
  .progress {
    width: min(14rem, 70vw);
    height: 3px;
    margin: 0 auto;
    border-radius: 999px;
    background: rgba(16, 185, 129, 0.15);
    overflow: hidden;
  }
  .progress > span {
    display: block;
    height: 100%;
    width: 40%;
    border-radius: inherit;
    background: #059669;
    animation: indeterminate 1.35s ease-in-out infinite;
  }
  .retry {
    display: none;
    margin-top: 1.25rem;
    font-family: "Avenir Next", "Segoe UI", sans-serif;
    font-size: 0.9rem;
  }
  .retry button {
    appearance: none;
    border: 0;
    background: transparent;
    color: #047857;
    font: inherit;
    text-decoration: underline;
    text-underline-offset: 0.15em;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
  }
  .retry button:hover { color: #065f46; }
  body.is-failed .progress { display: none; }
  body.is-failed .retry { display: block; }
  @keyframes indeterminate {
    0% { transform: translateX(-120%); }
    100% { transform: translateX(320%); }
  }
</style>
</head>
<body>
<main>
  <h1>Choysum</h1>
  <p id="status" aria-live="polite">Preparing setup…</p>
  <div class="progress" aria-hidden="true"><span></span></div>
  <div class="retry">
    <button type="button" id="retry">Refresh</button>
  </div>
</main>
<script>
(function () {
  var statusEl = document.getElementById("status");
  var retryEl = document.getElementById("retry");
  var leaving = false;
  if (retryEl) {
    retryEl.addEventListener("click", function () {
      window.location.reload();
    });
  }
  function goSetup() {
    if (leaving) return;
    leaving = true;
    document.body.classList.add("is-leaving");
    window.setTimeout(function () {
      window.location.assign("/web/setup");
    }, 220);
  }
  function apply(payload) {
    if (!payload || typeof payload !== "object") return;
    var phase = String(payload.phase || "");
    var message = String(payload.message || "");
    // Ready: fade without flashing "Opening setup…".
    if (phase === "ready") {
      goSetup();
      return;
    }
    if (message && statusEl) statusEl.textContent = message;
    var failed = phase === "failed";
    document.body.classList.toggle("is-failed", failed);
    if (statusEl) statusEl.setAttribute("aria-live", failed ? "assertive" : "polite");
  }
  function tick() {
    if (leaving) return;
    fetch("/-/setup/status", { cache: "no-store" })
      .then(function (res) {
        // Do not treat 404/5xx as ready: restart briefly unbinds routes and
        // /web/setup is not registered until application mode finishes start.
        if (!res.ok) throw new Error("status " + res.status);
        return res.json();
      })
      .then(function (payload) {
        if (payload) apply(payload);
      })
      .catch(function () {
        // Transient blip during application restart; keep polling.
      })
      .finally(function () {
        if (!leaving) window.setTimeout(tick, 1000);
      });
  }
  tick();
})();
</script>
</body>
</html>
`
