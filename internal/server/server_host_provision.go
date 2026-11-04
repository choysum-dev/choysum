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

// hostProvisionPeerWaitNanos is the peer-wait budget (nanoseconds). Tests override
// via store/load so parallel packages cannot race a bare duration var.
var hostProvisionPeerWaitNanos atomic.Int64

func init() {
	hostProvisionPeerWaitNanos.Store(int64(2 * time.Minute))
}

func hostProvisionPeerWait() time.Duration {
	return time.Duration(hostProvisionPeerWaitNanos.Load())
}

func setHostProvisionPeerWait(d time.Duration) {
	hostProvisionPeerWaitNanos.Store(int64(d))
}

// prepareHostProvisionForServe decides whether Empty workspaces defer install
// until after splash listening. Returns deferred=true when Serve should start
// bootstrap mode immediately and run provision in the background.
func (s *GRPCWebServer) prepareHostProvisionForServe(ctx context.Context) (deferred bool, err error) {
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
		return false, s.ensureHostProvision(ctx)
	}
	if !hasLocalProvisionModules(s.runtimeScope) {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn("host provision skipped: ModulesPath has no local modules; continuing in bootstrap mode")
		return false, nil
	}
	s.setHostProvisionUI(hostProvisionUIProvisioning, setupStatusMsgStarting)
	return true, nil
}

// deferredHostProvision runs after splash listening so browsers see progress
// immediately. On success it restarts into application mode (SetupPending).
func (s *GRPCWebServer) deferredHostProvision(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.ensureHostProvision(ctx); err != nil {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn("deferred host provision failed", "error", err)
		return
	}
	uiPhase, _ := s.hostProvisionUISnapshot()
	if uiPhase == hostProvisionUIFailed {
		return
	}
	// Promote bootstrap splash → application shell without tearing down Serve's
	// outer loop (restart re-plans from dist after install).
	if err := s.restart(); err != nil {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn("deferred host provision restart failed", "error", err)
		return
	}
	s.setHostProvisionUI(hostProvisionUIIdle, "")
}

// ensureHostProvision installs the minimal web-shell closure when the workspace
// is Empty, then re-plans so Serve can start in application mode (SetupPending).
// Install failures and missing application dist fall back to bootstrap mode
// (host pending splash; no /bootstrap SPA).
func (s *GRPCWebServer) ensureHostProvision(ctx context.Context) error {
	if s == nil || s.runtimeScope == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	phase, err := provision.Resolve(s.runtimeScope)
	if err != nil {
		// Test fixtures and pre-DB scopes have no session; skip host provision.
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
			return s.replanAfterHostProvision(false)
		}
		return nil
	}

	// Automatic first-run provision requires a local ModulesPath closure.
	// Empty temp dirs would otherwise fall through to registry and leave a
	// half-applied schema when published packages fail to build.
	if !hasLocalProvisionModules(s.runtimeScope) {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn("host provision skipped: ModulesPath has no local modules; continuing in bootstrap mode")
		return nil
	}

	s.setHostProvisionUI(hostProvisionUIProvisioning, setupStatusMsgStarting)
	result, err := s.runHostProvisionMinimal(ctx)
	if err != nil {
		if errors.Is(err, provision.ErrLeaseBusy) {
			s.setHostProvisionUI(hostProvisionUIWaitingPeer, setupStatusMsgWaiting)
			if waitErr := s.waitForHostProvisionPeer(ctx); waitErr != nil {
				s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
				s.logHostProvisionWarn(
					"host provision lease busy; continuing in bootstrap mode (restart after peer finishes if still pending)",
					"error", waitErr,
				)
				// Soft-replan anyway: peer may have finished while wait timed out
				// or Resolve transiently failed during the poll loop.
				_ = s.replanAfterHostProvision(false)
				return nil
			}
			if phase, resolveErr := provision.Resolve(s.runtimeScope); resolveErr == nil {
				s.logHostProvisionWarn("peer host provision finished", "phase", string(phase))
			}
			// Soft-fallback: peer path never hard-fails Serve on replan.
			_ = s.replanAfterHostProvision(false)
			return nil
		}
		if errors.Is(err, provision.ErrLeaseLost) {
			s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
			s.logHostProvisionWarn(
				"host provision lease lost mid-install; workspace may be partially provisioned; continuing in bootstrap mode",
				"error", err,
			)
			return nil
		}
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn("host provision failed; continuing in bootstrap mode", "error", err)
		return nil
	}
	if result.Skipped {
		return s.replanAfterHostProvision(true)
	}

	if err := s.replanAfterHostProvision(true); err != nil {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		return err
	}
	if s.runState.isBootstrapMode() {
		s.setHostProvisionUI(hostProvisionUIFailed, setupStatusMsgFailed)
		s.logHostProvisionWarn(
			"host provision completed but application dist is not ready; continuing in bootstrap mode",
			"phase", string(result.Phase),
			"reason", s.runState.reason(),
		)
	}
	return nil
}

func (s *GRPCWebServer) runHostProvisionMinimal(ctx context.Context) (provision.Result, error) {
	if s.hostProvisionFn != nil {
		return s.hostProvisionFn(ctx)
	}
	return provision.ProvisionMinimal(ctx, s.runtimeScope, provision.Options{})
}

func (s *GRPCWebServer) replanAfterHostProvision(required bool) error {
	err := s.planServe(s.runState.serveRequest())
	if err == nil {
		return nil
	}
	if !required {
		s.logHostProvisionWarn("replan after host provision skipped", "error", err)
		return nil
	}
	return xfmt.Errorf("replan after host provision: %w", err)
}

func (s *GRPCWebServer) waitForHostProvisionPeer(ctx context.Context) error {
	wait := hostProvisionPeerWait()
	s.logHostProvisionWarn("host provision lease busy; waiting for peer provision", "timeout", wait.String())
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
			return errors.New("timed out waiting for peer host provision")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *GRPCWebServer) logHostProvisionWarn(msg string, args ...any) {
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
