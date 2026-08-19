// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"errors"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/bus/inprocess"
	"github.com/choysum-dev/choysum/internal/tip/proto/tippb"
	"github.com/choysum-dev/choysum/pkg/bus"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"google.golang.org/grpc"
)

func TestEnsureEventsReusesHostRuntimeBus(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)

	events := &recordingEventBus{}
	state := taskRuntimeState{
		hostRuntimeProvider: taskcontract.StaticHostRuntimeProvider(taskcontract.Runtime{Events: events}),
	}
	got := state.ensureEvents(nil)
	if got != events {
		t.Fatalf("ensureEvents() = %p, want host runtime bus %p", got, events)
	}
	if again := state.ensureEvents(nil); again != events {
		t.Fatalf("ensureEvents() second call = %p, want same bus", again)
	}
	if bus.Host() != events {
		t.Fatal("ensureEvents() did not bind injected Events as pkg/bus host")
	}
}

func TestEnsureEventsCreatesSingleton(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)

	state := taskRuntimeState{}
	first := state.ensureEvents(nil)
	if first == nil {
		t.Fatal("ensureEvents() returned nil")
	}
	if second := state.ensureEvents(nil); second != first {
		t.Fatalf("ensureEvents() did not reuse the host singleton")
	}
	if bus.Host() != first {
		t.Fatal("ensureEvents() did not bind pkg/bus host singleton")
	}
	runtime := taskcontract.Runtime{}
	state.applyEvents(nil, &runtime)
	if runtime.Events != first {
		t.Fatal("applyEvents() did not reuse the already-hoisted event bus")
	}
}

func TestRegisterTipHubService(t *testing.T) {
	events := &recordingEventBus{}
	srv := &GRPCWebServer{
		server: grpc.NewServer(),
	}
	srv.taskRuntime.events = events
	srv.registerTipHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[tippb.TipHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, tippb.TipHub_ServiceDesc.ServiceName)
	}
	if srv.taskRuntime.events != events {
		t.Fatal("registerTipHubService replaced the host event bus")
	}
}

func TestRegisterTipHubServiceNilReceiver(t *testing.T) {
	(*GRPCWebServer)(nil).registerTipHubService()
	(&GRPCWebServer{}).registerTipHubService()
}

func TestRegisterTipHubServiceRegistryFailure(t *testing.T) {
	events := &recordingEventBus{}
	reg := &trackingRegistry{registerErr: errors.New("register failed")}
	srv := &GRPCWebServer{
		runtimeScope: newRichServerTestScope(t),
		server:       grpc.NewServer(),
		registry:     reg,
	}
	srv.taskRuntime.events = events
	srv.registerTipHubService()

	info := srv.server.GetServiceInfo()
	if _, ok := info[tippb.TipHub_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services = %#v, want %s", info, tippb.TipHub_ServiceDesc.ServiceName)
	}
	if len(reg.registerCalls) != 1 || reg.registerCalls[0] != tippb.TipHub_ServiceDesc.ServiceName {
		t.Fatalf("registry calls = %#v, want [%s]", reg.registerCalls, tippb.TipHub_ServiceDesc.ServiceName)
	}
}

func TestRegisterTipHubServiceRegistryFailureWithoutScope(t *testing.T) {
	reg := &trackingRegistry{registerErr: errors.New("register failed")}
	srv := &GRPCWebServer{
		server:   grpc.NewServer(),
		registry: reg,
	}
	srv.registerTipHubService()
	if len(reg.registerCalls) != 1 || reg.registerCalls[0] != tippb.TipHub_ServiceDesc.ServiceName {
		t.Fatalf("registry calls = %#v, want [%s]", reg.registerCalls, tippb.TipHub_ServiceDesc.ServiceName)
	}
}
