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

	// Re-read dist after module install/hotreload so newly built apps (e.g.
	// partner) join ServeTargets. Only when Serve() planned the process — unit
	// tests that call start()/Restart() directly keep their hand-built runState.
	if plan.action == recoveryActionRestart && s.runState.shouldReplanOnRestart() {
		if err := s.planServe(s.runState.serveRequest()); err != nil {
			result.startErr = err
			result.err = err
			if plan.wrapStartErr {
				result.err = xfmt.Errorf("Failed to start server: %w", err)
			}
			s.runtimeRecovery.recordFailure(plan.action)
			return result
		}
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
