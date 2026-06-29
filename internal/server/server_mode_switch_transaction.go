// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/internal/server/transport"
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

	s.runtimeScope.Logger().Info("application server ready", s.applicationServerReadyLogFields()...)

	return nil
}

func (s *GRPCWebServer) applicationServerReadyLogFields() []any {
	configuredAddress := ""
	if s != nil && s.address != nil {
		configuredAddress = strings.TrimSpace(s.address.Addr)
	}
	listenAddress := configuredAddress
	if s != nil && s.listener != nil && s.listener.Addr() != nil {
		listenAddress = strings.TrimSpace(s.listener.Addr().String())
	}
	if listenAddress == "" {
		return nil
	}

	scheme := "http"
	if s != nil && s.resolvedRuntimeOptions().enabledTLS {
		scheme = "https"
	}
	fields := []any{"address", listenAddress}
	if accessURL := transport.HTTPServerAccessURL(configuredAddress, listenAddress, scheme); accessURL != "" {
		fields = append(fields, "access_url", accessURL)
	}
	return fields
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
