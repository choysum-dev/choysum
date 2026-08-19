// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

// runtimeWithDefaultTaskRuntimeDeps fills the long-lived task runtime bundle
// with host-owned defaults while preserving any explicitly injected runtime
// components.
func runtimeWithDefaultTaskRuntimeDeps(runtimeScope scope.Scope, runtime taskcontract.Runtime) taskcontract.Runtime {
	if runtime.Queue == nil {
		runtime.Queue = newTaskRuntimeQueue(runtimeScope)
	}
	if runtime.Store == nil {
		runtime.Store = newTaskRuntimeScheduleStore(runtimeScope)
	}
	if bus.IsUsable(runtime.Events) {
		bus.SetHost(runtime.Events)
	} else {
		runtime.Events = bus.EnsureHost(runtimeScope)
	}
	return runtime
}
