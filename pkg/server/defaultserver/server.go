// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultserver

import (
	internalserver "github.com/choysum-dev/choysum/internal/server"
	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/server"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"github.com/choysum-dev/choysum/pkg/trace"
)

type Option struct {
	apply internalserver.ConstructorOption
}

// TaskHostRuntimeOption configures one component in the long-lived task host
// runtime bundle used by the default server.
type TaskHostRuntimeOption func(*taskcontract.Runtime)

func WithRegistry(r registry.Registry) Option {
	return Option{apply: internalserver.WithRegistry(r)}
}

func WithTelemetry(tel trace.Telemetry) Option {
	return Option{apply: internalserver.WithTelemetry(tel)}
}

// WithTaskHostRuntime is the preferred public wiring entry point for static
// long-lived task host runtime bundles on the default server.
func WithTaskHostRuntime(opts ...TaskHostRuntimeOption) Option {
	runtime := taskcontract.Runtime{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&runtime)
	}
	return WithTaskHostRuntimeProvider(taskcontract.StaticHostRuntimeProvider(runtime))
}

func TaskHostRuntimeWithQueue(queue taskcontract.TaskQueue) TaskHostRuntimeOption {
	return func(runtime *taskcontract.Runtime) {
		runtime.Queue = queue
	}
}

func TaskHostRuntimeWithStore(store taskcontract.ScheduleStore) TaskHostRuntimeOption {
	return func(runtime *taskcontract.Runtime) {
		runtime.Store = store
	}
}

func TaskHostRuntimeWithEvents(events taskcontract.EventBus) TaskHostRuntimeOption {
	return func(runtime *taskcontract.Runtime) {
		runtime.Events = events
	}
}

func TaskHostRuntimeWithCollector(collector taskcontract.GarbageCollector) TaskHostRuntimeOption {
	return func(runtime *taskcontract.Runtime) {
		runtime.Collector = collector
	}
}

// WithTaskHostRuntimeProvider installs a lower-level host runtime provider.
// Prefer WithTaskHostRuntime for public call sites unless the runtime bundle
// must vary with the scope at construction time.
func WithTaskHostRuntimeProvider(provider taskcontract.HostRuntimeProvider) Option {
	return Option{apply: internalserver.WithTaskHostRuntimeProvider(provider)}
}

// WithTaskRuntimeFactory adapts legacy task runtime factory call sites to the
// named host runtime provider seam.
func WithTaskRuntimeFactory(factory func(scope.Scope) taskcontract.Runtime) Option {
	return WithTaskHostRuntimeProvider(taskcontract.HostRuntimeProvider(factory))
}

func NewServer(runtimeScope scope.Scope, opts ...Option) server.Server {
	internalOpts := make([]internalserver.ConstructorOption, 0, len(opts))
	for _, opt := range opts {
		if opt.apply == nil {
			continue
		}
		internalOpts = append(internalOpts, opt.apply)
	}
	return internalserver.NewServer(runtimeScope, internalOpts...)
}
