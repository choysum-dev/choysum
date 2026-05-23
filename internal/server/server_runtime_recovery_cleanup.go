// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

func (s *GRPCWebServer) cleanupFailedStartup(result startupLifecycleResult) startupLifecycleResult {
	s.ready.Store(false)
	if result.CleanupPlan == (startupRecoveryPlan{}) {
		result.CleanupPlan = s.planStartupRecovery(result.Reload)
	}
	cleanup := s.executeStartupCleanup(result.Reload, result.CleanupPlan)
	result.Cleanup = cleanup.report()
	if cleanup.err != nil {
		s.logRecoveryFailure("startup_cleanup", result.Cleanup, result.logFields()...)
	}
	return result
}

func (s *GRPCWebServer) planStartupRecovery(reload bool) startupRecoveryPlan {
	return startupRecoveryPlan{
		clearJSRuntime: !reload && s.jsExecutor != nil,
		clearProxy:     s.proxy != nil,
		clearTransport: s.httpServer != nil || s.listener != nil || s.server != nil || s.grpcClientPool != nil,
	}
}

func (s *GRPCWebServer) applyStartupRecoveryPlan(plan startupRecoveryPlan, reload bool) startupRecoveryResult {
	result := startupRecoveryResult{
		plan:    plan,
		stopErr: s.stop(reload),
	}
	if plan.clearJSRuntime {
		s.jsExecutor = nil
		result.jsRuntimeCleared = true
	}
	if plan.clearProxy {
		s.proxy = nil
		result.proxyCleared = true
	}
	if plan.clearTransport {
		s.stopHTTPRuntime()
		s.stopGRPCTransport()
		s.listener = nil
		result.transportCleared = true
	}
	return result
}

func (s *GRPCWebServer) executeStartupCleanup(reload bool, plan startupRecoveryPlan) recoveryExecutionResult {
	cleanup := s.applyStartupRecoveryPlan(plan, reload)
	result := recoveryExecutionResult{
		action:         recoveryActionStartupCleanup,
		reload:         reload,
		stopErr:        cleanup.stopErr,
		err:            cleanup.stopErr,
		startupCleanup: cleanup,
	}
	s.runtimeRecovery.recordAttempt(recoveryActionStartupCleanup)
	if result.err != nil {
		s.runtimeRecovery.recordFailure(recoveryActionStartupCleanup)
	}
	result.diagnostics = s.runtimeRecovery.snapshot()
	return result
}
