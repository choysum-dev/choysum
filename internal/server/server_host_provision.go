// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	xfmt "golang.org/x/exp/errors/fmt"
)

// ensureHostProvision installs the minimal web-shell closure when the workspace
// is Empty, then re-plans so Serve can start in application mode (SetupPending).
// Bootstrap SPA remains available as a degraded fallback until PC-9.
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
		if s.runtimeScope.Session() == nil || s.runtimeScope.Session().DB == nil {
			return nil
		}
		return xfmt.Errorf("resolve setup phase before provision: %w", err)
	}
	if phase != provision.PhaseEmpty {
		return nil
	}

	result, err := provision.ProvisionMinimal(ctx, s.runtimeScope, provision.Options{})
	if err != nil {
		return xfmt.Errorf("host provision: %w", err)
	}
	if result.Skipped {
		return nil
	}

	if err := s.planServe(s.runState.serveRequest()); err != nil {
		return xfmt.Errorf("replan after host provision: %w", err)
	}
	if s.runState.isBootstrapMode() {
		s.runtimeScope.Logger().Warn(
			"host provision completed but runplan still selected bootstrap mode",
			"phase", string(result.Phase),
			"reason", s.runState.reason(),
		)
		return xfmt.Errorf(
			"host provision completed (phase=%s) but application dist is not ready: %s",
			result.Phase,
			s.runState.reason(),
		)
	}
	if s.runState.mode() != runplan.RunModeApplication {
		return xfmt.Errorf("host provision completed but unexpected run mode %q", s.runState.mode())
	}
	return nil
}
