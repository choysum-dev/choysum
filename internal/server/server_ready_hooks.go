// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

// runApplicationReadyHooks runs @HookReady for installed modules whose application
// is among the current process service targets.
//
// Must run after application services are registered, the JS runtime is active,
// and transport ingress is listening so cross-app Schedule RPC can Dial.
func (s *GRPCWebServer) runApplicationReadyHooks(ctx context.Context) error {
	if s == nil || s.runtimeScope == nil || s.jsExecutor == nil {
		return nil
	}
	session := s.runtimeScope.Session()
	if session == nil || session.DB == nil {
		return nil
	}
	if ctx == nil {
		ctx = s.runtimeScope.Context()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Hook JS uses createServiceByModel → gRPC unary when local pool misses;
	// request-path interceptors are not in play, so inject the dialer explicitly.
	if s.grpcClientPool != nil {
		ctx = grpcclient.ContextWithServiceDialer(ctx, s.grpcClientPool.Dial)
	}
	var installed []meta.Module
	if err := session.Where("status = ?", meta.Installed).Order("id ASC").Find(&installed).Error; err != nil {
		// Fresh / unmigrated DB: nothing to run yet. Do not use HasTable alone —
		// false negatives can skip ready hooks on a live schema.
		if isMissingSQLTableError(err) {
			return nil
		}
		return xfmt.Errorf("load installed modules for ready hooks: %w", err)
	}
	targets := s.runState.serviceTargets()
	if err := hooks.RunReadyForInstalledModules(ctx, s.runtimeScope, s.jsExecutor, hooks.RunReadyOptions{
		ApplicationNames: targets,
		Modules:          installed,
		// Application init already registered @Hook* under per-module CHOYSUM_* seeds
		// and defines resident __choysum_hook__; only invoke, do not Reload.
		InvokeOnly: true,
	}); err != nil {
		return xfmt.Errorf("application ready hooks: %w", err)
	}
	return nil
}

// isMissingSQLTableError reports dialect-specific "table does not exist" failures.
// Narrow on purpose: do not match generic "does not exist" (columns/constraints).
func isMissingSQLTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no such table") || strings.Contains(msg, "unknown table") {
		return true
	}
	if strings.Contains(msg, "table") && strings.Contains(msg, "doesn't exist") {
		return true
	}
	if strings.Contains(msg, `relation "`) && strings.Contains(msg, "does not exist") &&
		!strings.Contains(msg, "constraint") {
		return true
	}
	return false
}
