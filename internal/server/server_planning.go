// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import "github.com/choysum-dev/choysum/internal/server/runplan"

func (s *GRPCWebServer) planServe(serviceNames []string) error {
	opts := s.resolvedRuntimeOptions()
	manifest, decision, err := runplan.Plan(opts.distPath, opts.compileBundleMode, s.runtimeScope.Logger(), serviceNames)
	if err != nil {
		return err
	}

	s.runState.applyPlannedDecision(manifest, decision)
	if s.runState.isBootstrapMode() {
		// Degraded path when application dist is not ready (e.g. pre-PC-8 empty
		// workspaces before ProvisionMinimal, or a broken dist after provision).
		s.runtimeScope.Logger().Info("bootstrap mode fallback selected", "reason", s.runState.reason())
	}

	return nil
}
