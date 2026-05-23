// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

func (s *GRPCWebServer) finishRecoveryExecution(recoveryContext string, result recoveryExecutionResult, extra ...any) error {
	if result.err != nil {
		s.logRecoveryFailure(recoveryContext, result.report(), extra...)
	}
	return result.err
}

func (s *GRPCWebServer) logRecoveryFailure(recoveryContext string, report recoveryExecutionReport, extra ...any) {
	if s.runtimeScope == nil {
		return
	}
	fields := []any{"recovery_context", recoveryContext, "error", report.errorValue()}
	fields = append(fields, report.logFields()...)
	fields = append(fields, extra...)
	s.runtimeScope.Logger().Warn("server recovery failed", fields...)
}

func (r recoveryExecutionResult) report() recoveryExecutionReport {
	return recoveryExecutionReport{
		Action:             r.action,
		Reload:             r.reload,
		UsedCustomExecutor: r.usedCustomExecutor,
		Err:                r.err,
		StopErr:            r.stopErr,
		StartErr:           r.startErr,
		RollbackCount:      r.rollbackCount,
		Diagnostics:        r.diagnostics,
		StartupCleanup: startupRecoveryReport{
			PlannedJSRuntimeRecovery: r.startupCleanup.plan.clearJSRuntime,
			PlannedProxyRecovery:     r.startupCleanup.plan.clearProxy,
			PlannedTransportRecovery: r.startupCleanup.plan.clearTransport,
			JSRuntimeCleared:         r.startupCleanup.jsRuntimeCleared,
			ProxyCleared:             r.startupCleanup.proxyCleared,
			TransportCleared:         r.startupCleanup.transportCleared,
		},
	}
}

func (r recoveryExecutionReport) errorValue() error {
	if r.Err != nil {
		return r.Err
	}
	if r.StartErr != nil {
		return r.StartErr
	}
	return r.StopErr
}

func (r recoveryExecutionReport) logFields() []any {
	action := r.Diagnostics.action(r.Action)
	fields := []any{
		"recovery_action", string(r.Action),
		"recovery_reload", r.Reload,
		"recovery_attempt_count", action.Attempts,
		"recovery_failure_count", action.Failures,
		"recovery_used_custom_executor", r.UsedCustomExecutor,
	}
	if r.StopErr != nil {
		fields = append(fields, "recovery_stop_error", r.StopErr)
	}
	if r.StartErr != nil {
		fields = append(fields, "recovery_start_error", r.StartErr)
	}
	if r.RollbackCount > 0 {
		fields = append(fields, "recovery_mode_switch_rollback_count", r.RollbackCount)
	}
	if r.Action == recoveryActionStartupCleanup {
		fields = append(fields,
			"planned_js_runtime_recovery", r.StartupCleanup.PlannedJSRuntimeRecovery,
			"planned_proxy_recovery", r.StartupCleanup.PlannedProxyRecovery,
			"planned_transport_recovery", r.StartupCleanup.PlannedTransportRecovery,
			"js_runtime_cleared", r.StartupCleanup.JSRuntimeCleared,
			"proxy_cleared", r.StartupCleanup.ProxyCleared,
			"transport_cleared", r.StartupCleanup.TransportCleared,
		)
	}
	return fields
}
