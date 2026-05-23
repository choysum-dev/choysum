// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import "google.golang.org/grpc"

func (s *GRPCWebServer) registerGRPCServiceDesc(desc *grpc.ServiceDesc, impl any) error {
	s.server.RegisterService(desc, impl)
	return s.registerGRPCServiceEndpoint(desc.ServiceName)
}

func (s *GRPCWebServer) registerGRPCServiceEndpoint(serviceName string) error {
	if s.registry == nil {
		return nil
	}
	_, err := s.registry.Register(serviceName, s.address)
	return err
}
