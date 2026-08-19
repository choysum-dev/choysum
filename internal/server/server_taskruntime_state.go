// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/task"
	"github.com/choysum-dev/choysum/pkg/bus"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

type taskRuntimeState struct {
	hostRuntimeProvider taskcontract.HostRuntimeProvider
	events              bus.EventBus
	dispatcher          *task.Dispatcher
	scheduler           *task.Scheduler
	garbageCollector    taskcontract.GarbageCollector
}

type taskRuntimeStartResult struct {
	DispatcherStarted       bool
	SchedulerStarted        bool
	GarbageCollectorStarted bool
}

func (s *taskRuntimeState) started() bool {
	return s.dispatcher != nil
}

func (s *taskRuntimeState) shouldStart(dialer grpcclient.ServiceDialer, servesTaskRuntime bool) bool {
	return !s.started() && dialer != nil && servesTaskRuntime
}

func (s *taskRuntimeState) ensureEvents(runtimeScope scope.Scope) bus.EventBus {
	if bus.IsUsable(s.events) {
		return s.events
	}
	runtime := taskcontract.ResolveHostRuntime(s.hostRuntimeProvider, runtimeScope)
	s.applyEvents(runtimeScope, &runtime)
	return s.events
}

func (s *taskRuntimeState) applyEvents(runtimeScope scope.Scope, runtime *taskcontract.Runtime) {
	if bus.IsUsable(s.events) {
		runtime.Events = s.events
		bus.SetHost(s.events)
		return
	}
	if bus.IsUsable(runtime.Events) {
		s.events = runtime.Events
		bus.SetHost(s.events)
		runtime.Events = s.events
		return
	}
	s.events = bus.EnsureHost(runtimeScope)
	runtime.Events = s.events
}

func (s *taskRuntimeState) start(runtimeScope scope.Scope, dialer grpcclient.ServiceDialer) taskRuntimeStartResult {
	if s.started() {
		return taskRuntimeStartResult{}
	}

	runtime := taskcontract.ResolveHostRuntime(s.hostRuntimeProvider, runtimeScope)
	s.applyEvents(runtimeScope, &runtime)
	result := taskRuntimeStartResult{}

	dispatcher := task.NewDispatcherWithRuntime(runtimeScope, dialer, runtime)
	s.dispatcher = dispatcher
	dispatcher.Start()
	result.DispatcherStarted = true

	if s.scheduler == nil {
		s.scheduler = task.NewSchedulerWithRuntime(runtimeScope, runtime)
		s.scheduler.Start()
		result.SchedulerStarted = true
	}

	if s.garbageCollector == nil {
		collector := runtime.Collector
		if collector == nil {
			collector = task.NewGarbageCollector(runtimeScope)
		}
		s.garbageCollector = collector
		s.garbageCollector.Start()
		result.GarbageCollectorStarted = true
	}

	return result
}

func (s *taskRuntimeState) stop() {
	s.stopDispatcher()
	s.stopScheduler()
	s.stopGarbageCollector()
}

func (s *taskRuntimeState) stopDispatcher() {
	if s.dispatcher == nil {
		return
	}
	s.dispatcher.Stop()
	s.dispatcher = nil
}

func (s *taskRuntimeState) stopScheduler() {
	if s.scheduler == nil {
		return
	}
	s.scheduler.Stop()
	s.scheduler = nil
}

func (s *taskRuntimeState) stopGarbageCollector() {
	if s.garbageCollector == nil {
		return
	}
	s.garbageCollector.Stop()
	s.garbageCollector = nil
}
