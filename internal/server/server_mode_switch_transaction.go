// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	xfmt "golang.org/x/exp/errors/fmt"
)

type modeSwitchState struct {
	bootstrapMu sync.Mutex
}

type bootstrapModeSwitchPlan struct {
	Manifest *distmanifest.DistManifestV2
	Decision runplan.RunDecision
}

type bootstrapModeSwitchResult struct {
	Switched bool
	Mode     runplan.RunMode
	Reason   string
	Targets  []string
}

func (s *GRPCWebServer) executeBootstrapModeSwitch(ctx context.Context) (bootstrapModeSwitchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return bootstrapModeSwitchResult{}, xfmt.Errorf("bootstrap switch canceled: %w", err)
	}

	s.modeSwitch.bootstrapMu.Lock()
	defer s.modeSwitch.bootstrapMu.Unlock()

	if !s.runState.isBootstrapMode() {
		return bootstrapModeSwitchResult{}, nil
	}

	plan, err := s.planBootstrapModeSwitch()
	if err != nil {
		return bootstrapModeSwitchResult{}, err
	}

	return s.executeBootstrapModeSwitchPlan(plan)
}

func (s *GRPCWebServer) planBootstrapModeSwitch() (bootstrapModeSwitchPlan, error) {
	opts := s.resolvedRuntimeOptions()
	manifest, decision, err := runplan.Plan(opts.distPath, opts.compileBundleMode, s.runtimeScope.Logger(), nil)
	if err != nil {
		return bootstrapModeSwitchPlan{}, xfmt.Errorf("bootstrap switch failed to resolve application mode: %w", err)
	}
	if decision.RunMode != runplan.RunModeApplication {
		return bootstrapModeSwitchPlan{}, xfmt.Errorf("bootstrap switch target mode is %q", decision.RunMode)
	}
	return bootstrapModeSwitchPlan{Manifest: manifest, Decision: decision}, nil
}

func (s *GRPCWebServer) requestBootstrapModeSwitch(ctx context.Context) error {
	result, err := s.executeBootstrapModeSwitch(ctx)
	if err != nil {
		return err
	}
	if !result.Switched {
		return nil
	}

	s.runtimeScope.Logger().Info(
		"bootstrap runtime mode switched",
		result.logFields()...,
	)

	return nil
}

func (r bootstrapModeSwitchResult) logFields() []any {
	if !r.Switched {
		return nil
	}
	return []any{
		"mode", string(r.Mode),
		"reason", r.Reason,
		"targets", strings.Join(r.Targets, ","),
	}
}
