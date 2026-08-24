// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"google.golang.org/grpc"
)

func TestRegisterImportHubService(t *testing.T) {
	srv := &GRPCWebServer{
		server: grpc.NewServer(),
	}
	srv.registerImportHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[importpb.ImportHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, importpb.ImportHub_ServiceDesc.ServiceName)
	}
}

func TestRegisterImportHubServiceNilReceiver(t *testing.T) {
	(*GRPCWebServer)(nil).registerImportHubService()
	(&GRPCWebServer{}).registerImportHubService()
}

func TestRegisterImportHubServiceRegistryFailure(t *testing.T) {
	reg := &trackingRegistry{registerErr: errors.New("register failed")}
	srv := &GRPCWebServer{
		runtimeScope: newRichServerTestScope(t),
		server:       grpc.NewServer(),
		registry:     reg,
	}
	srv.registerImportHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[importpb.ImportHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, importpb.ImportHub_ServiceDesc.ServiceName)
	}
	if len(reg.registerCalls) != 1 || reg.registerCalls[0] != importpb.ImportHub_ServiceDesc.ServiceName {
		t.Fatalf("registry calls = %#v, want [%s]", reg.registerCalls, importpb.ImportHub_ServiceDesc.ServiceName)
	}
}
