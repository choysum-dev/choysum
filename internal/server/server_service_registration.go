// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"

	bootstrapservice "github.com/choysum-dev/choysum/internal/bootstrap/service"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) hasGrpcMethod(fullMethod string) bool {
	return s.registration.hasGrpcMethod(fullMethod)
}

func (s *GRPCWebServer) registerApplicationServices() (registrationBatchResult, error) {
	batch := s.registration.beginBatch()
	builder := s.newApplicationServiceBuilder()
	watchPlans := []registrationWatchPlan{}
	for _, plan := range builder.plans() {
		svc, err := builder.build(plan)
		if err != nil {
			return registrationBatchResult{}, err
		}
		if err := s.registerApplicationBinding(batch, svc); err != nil {
			return registrationBatchResult{}, err
		}
		watchPlans = append(watchPlans, plan.Watches...)
	}

	registration := batch.commit()
	if err := s.applyRegistrationWatchPlans(watchPlans); err != nil {
		return registrationBatchResult{}, err
	}
	return registration, nil
}

func (s *GRPCWebServer) unregisterAllServices() error {
	if s.registry == nil {
		return nil
	}
	if err := s.registry.UnRegisterAll(); err != nil {
		return xfmt.Errorf("Failed to unregister all services: %w", err)
	}
	return nil
}

func (s *GRPCWebServer) registerBootstrapService() (bootstrapModeStartupResult, error) {
	svc, err := bootstrapservice.NewBootstrapService(
		s.runtimeScope,
		bootstrapservice.WithSwitchModeFunc(s.requestBootstrapModeSwitch),
		bootstrapservice.WithRuntimeReadyFunc(s.bootstrapValidateRuntimeReady),
	)
	if err != nil {
		return bootstrapModeStartupResult{}, xfmt.Errorf("Failed to create bootstrap service: %w", err)
	}

	batch := s.registration.beginBatch()
	if err := batch.registerBinding(s, svc); err != nil {
		return bootstrapModeStartupResult{}, xfmt.Errorf("Failed to register bootstrap grpc service: %w", err)
	}
	return bootstrapModeStartupResult{ServiceName: svc.Name(), Registration: batch.commit()}, nil
}

func (s *GRPCWebServer) bootstrapValidateRuntimeReady(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return xfmt.Errorf("bootstrap runtime readiness canceled: %w", err)
		}
	}

	if s == nil {
		return xfmt.Errorf("runtime server is not available")
	}

	opts := s.resolvedRuntimeOptions()
	return runplan.ValidateBootstrapRuntimeReady(ctx, opts.distPath, opts.compileBundleMode)
}
