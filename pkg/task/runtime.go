// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Runtime bundles the task runtime contracts used by the default task runtime
// wiring.
//
// Callers may provide a full replacement bundle or override only selected
// components. Nil fields mean the host runtime should keep using its default
// implementation for that dependency.
type Runtime struct {
	Queue     TaskQueue
	Store     ScheduleStore
	Events    bus.EventBus
	Collector GarbageCollector
}

// HostRuntimeProvider resolves the long-lived, host-owned task runtime bundle
// used by task hosts such as the default server wiring.
type HostRuntimeProvider func(scope.Scope) Runtime

// StaticHostRuntimeProvider adapts a concrete runtime bundle into a host
// runtime provider.
func StaticHostRuntimeProvider(runtime Runtime) HostRuntimeProvider {
	return func(scope.Scope) Runtime { return runtime }
}

// ResolveHostRuntime evaluates the provider for the supplied scope and returns an empty
// runtime bundle when no provider is configured.
func ResolveHostRuntime(provider HostRuntimeProvider, runtimeScope scope.Scope) Runtime {
	if provider == nil {
		return Runtime{}
	}
	return provider(runtimeScope)
}
