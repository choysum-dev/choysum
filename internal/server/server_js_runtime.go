// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
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
	if reload || s.jsExecutor == nil {
		return nil
	}
	if err := s.jsExecutor.Stop(); err != nil {
		return xfmt.Errorf("Failed to stop js executor: %w", err)
	}
	return nil
}
