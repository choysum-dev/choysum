// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

type startupPhase string

const (
	startupPhaseGRPCTransport    startupPhase = "grpc_transport"
	startupPhaseModeRuntime      startupPhase = "mode_runtime"
	startupPhaseTransportIngress startupPhase = "transport_ingress"
)

type startupLifecycleResult struct {
	Reload         bool
	Succeeded      bool
	ReadyActivated bool
	FailedPhase    startupPhase
	AuthRuntime    authRuntimeSetupResult
	ModeRuntime    modeRuntimeStartupSummary
	TaskRuntime    taskRuntimeStartupSummary
	Err            error
	CleanupPlan    startupRecoveryPlan
	Cleanup        recoveryExecutionReport
}

func (s *GRPCWebServer) runStartupLifecycle(reload bool, opts runtimeOptions) startupLifecycleResult {
	result := startupLifecycleResult{Reload: reload}
	s.initStartupState(opts)
	result.AuthRuntime = s.startAuthRuntime(opts)
	if result.AuthRuntime.Degraded {
		s.runtimeScope.Logger().Warn("auth runtime degraded", result.AuthRuntime.logFields()...)
	}
	failedPhase, modeRuntime, taskRuntime, err := s.startRuntimeOwners(reload, opts)
	result.ModeRuntime = modeRuntime
	result.TaskRuntime = taskRuntime
	if err != nil {
		result.Err = err
		result.FailedPhase = failedPhase
		result.CleanupPlan = s.planStartupRecovery(reload)
		return s.cleanupFailedStartup(result)
	}
	s.ready.Store(true)
	result.Succeeded = true
	result.ReadyActivated = true
	return result
}

func (s *GRPCWebServer) initStartupState(opts runtimeOptions) {
	s.ready.Store(false)
	s.runState.ensureStartupDefaults(opts)
	s.initBaseServerState(opts)
}

func (s *GRPCWebServer) startRuntimeOwners(reload bool, opts runtimeOptions) (startupPhase, modeRuntimeStartupSummary, taskRuntimeStartupSummary, error) {
	if err := s.startGRPCTransport(opts); err != nil {
		return startupPhaseGRPCTransport, modeRuntimeStartupSummary{}, taskRuntimeStartupSummary{}, err
	}
	s.registerInternalRPCServices(opts)
	modeRuntime, err := s.startModeRuntime(reload)
	if err != nil {
		return startupPhaseModeRuntime, modeRuntime, taskRuntimeStartupSummary{}, err
	}
	taskRuntime := s.startTaskRuntime()
	if err := s.startTransportIngress(opts, reload); err != nil {
		return startupPhaseTransportIngress, modeRuntime, taskRuntime, err
	}
	return "", modeRuntime, taskRuntime, nil
}

func (r startupLifecycleResult) errorValue() error {
	return r.Err
}

func (r startupLifecycleResult) logFields() []any {
	fields := []any{
		"startup_reload", r.Reload,
		"startup_succeeded", r.Succeeded,
		"startup_ready_activated", r.ReadyActivated,
	}
	if r.FailedPhase != "" {
		fields = append(fields, "startup_failed_phase", string(r.FailedPhase))
	}
	fields = append(fields, r.AuthRuntime.logFields()...)
	fields = append(fields, r.ModeRuntime.logFields()...)
	fields = append(fields, r.TaskRuntime.logFields()...)
	if r.Err != nil {
		fields = append(fields, "startup_error", r.Err)
	}
	return fields
}
