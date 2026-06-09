// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	modulesPath           string
	distPath              string
	defaultChoysumPath    string
	backendEnv            map[string]any
	grpcAuthentication    bool
	grpcMethodAccess      bool
	grpcRecordRule        bool
	grpcCompanyFilter     bool
	grpcFieldRule         bool
	authzDecisionLog      string
	authzDecisionAudit    bool
	taskDefaultMaxAttempt int
	compileSourceMap      bool
	compileMinify         bool
	compileTreeShaking    bool
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool, taskOpts scope.TaskRuntimeOptions, hasTaskOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool, envOpts scope.RuntimeEnvironmentOptions, hasEnvOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()

	opts := runtimeOptions{
		backendEnv:         map[string]any{},
		grpcAuthentication: true,
		grpcMethodAccess:   true,
		grpcRecordRule:     true,
		grpcCompanyFilter:  true,
		grpcFieldRule:      true,
		compileSourceMap:   compileDefaults.SourceMap,
		compileMinify:      compileDefaults.Minify,
		compileTreeShaking: compileDefaults.TreeShaking,
	}

	if hasPathOpts {
		opts.modulesPath = pathOpts.ModulesPath
		opts.distPath = pathOpts.DistPath
		opts.defaultChoysumPath = pathOpts.DefaultChoysumPath
	}

	if hasEnvOpts && envOpts.BackendEnv != nil {
		opts.backendEnv = envOpts.BackendEnv
	}

	if hasAuthOpts {
		opts.grpcAuthentication = authOpts.GrpcAuthentication
		opts.grpcMethodAccess = authOpts.GrpcMethodAccess
		opts.grpcRecordRule = authOpts.GrpcRecordRule
		opts.grpcCompanyFilter = authOpts.GrpcCompanyFilter
		opts.grpcFieldRule = authOpts.GrpcFieldRule
		opts.authzDecisionLog = authOpts.AuthzDecisionLog
		opts.authzDecisionAudit = authOpts.AuthzDecisionAudit
	}

	if hasTaskOpts && taskOpts.Task != nil && taskOpts.Task.Dispatch != nil {
		opts.taskDefaultMaxAttempt = taskOpts.Task.Dispatch.DefaultMaxAttempts
	}

	if hasCompileOpts {
		opts.compileSourceMap = compileOpts.SourceMap
		opts.compileMinify = compileOpts.Minify
		opts.compileTreeShaking = compileOpts.TreeShaking
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	taskOpts, hasTaskOpts := scope.TaskRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	envOpts, hasEnvOpts := scope.RuntimeEnvironmentOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, authOpts, hasAuthOpts, taskOpts, hasTaskOpts, compileOpts, hasCompileOpts, envOpts, hasEnvOpts)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.distPath) != ""
}

func (b *ModuleBuilder) resolvedRuntimeOptions() runtimeOptions {
	if b != nil && hasRuntimeOptions(b.runtimeOptions) {
		return b.runtimeOptions
	}
	if b != nil && b.runtimeScope != nil {
		return runtimeOptionsFromScope(b.runtimeScope)
	}
	if b != nil {
		return b.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("backend builder runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("backend builder runtime options: distPath is required")
	}
	if strings.TrimSpace(o.defaultChoysumPath) == "" {
		return xfmt.Errorf("backend builder runtime options: defaultChoysumPath is required")
	}
	return nil
}
