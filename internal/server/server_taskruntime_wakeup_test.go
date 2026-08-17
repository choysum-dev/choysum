// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/bus"
	"google.golang.org/grpc"
)

type recordingEventBus struct {
	published []bus.Event
}

func (b *recordingEventBus) Publish(ctx context.Context, event bus.Event) error {
	b.published = append(b.published, event)
	return nil
}

func (b *recordingEventBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nopSubscription{}, nil
}

type nopSubscription struct{}

func (nopSubscription) Close() error { return nil }

func TestPublishTaskDispatchWakeupPublishesWhenEventsConfigured(t *testing.T) {
	var nilSrv *GRPCWebServer
	nilSrv.publishTaskDispatchWakeup("noop")

	events := &recordingEventBus{}
	srv := &GRPCWebServer{}
	srv.taskRuntime.events = events

	interceptor := srv.taskRuntimeWakeInterceptor()
	for _, method := range []string{"/task.Job/EnqueueJob", "/task.Schedule/TriggerSchedule"} {
		resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, func(ctx context.Context, req any) (any, error) {
			return method, nil
		})
		if err != nil || resp != method {
			t.Fatalf("interceptor(%q) = %#v err=%v", method, resp, err)
		}
	}

	if len(events.published) != 2 {
		t.Fatalf("published = %d, want 2", len(events.published))
	}
	if events.published[0].Topic != bus.TopicDispatchWakeup || events.published[0].Source != "enqueue" {
		t.Fatalf("first event = %#v", events.published[0])
	}
	if events.published[1].Source != "trigger_schedule" {
		t.Fatalf("second event source = %q", events.published[1].Source)
	}
}
