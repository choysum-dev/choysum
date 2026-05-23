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
		s.runtimeScope.Logger().Info("bootstrap mode selected", "reason", s.runState.reason())
	}

	return nil
}
