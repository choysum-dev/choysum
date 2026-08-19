// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/tip/hub"
	"github.com/choysum-dev/choysum/internal/tip/proto/tippb"
)

func (s *GRPCWebServer) registerTipHubService() {
	if s == nil || s.server == nil {
		return
	}
	events := s.taskRuntime.ensureEvents(s.runtimeScope)
	tipHub := hub.New(events)
	if err := s.registerGRPCServiceDesc(&tippb.TipHub_ServiceDesc, tipHub); err != nil {
		if s.runtimeScope != nil {
			s.runtimeScope.Logger().Warn("tip hub service endpoint registration failed", "error", err)
		}
	}
}
