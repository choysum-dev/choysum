// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

const defaultDriver = "inprocess"

type runtimeOptions struct {
	driver string
}

func newRuntimeOptions(driver string) runtimeOptions {
	if driver == "" {
		driver = defaultDriver
	}
	return runtimeOptions{driver: driver}
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	// V1: no dedicated bus.driver config key yet; CE always uses inprocess.
	// EE drivers will read an explicit config key here in a later PR.
	_ = runtimeScope
	return newRuntimeOptions("")
}
