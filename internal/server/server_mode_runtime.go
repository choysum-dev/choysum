// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"sort"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	xfmt "golang.org/x/exp/errors/fmt"
)

type modeRuntimeStartupSummary struct {
	Mode                      runplan.RunMode
	RegisteredServiceCount    int
	RegisteredGRPCMethodCount int
	InitScriptCount           int
	JSRuntimePrepared         bool
	JSRuntimeActivated        bool
	JSRuntimeReload           bool
}

type applicationModeStartupResult struct {
	Reload              bool
	Registration        registrationBatchResult
	InitScripts         []*jsengine.JsScript
	JSRuntimePrepared   bool
	JSRuntimeActivated  bool
	JSRuntimeDurationMs int64
	JSRuntimeInfo       *jsexecutor.RuntimeInfo
}

type bootstrapModeStartupResult struct {
	ServiceName  string
	Registration registrationBatchResult
}

func (s *GRPCWebServer) startModeRuntime(reload bool) (modeRuntimeStartupSummary, error) {
	if s.runState.isBootstrapMode() {
		return s.startBootstrapModeRuntime()
	}
	return s.startApplicationModeRuntime(reload)
}

func (s *GRPCWebServer) startBootstrapModeRuntime() (modeRuntimeStartupSummary, error) {
	result, err := s.prepareBootstrapModeStartup()
	if err != nil {
		return result.summary(), err
	}
	return s.finishBootstrapModeStartup(result), nil
}

func (s *GRPCWebServer) startApplicationModeRuntime(reload bool) (modeRuntimeStartupSummary, error) {
	result, err := s.prepareApplicationModeStartup(reload)
	if err != nil {
		return result.summary(), err
	}
	return s.finishApplicationModeStartup(result)
}

func (s *GRPCWebServer) prepareApplicationModeStartup(reload bool) (applicationModeStartupResult, error) {
	result := applicationModeStartupResult{Reload: reload}
	if err := s.ensureJsExecutor(reload); err != nil {
		return result, err
	}
	result.JSRuntimePrepared = s.jsExecutor != nil
	if runtimeInfoReader, ok := s.jsExecutor.(jsexecutor.RuntimeInfoReader); ok {
		runtimeInfo := runtimeInfoReader.RuntimeInfo()
		result.JSRuntimeInfo = &runtimeInfo
	}

	registration, err := s.registerApplicationServices()
	if err != nil {
		return result, err
	}
	result.Registration = registration
	result.InitScripts, err = jsengine.DedupeInitScripts(registration.InitScripts)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (s *GRPCWebServer) prepareBootstrapModeStartup() (bootstrapModeStartupResult, error) {
	return s.registerBootstrapService()
}

func (s *GRPCWebServer) finishApplicationModeStartup(result applicationModeStartupResult) (modeRuntimeStartupSummary, error) {
	if !result.Reload {
		s.runtimeScope.Logger().Debug("js executor starting", "init_script_count", len(result.InitScripts))
		s.jsExecutor.SetJsScripts(result.InitScripts)
		startedAt := time.Now()
		if err := s.jsExecutor.Start(); err != nil {
			return result.summary(), xfmt.Errorf("Failed to start js executor: %w", err)
		}
		result.JSRuntimeDurationMs = time.Since(startedAt).Milliseconds()
		s.runtimeScope.Logger().Info("js executor started", result.startupLogFields()...)
	} else {
		s.runtimeScope.Logger().Debug("js executor reloading", "init_script_count", len(result.InitScripts))
		startedAt := time.Now()
		if err := s.jsExecutor.Reload(result.InitScripts...); err != nil {
			return result.summary(), xfmt.Errorf("Failed to reload js executor: %w", err)
		}
		result.JSRuntimeDurationMs = time.Since(startedAt).Milliseconds()
		s.runtimeScope.Logger().Info("js executor reloaded", result.startupLogFields()...)
	}
	result.JSRuntimeActivated = true

	return result.summary(), nil
}

func (s *GRPCWebServer) finishBootstrapModeStartup(result bootstrapModeStartupResult) modeRuntimeStartupSummary {
	if result.ServiceName != "" {
		s.runState.switchToBootstrapService(result.ServiceName)
	}
	s.runtimeScope.Logger().Debug("bootstrap service registered", "reason", s.runState.reason())
	return result.summary()
}

func (r applicationModeStartupResult) summary() modeRuntimeStartupSummary {
	return modeRuntimeStartupSummary{
		Mode:                      runplan.RunModeApplication,
		RegisteredServiceCount:    len(r.Registration.Bindings),
		RegisteredGRPCMethodCount: len(r.Registration.GRPCMethods),
		InitScriptCount:           len(r.InitScripts),
		JSRuntimePrepared:         r.JSRuntimePrepared,
		JSRuntimeActivated:        r.JSRuntimeActivated,
		JSRuntimeReload:           r.Reload && r.JSRuntimeActivated,
	}
}

func (r applicationModeStartupResult) startupLogFields() []any {
	fields := []any{
		"init_script_count", len(r.InitScripts),
		"duration_ms", r.JSRuntimeDurationMs,
	}
	if r.JSRuntimeInfo != nil {
		fields = append(fields,
			"min_pool_size", r.JSRuntimeInfo.MinPoolSize,
			"max_pool_size", r.JSRuntimeInfo.MaxPoolSize,
		)
	}
	serviceNames := r.registeredServiceNames()
	if len(serviceNames) == 0 {
		return fields
	}
	return append(fields,
		"service_count", len(serviceNames),
		"services", serviceNames,
	)
}

func (r applicationModeStartupResult) registeredServiceNames() []string {
	names := make([]string, 0, len(r.Registration.Bindings))
	for _, binding := range r.Registration.Bindings {
		if binding == nil {
			continue
		}
		name := strings.TrimSpace(binding.Name())
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r bootstrapModeStartupResult) summary() modeRuntimeStartupSummary {
	return modeRuntimeStartupSummary{
		Mode:                      runplan.RunModeBootstrap,
		RegisteredServiceCount:    len(r.Registration.Bindings),
		RegisteredGRPCMethodCount: len(r.Registration.GRPCMethods),
		InitScriptCount:           len(r.Registration.InitScripts),
	}
}

func (r modeRuntimeStartupSummary) logFields() []any {
	if r.Mode == "" {
		return nil
	}
	return []any{
		"startup_mode_runtime_mode", string(r.Mode),
		"startup_mode_runtime_registered_service_count", r.RegisteredServiceCount,
		"startup_mode_runtime_registered_grpc_method_count", r.RegisteredGRPCMethodCount,
		"startup_mode_runtime_init_script_count", r.InitScriptCount,
		"startup_mode_runtime_js_prepared", r.JSRuntimePrepared,
		"startup_mode_runtime_js_activated", r.JSRuntimeActivated,
		"startup_mode_runtime_js_reload", r.JSRuntimeReload,
	}
}
