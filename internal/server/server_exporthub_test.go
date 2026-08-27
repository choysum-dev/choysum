// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"google.golang.org/grpc"
)

func TestRegisterExportHubService(t *testing.T) {
	srv := &GRPCWebServer{
		server: grpc.NewServer(),
	}
	srv.registerExportHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[exportpb.ExportHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, exportpb.ExportHub_ServiceDesc.ServiceName)
	}
}

func TestRegisterExportHubServiceNilReceiver(t *testing.T) {
	(*GRPCWebServer)(nil).registerExportHubService()
	(&GRPCWebServer{}).registerExportHubService()
}

func TestRegisterExportHubServiceRegistryFailure(t *testing.T) {
	reg := &trackingRegistry{registerErr: errors.New("register failed")}
	srv := &GRPCWebServer{
		runtimeScope: newRichServerTestScope(t),
		server:       grpc.NewServer(),
		registry:     reg,
	}
	srv.registerExportHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[exportpb.ExportHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, exportpb.ExportHub_ServiceDesc.ServiceName)
	}
	if len(reg.registerCalls) != 1 || reg.registerCalls[0] != exportpb.ExportHub_ServiceDesc.ServiceName {
		t.Fatalf("registry calls = %#v, want [%s]", reg.registerCalls, exportpb.ExportHub_ServiceDesc.ServiceName)
	}
}
