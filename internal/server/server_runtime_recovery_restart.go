// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) Restart() error {
	result := s.executeRestartPlan(recoveryExecutionPlan{action: recoveryActionRestart, reload: true})
	return s.finishRecoveryExecution("restart", result)
}

func (s *GRPCWebServer) restart() error {
	result := s.executeRestartPlan(recoveryExecutionPlan{
		action:       recoveryActionRestart,
		reload:       true,
		wrapStopErr:  true,
		wrapStartErr: true,
	})
	return s.finishRecoveryExecution("restart", result)
}

func (s *GRPCWebServer) executeBootstrapModeSwitchPlan(plan bootstrapModeSwitchPlan) (bootstrapModeSwitchResult, error) {
	previous := s.runState.snapshot()
	s.runState.applyBootstrapDecision(plan.Manifest, plan.Decision)

	result := s.executeRestartPlan(recoveryExecutionPlan{
		action:       recoveryActionModeSwitchRestart,
		reload:       false,
		wrapStopErr:  true,
		wrapStartErr: true,
		executor:     s.runtimeRecovery.modeSwitchRestartExecutor,
	})
	if result.err != nil {
		s.runState.restore(previous)
		result.rollbackCount = s.runtimeRecovery.recordModeSwitchRollback()
		return bootstrapModeSwitchResult{}, xfmt.Errorf("bootstrap switch restart failed: %w", s.finishRecoveryExecution("bootstrap_mode_switch", result))
	}
	return bootstrapModeSwitchResult{
		Switched: true,
		Mode:     s.runState.mode(),
		Reason:   s.runState.reason(),
		Targets:  s.runState.serviceTargets(),
	}, nil
}

func (s *GRPCWebServer) executeRestartPlan(plan recoveryExecutionPlan) recoveryExecutionResult {
	result := recoveryExecutionResult{
		action: plan.action,
		reload: plan.reload,
	}
	s.runtimeRecovery.recordAttempt(plan.action)

	s.runtimeRecovery.restartMu.Lock()
	defer s.runtimeRecovery.restartMu.Unlock()
	defer func() {
		result.diagnostics = s.runtimeRecovery.snapshot()
	}()

	if plan.executor != nil {
		result.usedCustomExecutor = true
		result.err = plan.executor()
		if result.err != nil {
			s.runtimeRecovery.recordFailure(plan.action)
		}
		return result
	}

	result.stopErr = s.stop(plan.reload)
	if result.stopErr != nil {
		result.err = result.stopErr
		if plan.wrapStopErr {
			result.err = xfmt.Errorf("Failed to stop server: %w", result.stopErr)
		}
		s.runtimeRecovery.recordFailure(plan.action)
		return result
	}

	result.startErr = s.start(plan.reload)
	if result.startErr != nil {
		result.err = result.startErr
		if plan.wrapStartErr {
			result.err = xfmt.Errorf("Failed to start server: %w", result.startErr)
		}
		s.runtimeRecovery.recordFailure(plan.action)
	}
	return result
}
