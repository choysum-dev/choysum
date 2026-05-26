// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"os"

	pkgbackend "github.com/choysum-dev/choysum/internal/testing/backend"
	pkgdiscovery "github.com/choysum-dev/choysum/internal/testing/discovery"
	pkgfrontend "github.com/choysum-dev/choysum/internal/testing/frontend"
	pkgtypecheck "github.com/choysum-dev/choysum/internal/testing/typecheck"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// RunWithDefaults fills missing callbacks in opts with the package-default
// implementations and then runs the orchestrator.
//
// This keeps cmd wrappers thin while still allowing dependency injection
// in tests by calling Run(...) directly with custom callbacks.
func RunWithDefaults(ctx context.Context, opts RunOptions) error {
	if opts.ResolveApps == nil {
		opts.ResolveApps = pkgdiscovery.ResolveTestApps
	}
	if opts.HasBackendTests == nil {
		opts.HasBackendTests = pkgdiscovery.HasAnyBackendTests
	}
	if opts.HasFrontendTests == nil {
		opts.HasFrontendTests = pkgdiscovery.HasAnyFrontendTests
	}
	if opts.Typecheck == nil {
		stdout := opts.Stdout
		if stdout == nil {
			stdout = os.Stdout
		}
		stderr := opts.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		opts.Typecheck = func(ctx context.Context, runtimeScope scope.Scope, repoRoot string, app string) error {
			runtimeOpts := runtimeOptionsFromScope(runtimeScope)
			if runtimeScope == nil || !runtimeOpts.hasConfig {
				return xfmt.Errorf("typecheck: invalid scope")
			}
			typecheckOpts := pkgtypecheck.RunOptions{
				AddonsPath: runtimeOpts.addonsPath,
				NpmPath:    runtimeOpts.npmPath,
				RepoRoot:   repoRoot,
				TmpPath:    runtimeOpts.tmpPath,
				Target:     app,
				Keep:       opts.Keep,
				Stdout:     stdout,
				Stderr:     stderr,
			}
			return pkgtypecheck.TypecheckApp(ctx, typecheckOpts, app)
		}
	}
	if opts.RunBackend == nil {
		opts.RunBackend = pkgbackend.RunOneAppBackendTests
	}
	if opts.RunFrontend == nil {
		opts.RunFrontend = func(
			ctx context.Context,
			repoRoot string,
			app string,
			junitPath string,
			pattern string,
			coverage bool,
			coverageReport bool,
			coverageCheck bool,
			feCoverageAll bool,
			coverageReportDir string,
			coverageLines int,
			coverageFunctions int,
			coverageBranches int,
			coverageStatements int,
		) (bool, error) {
			tmpRoot := ""
			if envOpts := runtimeOptionsFromScope(opts.Env); envOpts.hasConfig {
				tmpRoot = envOpts.tmpPath
			}
			return pkgfrontend.RunOneAppFrontendTests(
				ctx,
				repoRoot,
				app,
				junitPath,
				pattern,
				coverage,
				coverageReport,
				coverageCheck,
				feCoverageAll,
				coverageReportDir,
				coverageLines,
				coverageFunctions,
				coverageBranches,
				coverageStatements,
				tmpRoot,
				opts.Keep,
			)
		}
	}
	return Run(ctx, opts)
}
