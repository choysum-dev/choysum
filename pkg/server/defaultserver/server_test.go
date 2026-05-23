// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultserver

import (
	"context"
	"reflect"
	"testing"
	"time"
	"unsafe"

	internalserver "github.com/choysum-dev/choysum/internal/server"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

func TestOptionsWireServerFieldsThroughConstructor(t *testing.T) {
	exec := jsexecutortest.NewUninitializedExecutor()
	created := NewServer(
		nil,
		WithRegistry(fakeRegistry{}),
		Option{apply: internalserver.WithExecutor(exec)},
		WithTelemetry(fakeTelemetry{}),
		WithTaskHostRuntimeProvider(taskcontract.StaticHostRuntimeProvider(taskcontract.Runtime{})),
	)
	t.Cleanup(func() { closeWatcher(t, created) })

	assertFieldNonNil(t, created, "registry")
	assertFieldNonNil(t, created, "jsExecutor")
	assertFieldNonNil(t, created, "telemetry")
	assertTaskRuntimeFieldNonNil(t, created, "hostRuntimeProvider")
}

func TestWithTaskHostRuntimeBuildsStaticProviderFromWiringOptions(t *testing.T) {
	queue := &fakeTaskQueue{}
	store := &fakeScheduleStore{}
	events := &fakeTaskEventBus{}
	collector := &fakeTaskCollector{}
	created := NewServer(nil, WithTaskHostRuntime(
		TaskHostRuntimeWithQueue(queue),
		TaskHostRuntimeWithStore(store),
		TaskHostRuntimeWithEvents(events),
		TaskHostRuntimeWithCollector(collector),
	))
	t.Cleanup(func() { closeWatcher(t, created) })

	provider := readHostRuntimeProvider(t, created)
	runtime := provider(nil)
	if runtime.Queue != queue {
		t.Fatal("expected task host runtime queue from wiring option")
	}
	if runtime.Store != store {
		t.Fatal("expected task host runtime store from wiring option")
	}
	if runtime.Events != events {
		t.Fatal("expected task host runtime event bus from wiring option")
	}
	if runtime.Collector != collector {
		t.Fatal("expected task host runtime collector from wiring option")
	}
}

func TestWithTaskRuntimeFactoryInstallsProvider(t *testing.T) {
	wantQueue := &fakeTaskQueue{}
	created := NewServer(nil, WithTaskRuntimeFactory(func(scope.Scope) taskcontract.Runtime {
		return taskcontract.Runtime{Queue: wantQueue}
	}))
	t.Cleanup(func() { closeWatcher(t, created) })

	provider := readHostRuntimeProvider(t, created)
	runtime := provider(nil)
	if runtime.Queue != wantQueue {
		t.Fatal("expected task runtime factory to install provider output")
	}
}

func TestNewServerReturnsServerInterfaceWithoutConcreteAssertion(t *testing.T) {
	created := NewServer(nil, WithRegistry(fakeRegistry{}))
	if created == nil {
		t.Fatal("expected server instance")
	}

	assertHotreloadFieldNil(t, created, "watcher")
	assertHotreloadFieldNil(t, created, "queue")
	closeWatcher(t, created)
}

type fakeTelemetry struct{}

func (fakeTelemetry) ServerOptions() []grpc.ServerOption { return nil }

func (fakeTelemetry) Shutdown(context.Context) error { return nil }

type fakeRegistry struct{}

func (fakeRegistry) Scheme() string                                                 { return "fake" }
func (fakeRegistry) Register(string, *resolver.Address) (*registry.Endpoint, error) { return nil, nil }
func (fakeRegistry) UnRegister(*registry.Endpoint) error                            { return nil }
func (fakeRegistry) UnRegisterAll() error                                           { return nil }
func (fakeRegistry) ListServices() ([]*registry.Endpoint, error)                    { return nil, nil }
func (fakeRegistry) GetService(string) ([]*registry.Endpoint, error)                { return nil, nil }
func (fakeRegistry) Resolver() resolver.Builder                                     { return fakeResolverBuilder{} }

type fakeResolverBuilder struct{}

func (fakeResolverBuilder) Scheme() string { return "fake" }

func (fakeResolverBuilder) Build(resolver.Target, resolver.ClientConn, resolver.BuildOptions) (resolver.Resolver, error) {
	return fakeResolver{}, nil
}

type fakeResolver struct{}

func (fakeResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (fakeResolver) Close() {}

func assertFieldNonNil(t *testing.T, target any, fieldName string) {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("missing field %q", fieldName)
	}
	if field.IsNil() {
		t.Fatalf("expected field %q to be non-nil", fieldName)
	}
}

func closeWatcher(t *testing.T, target any) {
	t.Helper()
	watcher := readHotreloadField(t, target, "watcher").Interface().(*fsnotify.Watcher)
	if watcher != nil {
		_ = watcher.Close()
	}
}

func assertHotreloadFieldNonNil(t *testing.T, target any, fieldName string) {
	t.Helper()
	field := readHotreloadField(t, target, fieldName)
	if field.IsNil() {
		t.Fatalf("expected hotreload field %q to be non-nil", fieldName)
	}
}

func assertHotreloadFieldNil(t *testing.T, target any, fieldName string) {
	t.Helper()
	field := readHotreloadField(t, target, fieldName)
	if !field.IsNil() {
		t.Fatalf("expected hotreload field %q to be nil", fieldName)
	}
}

func readHotreloadField(t *testing.T, target any, fieldName string) reflect.Value {
	t.Helper()
	hotreload := reflect.ValueOf(target).Elem().FieldByName("hotreload")
	if !hotreload.IsValid() {
		t.Fatal("missing hotreload field")
	}
	hotreload = reflect.NewAt(hotreload.Type(), unsafe.Pointer(hotreload.UnsafeAddr())).Elem()
	field := hotreload.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("missing hotreload field %q", fieldName)
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
}

func readHostRuntimeProvider(t *testing.T, target any) taskcontract.HostRuntimeProvider {
	t.Helper()
	field := readTaskRuntimeField(t, target, "hostRuntimeProvider")
	provider, ok := field.Interface().(taskcontract.HostRuntimeProvider)
	if !ok || provider == nil {
		t.Fatal("expected task host runtime provider to be set")
	}
	return provider
}

func assertTaskRuntimeFieldNonNil(t *testing.T, target any, fieldName string) {
	t.Helper()
	field := readTaskRuntimeField(t, target, fieldName)
	if field.IsNil() {
		t.Fatalf("expected task runtime field %q to be non-nil", fieldName)
	}
}

func readTaskRuntimeField(t *testing.T, target any, fieldName string) reflect.Value {
	t.Helper()
	taskRuntime := reflect.ValueOf(target).Elem().FieldByName("taskRuntime")
	if !taskRuntime.IsValid() {
		t.Fatal("missing taskRuntime field")
	}
	taskRuntime = reflect.NewAt(taskRuntime.Type(), unsafe.Pointer(taskRuntime.UnsafeAddr())).Elem()
	field := taskRuntime.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("missing taskRuntime field %q", fieldName)
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
}

type fakeTaskQueue struct{}

func (*fakeTaskQueue) Enqueue(context.Context, taskcontract.QueueJob) error { return nil }

func (*fakeTaskQueue) ListReady(context.Context, time.Time, int) ([]taskcontract.QueueJob, error) {
	return nil, nil
}

func (*fakeTaskQueue) TryClaim(context.Context, string, time.Time) (bool, error) { return false, nil }

func (*fakeTaskQueue) Get(context.Context, string) (*taskcontract.QueueJob, error) { return nil, nil }

func (*fakeTaskQueue) UpdateAttempt(context.Context, string, int, time.Time) error { return nil }

func (*fakeTaskQueue) MarkSucceeded(context.Context, string, taskcontract.QueueSuccess) error {
	return nil
}

func (*fakeTaskQueue) MarkFailed(context.Context, string, taskcontract.QueueFailure) error {
	return nil
}

func (*fakeTaskQueue) Retry(context.Context, string, taskcontract.QueueRetry) error { return nil }

func (*fakeTaskQueue) MarkCancelled(context.Context, string, taskcontract.QueueCancellation) error {
	return nil
}

type fakeScheduleStore struct{}

func (*fakeScheduleStore) ListDue(context.Context, time.Time, int) ([]taskcontract.ScheduleEntry, error) {
	return nil, nil
}

func (*fakeScheduleStore) TryAdvanceDue(context.Context, string, time.Time, time.Time, time.Time) (bool, error) {
	return false, nil
}

func (*fakeScheduleStore) UpdateNextRun(context.Context, string, time.Time, time.Time) error {
	return nil
}

func (*fakeScheduleStore) MarkTriggered(context.Context, string, time.Time, time.Time) error {
	return nil
}

func (*fakeScheduleStore) Disable(context.Context, string, time.Time) error { return nil }

type fakeTaskEventBus struct{}

func (*fakeTaskEventBus) Publish(context.Context, taskcontract.Event) error { return nil }

func (*fakeTaskEventBus) Subscribe(string, taskcontract.EventHandler) (taskcontract.Subscription, error) {
	return fakeTaskSubscription{}, nil
}

type fakeTaskSubscription struct{}

func (fakeTaskSubscription) Close() error { return nil }

type fakeTaskCollector struct{}

func (*fakeTaskCollector) Start() {}

func (*fakeTaskCollector) Stop() {}
