// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// RunFunc executes a validated import spec.
type RunFunc func(context.Context, scope.Scope, Spec) (Report, error)

var runFn RunFunc = unregisteredRun

func unregisteredRun(context.Context, scope.Scope, Spec) (Report, error) {
	return Report{}, Errorf(CodeRunnerNotRegistered, "import runner is not linked; import the runner package")
}

// SetRun wires the default runner implementation. Called from internal/import/runner init.
func SetRun(fn RunFunc) {
	if fn == nil {
		runFn = unregisteredRun
		return
	}
	runFn = fn
}

// Run executes an import synchronously.
func Run(ctx context.Context, runtimeScope scope.Scope, spec Spec) (Report, error) {
	if err := ValidateSpec(spec); err != nil {
		return Report{}, err
	}
	if spec.Async {
		return Report{}, ErrAsyncNotSupported
	}
	return runFn(ctx, runtimeScope, spec)
}
