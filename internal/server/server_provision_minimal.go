// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// provisionMinimalPeerWaitNanos is the peer-wait budget (nanoseconds). Tests override
// via store/load so parallel packages cannot race a bare duration var.
var provisionMinimalPeerWaitNanos atomic.Int64

func init() {
	provisionMinimalPeerWaitNanos.Store(int64(2 * time.Minute))
}

func provisionMinimalPeerWait() time.Duration {
	return time.Duration(provisionMinimalPeerWaitNanos.Load())
}

func setProvisionMinimalPeerWait(d time.Duration) {
	provisionMinimalPeerWaitNanos.Store(int64(d))
}

// prepareProvisionMinimalForServe decides whether Empty workspaces defer install
// until after splash listening. Returns deferred=true when Serve should start
// bootstrap mode immediately and run provision in the background.
func (s *GRPCWebServer) prepareProvisionMinimalForServe(ctx context.Context) (deferred bool, err error) {
	if s == nil || s.runtimeScope == nil {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	phase, resolveErr := provision.Resolve(s.runtimeScope)
	if resolveErr != nil {
		session := s.runtimeScope.Session()
		if session == nil || session.DB == nil {
			return false, nil
		}
		return false, xfmt.Errorf("resolve setup phase before provision: %w", resolveErr)
	}
	if phase != provision.PhaseEmpty {
		// Peer may have finished while we were still on splash; refresh plan later via Serve.
		return false, s.ensureProvisionMinimal(ctx)
	}
	if !hasLocalProvisionModules(s.runtimeScope) {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn("ProvisionMinimal skipped: ModulesPath has no local modules; continuing in bootstrap mode")
		return false, nil
	}
	s.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	return true, nil
}

// deferredProvisionMinimal runs after splash listening so browsers see progress
// immediately. On success it restarts into application mode (SetupPending).
func (s *GRPCWebServer) deferredProvisionMinimal(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.ensureProvisionMinimal(ctx); err != nil {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn("deferred ProvisionMinimal failed", "error", err)
		return
	}
	uiPhase, _ := s.setupSplashUISnapshot()
	if uiPhase == setupSplashUIFailed {
		return
	}
	// Promote bootstrap splash → application shell without tearing down Serve's
	// outer loop (restart re-plans from dist after install).
	var restartErr error
	if s.deferredRestartFn != nil {
		restartErr = s.deferredRestartFn()
	} else {
		restartErr = s.restart()
	}
	if restartErr != nil {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn("deferred ProvisionMinimal restart failed", "error", restartErr)
		return
	}
	s.setSetupSplashUI(setupSplashUIIdle, "")
}

// kickDeferredProvisionMinimalIfNeeded starts background ProvisionMinimal once
// the bootstrap listener is up. Serve sets the flag; start failures clear it.
func (s *GRPCWebServer) kickDeferredProvisionMinimalIfNeeded() {
	if s == nil || !s.startDeferredProvisionMinimal {
		return
	}
	s.startDeferredProvisionMinimal = false
	ctx := context.Background()
	if s.runtimeScope != nil && s.runtimeScope.Context() != nil {
		ctx = s.runtimeScope.Context()
	}
	go s.deferredProvisionMinimal(ctx)
}

// ensureProvisionMinimal installs the minimal web-shell closure when the workspace
// is Empty, then re-plans so Serve can start in application mode (SetupPending).
// Install failures and missing application dist fall back to bootstrap mode
// (setup splash; no /bootstrap SPA).
func (s *GRPCWebServer) ensureProvisionMinimal(ctx context.Context) error {
	if s == nil || s.runtimeScope == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	phase, err := provision.Resolve(s.runtimeScope)
	if err != nil {
		// Test fixtures and pre-DB scopes have no session; skip ProvisionMinimal.
		session := s.runtimeScope.Session()
		if session == nil || session.DB == nil {
			return nil
		}
		return xfmt.Errorf("resolve setup phase before provision: %w", err)
	}
	if phase != provision.PhaseEmpty {
		// Another actor may have finished provision after the first planServe;
		// refresh the run mode when we are still on the bootstrap fallback.
		if s.runState.isBootstrapMode() {
			return s.replanAfterProvisionMinimal(false)
		}
		return nil
	}

	// Automatic ProvisionMinimal requires a local ModulesPath closure.
	// Empty temp dirs would otherwise fall through to registry and leave a
	// half-applied schema when published packages fail to build.
	if !hasLocalProvisionModules(s.runtimeScope) {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn("ProvisionMinimal skipped: ModulesPath has no local modules; continuing in bootstrap mode")
		return nil
	}

	s.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
	result, err := s.runProvisionMinimal(ctx)
	if err != nil {
		if errors.Is(err, provision.ErrLeaseBusy) {
			s.setSetupSplashUI(setupSplashUIWaitingPeer, setupStatusMsgWaiting)
			if waitErr := s.waitForProvisionMinimalPeer(ctx); waitErr != nil {
				s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
				s.logProvisionMinimalWarn(
					"ProvisionMinimal lease busy; continuing in bootstrap mode (restart after peer finishes if still pending)",
					"error", waitErr,
				)
				// Soft-replan anyway: peer may have finished while wait timed out
				// or Resolve transiently failed during the poll loop.
				_ = s.replanAfterProvisionMinimal(false)
				return nil
			}
			if phase, resolveErr := provision.Resolve(s.runtimeScope); resolveErr == nil {
				s.logProvisionMinimalWarn("peer ProvisionMinimal finished", "phase", string(phase))
			}
			// Soft-fallback: peer path never hard-fails Serve on replan.
			_ = s.replanAfterProvisionMinimal(false)
			// Clear waiting_peer so status is not stuck on "waiting" after the
			// peer finished; deferredProvisionMinimal then restart()+idle.
			s.setSetupSplashUI(setupSplashUIProvisioning, setupStatusMsgStarting)
			return nil
		}
		if errors.Is(err, provision.ErrLeaseLost) {
			s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
			s.logProvisionMinimalWarn(
				"ProvisionMinimal lease lost mid-install; workspace may be partially provisioned; continuing in bootstrap mode",
				"error", err,
			)
			return nil
		}
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn("ProvisionMinimal failed; continuing in bootstrap mode", "error", err)
		return nil
	}
	if result.Skipped {
		return s.replanAfterProvisionMinimal(true)
	}

	if err := s.replanAfterProvisionMinimal(true); err != nil {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		return err
	}
	if s.runState.isBootstrapMode() {
		s.setSetupSplashUI(setupSplashUIFailed, setupStatusMsgFailed)
		s.logProvisionMinimalWarn(
			"ProvisionMinimal completed but application dist is not ready; continuing in bootstrap mode",
			"phase", string(result.Phase),
			"reason", s.runState.reason(),
		)
	}
	return nil
}

func (s *GRPCWebServer) runProvisionMinimal(ctx context.Context) (provision.Result, error) {
	if s.provisionMinimalFn != nil {
		return s.provisionMinimalFn(ctx)
	}
	return provision.ProvisionMinimal(ctx, s.runtimeScope, provision.Options{})
}

func (s *GRPCWebServer) replanAfterProvisionMinimal(required bool) error {
	err := s.planServe(s.runState.serveRequest())
	if err == nil {
		return nil
	}
	if !required {
		s.logProvisionMinimalWarn("replan after ProvisionMinimal skipped", "error", err)
		return nil
	}
	return xfmt.Errorf("replan after ProvisionMinimal: %w", err)
}

func (s *GRPCWebServer) waitForProvisionMinimalPeer(ctx context.Context) error {
	wait := provisionMinimalPeerWait()
	s.logProvisionMinimalWarn("ProvisionMinimal lease busy; waiting for peer provision", "timeout", wait.String())
	deadline := time.Now().Add(wait)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		phase, err := provision.Resolve(s.runtimeScope)
		if err != nil {
			return err
		}
		if phase != provision.PhaseEmpty {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for peer ProvisionMinimal")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *GRPCWebServer) logProvisionMinimalWarn(msg string, args ...any) {
	if s == nil || s.runtimeScope == nil {
		return
	}
	if log := s.runtimeScope.Logger(); log != nil {
		log.Warn(msg, args...)
	}
}

func hasLocalProvisionModules(runtimeScope scope.Scope) bool {
	if runtimeScope == nil {
		return false
	}
	paths, ok := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	if !ok {
		return false
	}
	modulesPath := strings.TrimSpace(paths.ModulesPath)
	if modulesPath == "" {
		return false
	}
	// Require the full local closure so provision does not start and fail
	// mid-install when a dependency package is missing from ModulesPath.
	for _, name := range provision.MinimalClosureModules() {
		info, err := os.Stat(filepath.Join(modulesPath, name, "package.json"))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}
