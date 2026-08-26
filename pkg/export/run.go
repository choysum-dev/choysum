// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

import (
	"context"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// RunFunc executes a validated export spec.
type RunFunc func(context.Context, scope.Scope, Spec) (importpkg.Report, error)

var runFn RunFunc = unregisteredRun

func unregisteredRun(context.Context, scope.Scope, Spec) (importpkg.Report, error) {
	return importpkg.Report{}, Errorf(CodeRunnerNotRegistered, "export runner is not linked; import the runner package")
}

// SetRun wires the default runner implementation. Called from internal/export/runner init.
func SetRun(fn RunFunc) {
	if fn == nil {
		runFn = unregisteredRun
		return
	}
	runFn = fn
}

// Run executes an export synchronously.
func Run(ctx context.Context, runtimeScope scope.Scope, spec Spec) (importpkg.Report, error) {
	if err := ValidateSpec(spec); err != nil {
		return importpkg.Report{}, err
	}
	if spec.Async {
		return importpkg.Report{}, ErrAsyncNotSupported
	}
	return runFn(ctx, runtimeScope, spec)
}
