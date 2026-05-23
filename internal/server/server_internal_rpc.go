// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import "github.com/choysum-dev/choysum/internal/jobtoken"

func (s *GRPCWebServer) registerInternalRPCServices(opts runtimeOptions) {
	s.registerInternalJobTokenService(opts)
}

func (s *GRPCWebServer) registerInternalJobTokenService(opts runtimeOptions) {
	if !opts.authEnabled || s.authenticator == nil {
		return
	}

	jobTokenSvc := jobtoken.NewService(s.runtimeScope, s.authenticator)
	if desc, err := jobTokenSvc.ServiceDesc(); err == nil {
		if err := s.registerGRPCServiceDesc(desc, jobTokenSvc); err != nil {
			s.runtimeScope.Logger().Warn("job token service endpoint registration failed", "error", err)
		}
	} else {
		s.runtimeScope.Logger().Warn("job token service descriptor build failed", "error", err)
	}
}
