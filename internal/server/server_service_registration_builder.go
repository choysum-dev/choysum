// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"path/filepath"
	"strings"

	internalservice "github.com/choysum-dev/choysum/internal/service"
	xfmt "golang.org/x/exp/errors/fmt"
)

type registrationWatchPlan struct {
	ServiceName string
	ModuleName  string
	Root        string
}

type applicationServicePlan struct {
	Name    string
	Watches []registrationWatchPlan
}

type applicationServiceBuilder struct {
	server *GRPCWebServer
	opts   runtimeOptions
}

func (s *GRPCWebServer) newApplicationServiceBuilder() applicationServiceBuilder {
	return applicationServiceBuilder{server: s, opts: s.resolvedRuntimeOptions()}
}

func (b applicationServiceBuilder) plans() []applicationServicePlan {
	targets := b.server.runState.serviceTargets()
	plans := make([]applicationServicePlan, 0, len(targets))
	for _, target := range targets {
		plans = append(plans, b.plan(target))
	}
	return plans
}

func (b applicationServiceBuilder) plan(target string) applicationServicePlan {
	return applicationServicePlan{
		Name:    target,
		Watches: b.watchPlans(target),
	}
}

func (b applicationServiceBuilder) build(plan applicationServicePlan) (*internalservice.ApplicationService, error) {
	svc, err := internalservice.NewApplicationService(
		b.server.runtimeScope,
		plan.Name,
		b.server.JSExecutor(),
		internalservice.WithHasGrpcMethod(b.server.hasGrpcMethod),
		internalservice.WithBundleMode(b.server.runState.bundleMode()),
	)
	if err != nil {
		return nil, xfmt.Errorf("Failed to create application service: %w", err)
	}
	return svc, nil
}

func (b applicationServiceBuilder) watchPlans(target string) []registrationWatchPlan {
	app, ok := b.server.runState.manifestApp(target)
	if !ok {
		return nil
	}
	return buildApplicationWatchPlans(target, b.opts.modulesPath, app.Dev.Modules)
}

func buildApplicationWatchPlans(serviceName string, modulesPath string, modules []string) []registrationWatchPlan {
	seen := map[string]bool{}
	plans := make([]registrationWatchPlan, 0, len(modules))
	for _, moduleName := range modules {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" || seen[moduleName] {
			continue
		}
		seen[moduleName] = true
		plans = append(plans, registrationWatchPlan{
			ServiceName: serviceName,
			ModuleName:  moduleName,
			Root:        filepath.Join(modulesPath, moduleName),
		})
	}
	return plans
}

func (s *GRPCWebServer) registerApplicationBinding(batch *registrationBatch, binding registrationBinding) error {
	if err := batch.registerBinding(s, binding); err != nil {
		return xfmt.Errorf("Failed to register service: %w", err)
	}
	s.runtimeScope.Logger().Debug("application service registered", "service", binding.Name())
	return nil
}
