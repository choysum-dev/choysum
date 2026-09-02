// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

type captureBus struct {
	mu     sync.Mutex
	events []bus.Event
}

func (b *captureBus) Publish(_ context.Context, event bus.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *captureBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nopSub{}, nil
}

func (b *captureBus) snapshot() []bus.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]bus.Event, len(b.events))
	copy(out, b.events)
	return out
}

type nopSub struct{}

func (nopSub) Close() error { return nil }

type failPublishBus struct{}

func (failPublishBus) Publish(context.Context, bus.Event) error {
	return context.Canceled
}

func (failPublishBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nopSub{}, nil
}

func TestPublishModuleOpChangedEmitsForMetaJobs(t *testing.T) {
	events := &captureBus{}
	d := &Dispatcher{events: events}
	job := &Job{
		Id:                "job-1",
		FullMethod:        "meta.MetaModule/ExecuteInstall",
		TriggeredByUserId: "user-1",
	}
	d.publishModuleOpChanged(job, "task.Dispatcher.succeedJob")

	got := events.snapshot()
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
	if got[0].Topic != bus.TopicMetaModuleOpChanged {
		t.Fatalf("topic = %q", got[0].Topic)
	}
	if got[0].Source != "task.Dispatcher.succeedJob" {
		t.Fatalf("source = %q", got[0].Source)
	}
	if got[0].Payload["jobId"] != "job-1" || got[0].Payload["resId"] != "job-1" {
		t.Fatalf("payload locators = %#v", got[0].Payload)
	}
	if got[0].Payload["userId"] != "user-1" || got[0].Payload["model"] != metaModuleOpTipModel {
		t.Fatalf("payload identity = %#v", got[0].Payload)
	}
	if got[0].At.IsZero() || time.Since(got[0].At) > time.Minute {
		t.Fatalf("unexpected At: %v", got[0].At)
	}
}

func TestPublishModuleOpChangedSkipsNonMetaAndMissingBus(t *testing.T) {
	events := &captureBus{}
	d := &Dispatcher{events: events}
	d.publishModuleOpChanged(&Job{Id: "j", FullMethod: "auth.User/Login", TriggeredByUserId: "u"}, "x")
	d.publishModuleOpChanged(nil, "x")
	d.publishModuleOpChanged(&Job{Id: "", FullMethod: "meta.MetaModule/ExecuteUpgrade", TriggeredByUserId: "u"}, "x")
	(&Dispatcher{}).publishModuleOpChanged(&Job{Id: "j", FullMethod: "meta.MetaModule/ExecuteInstall"}, "x")
	if len(events.snapshot()) != 0 {
		t.Fatalf("expected no publishes, got %#v", events.snapshot())
	}
}

func TestPublishModuleOpChangedUsesSchedulerWhenTriggeredMissing(t *testing.T) {
	events := &captureBus{}
	d := &Dispatcher{events: events}
	d.publishModuleOpChanged(&Job{
		Id:              "job-2",
		FullMethod:      "meta.MetaModule/ExecuteUninstall",
		SchedulerUserId: "sched-1",
	}, "")
	got := events.snapshot()
	if len(got) != 1 {
		t.Fatalf("events = %d", len(got))
	}
	if got[0].Source != "task.Dispatcher" {
		t.Fatalf("default source = %q", got[0].Source)
	}
	if got[0].Payload["userId"] != "sched-1" {
		t.Fatalf("userId = %#v", got[0].Payload["userId"])
	}
}

func TestPublishModuleOpChangedIgnoresPublishErrors(t *testing.T) {
	d := &Dispatcher{events: failPublishBus{}}
	d.publishModuleOpChanged(&Job{
		Id:                "job-3",
		FullMethod:        "meta.MetaModule/ExecuteInstall",
		TriggeredByUserId: "u1",
	}, "task.Dispatcher.failJob")
}

func TestIsMetaModuleOpJob(t *testing.T) {
	if isMetaModuleOpJob(nil) {
		t.Fatal("nil job")
	}
	if !isMetaModuleOpJob(&Job{FullMethod: "meta.MetaModule/ExecuteInstall"}) {
		t.Fatal("install")
	}
	if isMetaModuleOpJob(&Job{FullMethod: "meta.MetaModule/PlanOperation"}) {
		t.Fatal("plan should not tip")
	}
}
