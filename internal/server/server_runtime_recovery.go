// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"sync"
	"sync/atomic"
)

type recoveryAction string

const (
	recoveryActionRestart           recoveryAction = "restart"
	recoveryActionModeSwitchRestart recoveryAction = "bootstrap_mode_switch_restart"
	recoveryActionStartupCleanup    recoveryAction = "failed_startup_cleanup"
)

type recoveryExecutorFunc func() error

type runtimeRecoveryState struct {
	restartMu                 sync.Mutex
	modeSwitchRestartExecutor recoveryExecutorFunc
	diagnostics               recoveryDiagnostics
}

type recoveryDiagnostics struct {
	restartAttempts           atomic.Uint64
	restartFailures           atomic.Uint64
	modeSwitchRestartAttempts atomic.Uint64
	modeSwitchRestartFailures atomic.Uint64
	startupCleanupAttempts    atomic.Uint64
	startupCleanupFailures    atomic.Uint64
	modeSwitchRollbacks       atomic.Uint64
}

type startupRecoveryPlan struct {
	clearJSRuntime bool
	clearProxy     bool
	clearTransport bool
}

type startupRecoveryResult struct {
	plan             startupRecoveryPlan
	stopErr          error
	jsRuntimeCleared bool
	proxyCleared     bool
	transportCleared bool
}

type recoveryExecutionPlan struct {
	action       recoveryAction
	reload       bool
	wrapStopErr  bool
	wrapStartErr bool
	executor     recoveryExecutorFunc
}

type recoveryExecutionResult struct {
	action             recoveryAction
	reload             bool
	rollbackCount      uint64
	usedCustomExecutor bool
	stopErr            error
	startErr           error
	err                error
	startupCleanup     startupRecoveryResult
	diagnostics        recoveryDiagnosticsSnapshot
}

type recoveryActionDiagnostics struct {
	Attempts uint64
	Failures uint64
}

type recoveryDiagnosticsSnapshot struct {
	Restart             recoveryActionDiagnostics
	ModeSwitchRestart   recoveryActionDiagnostics
	StartupCleanup      recoveryActionDiagnostics
	ModeSwitchRollbacks uint64
}

type startupRecoveryReport struct {
	PlannedJSRuntimeRecovery bool
	PlannedProxyRecovery     bool
	PlannedTransportRecovery bool
	JSRuntimeCleared         bool
	ProxyCleared             bool
	TransportCleared         bool
}

type recoveryExecutionReport struct {
	Action             recoveryAction
	Reload             bool
	UsedCustomExecutor bool
	Err                error
	StopErr            error
	StartErr           error
	RollbackCount      uint64
	Diagnostics        recoveryDiagnosticsSnapshot
	StartupCleanup     startupRecoveryReport
}

func (r *runtimeRecoveryState) snapshot() recoveryDiagnosticsSnapshot {
	return recoveryDiagnosticsSnapshot{
		Restart: recoveryActionDiagnostics{
			Attempts: r.diagnostics.restartAttempts.Load(),
			Failures: r.diagnostics.restartFailures.Load(),
		},
		ModeSwitchRestart: recoveryActionDiagnostics{
			Attempts: r.diagnostics.modeSwitchRestartAttempts.Load(),
			Failures: r.diagnostics.modeSwitchRestartFailures.Load(),
		},
		StartupCleanup: recoveryActionDiagnostics{
			Attempts: r.diagnostics.startupCleanupAttempts.Load(),
			Failures: r.diagnostics.startupCleanupFailures.Load(),
		},
		ModeSwitchRollbacks: r.diagnostics.modeSwitchRollbacks.Load(),
	}
}

func (r recoveryDiagnosticsSnapshot) action(action recoveryAction) recoveryActionDiagnostics {
	switch action {
	case recoveryActionRestart:
		return r.Restart
	case recoveryActionModeSwitchRestart:
		return r.ModeSwitchRestart
	case recoveryActionStartupCleanup:
		return r.StartupCleanup
	default:
		return recoveryActionDiagnostics{}
	}
}

func (r *runtimeRecoveryState) recordAttempt(action recoveryAction) uint64 {
	switch action {
	case recoveryActionRestart:
		return r.diagnostics.restartAttempts.Add(1)
	case recoveryActionModeSwitchRestart:
		return r.diagnostics.modeSwitchRestartAttempts.Add(1)
	case recoveryActionStartupCleanup:
		return r.diagnostics.startupCleanupAttempts.Add(1)
	default:
		return 0
	}
}

func (r *runtimeRecoveryState) recordFailure(action recoveryAction) uint64 {
	switch action {
	case recoveryActionRestart:
		return r.diagnostics.restartFailures.Add(1)
	case recoveryActionModeSwitchRestart:
		return r.diagnostics.modeSwitchRestartFailures.Add(1)
	case recoveryActionStartupCleanup:
		return r.diagnostics.startupCleanupFailures.Add(1)
	default:
		return 0
	}
}

func (r *runtimeRecoveryState) recordModeSwitchRollback() uint64 {
	return r.diagnostics.modeSwitchRollbacks.Add(1)
}
