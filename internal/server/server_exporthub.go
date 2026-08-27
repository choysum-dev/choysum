// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/export/hub"
	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
)

func (s *GRPCWebServer) registerExportHubService() {
	if s == nil || s.server == nil {
		return
	}
	exportHub := hub.New(hub.Deps{
		RuntimeScope: s.runtimeScope,
		JSExecutor:   s.JSExecutor(),
	})
	if err := s.registerGRPCServiceDesc(&exportpb.ExportHub_ServiceDesc, exportHub); err != nil {
		if s.runtimeScope != nil {
			s.runtimeScope.Logger().Warn("export hub service endpoint registration failed", "error", err)
		}
	}
}
