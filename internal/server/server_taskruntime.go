// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"google.golang.org/grpc"
)

type taskRuntimeStartupSummary struct {
	Requested               bool
	Started                 bool
	DispatcherStarted       bool
	SchedulerStarted        bool
	GarbageCollectorStarted bool
}

func (s *GRPCWebServer) startTaskRuntime() taskRuntimeStartupSummary {
	summary := taskRuntimeStartupSummary{Requested: s.servesTaskRuntime()}
	dialer := s.taskRuntimeDialer()
	if !s.taskRuntime.shouldStart(dialer, summary.Requested) {
		summary.Started = s.taskRuntime.started()
		return summary
	}
	result := s.taskRuntime.start(s.runtimeScope, dialer)
	summary.Started = s.taskRuntime.started()
	summary.DispatcherStarted = result.DispatcherStarted
	summary.SchedulerStarted = result.SchedulerStarted
	summary.GarbageCollectorStarted = result.GarbageCollectorStarted
	if result.DispatcherStarted {
		s.runtimeScope.Logger().Debug("task dispatcher started")
	}
	if result.SchedulerStarted {
		s.runtimeScope.Logger().Debug("task scheduler started")
	}
	if result.GarbageCollectorStarted {
		s.runtimeScope.Logger().Debug("task garbage collector started")
	}
	return summary
}

func (s *GRPCWebServer) servesTaskRuntime() bool {
	return s.runState.serves("task")
}

func (s *GRPCWebServer) taskRuntimeDialer() grpcclient.ServiceDialer {
	if s == nil || s.grpcClientPool == nil {
		return nil
	}
	return s.grpcClientPool.Dial
}

func (s *GRPCWebServer) stopTaskRuntime() {
	s.taskRuntime.stop()
}

func (s *GRPCWebServer) taskRuntimeWakeInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil || info == nil {
			return resp, err
		}
		switch info.FullMethod {
		case "/task.Job/EnqueueJob":
			s.publishTaskDispatchWakeup("enqueue")
		case "/task.Schedule/TriggerSchedule":
			s.publishTaskDispatchWakeup("trigger_schedule")
		}
		return resp, err
	}
}

func (s *GRPCWebServer) publishTaskDispatchWakeup(source string) {
	if s == nil {
		return
	}
	events := s.taskRuntime.events
	if events == nil {
		return
	}
	_ = events.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicDispatchWakeup,
		Source: source,
		At:     time.Now().UTC(),
	})
}

func (s *GRPCWebServer) taskRuntimeUnaryInterceptors() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{s.taskRuntimeWakeInterceptor()}
}

func (r taskRuntimeStartupSummary) logFields() []any {
	if !r.Requested && !r.Started && !r.DispatcherStarted && !r.SchedulerStarted && !r.GarbageCollectorStarted {
		return nil
	}
	return []any{
		"startup_task_runtime_requested", r.Requested,
		"startup_task_runtime_started", r.Started,
		"startup_task_runtime_dispatcher_started", r.DispatcherStarted,
		"startup_task_runtime_scheduler_started", r.SchedulerStarted,
		"startup_task_runtime_gc_started", r.GarbageCollectorStarted,
	}
}

func WithTaskHostRuntimeProvider(provider taskcontract.HostRuntimeProvider) ConstructorOption {
	return constructorOptionFunc(func(s *GRPCWebServer) { s.taskRuntime.hostRuntimeProvider = provider })
}

func WithTaskRuntimeFactory(factory func(scope.Scope) taskcontract.Runtime) ConstructorOption {
	return WithTaskHostRuntimeProvider(taskcontract.HostRuntimeProvider(factory))
}
