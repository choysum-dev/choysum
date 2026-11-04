// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/choysum-dev/choysum/internal/provision"
)

// Host-visible provision wait states for the splash page (not setup.completed_at).
type setupSplashUIPhase string

const (
	setupSplashUIIdle         setupSplashUIPhase = "idle"
	setupSplashUIProvisioning setupSplashUIPhase = "provisioning"
	setupSplashUIWaitingPeer  setupSplashUIPhase = "waiting_peer"
	setupSplashUIFailed       setupSplashUIPhase = "failed"
)

// Splash / setup-status copy (English only; not i18n).
const (
	setupStatusMsgStarting = "Preparing setup…"
	setupStatusMsgWaiting  = "Setup is already being prepared…"
	setupStatusMsgFailed   = "Could not prepare setup. Check the terminal, then run again."
	setupStatusMsgReady    = "Opening setup…"
)

type setupSplashUIState struct {
	mu      sync.RWMutex
	phase   setupSplashUIPhase
	message string
}

func (s *GRPCWebServer) setSetupSplashUI(phase setupSplashUIPhase, message string) {
	if s == nil {
		return
	}
	s.setupSplashUI.mu.Lock()
	defer s.setupSplashUI.mu.Unlock()
	s.setupSplashUI.phase = phase
	s.setupSplashUI.message = message
}

func (s *GRPCWebServer) setupSplashUISnapshot() (setupSplashUIPhase, string) {
	if s == nil {
		return setupSplashUIIdle, ""
	}
	s.setupSplashUI.mu.RLock()
	defer s.setupSplashUI.mu.RUnlock()
	phase := s.setupSplashUI.phase
	if phase == "" {
		phase = setupSplashUIIdle
	}
	return phase, s.setupSplashUI.message
}

type setupStatusResponse struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

func (s *GRPCWebServer) serveSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	resp := s.setupStatusPayload()
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *GRPCWebServer) setupStatusPayload() setupStatusResponse {
	uiPhase, uiMsg := s.setupSplashUISnapshot()
	// Prefer in-flight splash UI over DB phase: ProvisionMinimal flips the workspace
	// to SetupPending before restart() registers /web/, and advertising ready
	// here would send the splash to a 404.
	switch uiPhase {
	case setupSplashUIProvisioning:
		if uiMsg == "" {
			uiMsg = setupStatusMsgStarting
		}
		return setupStatusResponse{Phase: "starting", Message: uiMsg}
	case setupSplashUIWaitingPeer:
		if uiMsg == "" {
			uiMsg = setupStatusMsgWaiting
		}
		return setupStatusResponse{Phase: "waiting", Message: uiMsg}
	case setupSplashUIFailed:
		if uiMsg == "" {
			uiMsg = setupStatusMsgFailed
		}
		return setupStatusResponse{Phase: "failed", Message: uiMsg}
	}

	if s != nil && s.runtimeScope != nil {
		if phase, err := provision.Resolve(s.runtimeScope); err == nil {
			switch phase {
			case provision.PhaseSetupPending, provision.PhaseReady:
				return setupStatusResponse{
					Phase:   "ready",
					Message: setupStatusMsgReady,
				}
			}
		}
	}

	// Idle + Empty (or unresolved): keep the waiting face, never expose "idle".
	return setupStatusResponse{
		Phase:   "starting",
		Message: setupStatusMsgStarting,
	}
}
