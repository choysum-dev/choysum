// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/import/hub"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
)

func (s *GRPCWebServer) registerImportHubService() {
	if s == nil || s.server == nil {
		return
	}
	importHub := hub.New(hub.Deps{
		RuntimeScope: s.runtimeScope,
		JSExecutor:   s.JSExecutor(),
	})
	if err := s.registerGRPCServiceDesc(&importpb.ImportHub_ServiceDesc, importHub); err != nil {
		if s.runtimeScope != nil {
			s.runtimeScope.Logger().Warn("import hub service endpoint registration failed", "error", err)
		}
	}
}
