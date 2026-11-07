// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"log/slog"

	"github.com/choysum-dev/choysum/internal/defaultengine/quickjsruntime"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) JSExecutor() jsexecutor.JsExecutor {
	return s.jsExecutor
}

func (s *GRPCWebServer) ensureJsExecutor(reload bool) error {
	if reload && s.jsExecutor != nil {
		return nil
	}
	if !reload || s.jsExecutor == nil {
		executor, err := jsexecutor.NewRuntimeExecutor(s.runtimeScope, s.authenticator)
		if err != nil {
			return xfmt.Errorf("Failed to create runtime executor: %w", err)
		}
		s.jsExecutor = executor
	}
	return nil
}

func (s *GRPCWebServer) stopJSExecutor(reload bool) error {
	// Process-scoped moduleManagement compiler is independent of the runtime
	// executor; always stop it so reload/shutdown does not leave a warm pool.
	if err := quickjsruntime.StopProcessModuleManagementHost(); err != nil {
		log := slog.Default()
		if s.runtimeScope != nil {
			if l := s.runtimeScope.Logger(); l != nil {
				log = l
			}
		}
		log.Warn("stop process module management host", "error", err)
	}
	if reload || s.jsExecutor == nil {
		return nil
	}
	if err := s.jsExecutor.Stop(); err != nil {
		return xfmt.Errorf("Failed to stop js executor: %w", err)
	}
	return nil
}
