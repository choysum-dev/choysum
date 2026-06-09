// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestBackendRuntimeOptionsNewAndValidate(t *testing.T) {
	t.Parallel()

	compileDefaults := config.NewDefaultCompileConfig()
	base := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false)
	if base.backendEnv == nil {
		t.Fatal("default backendEnv should be initialized")
	}
	if len(base.backendEnv) != 0 {
		t.Fatalf("default backendEnv length = %d, want 0", len(base.backendEnv))
	}
	if !base.grpcAuthentication || !base.grpcMethodAccess || !base.grpcRecordRule || !base.grpcCompanyFilter || !base.grpcFieldRule {
		t.Fatalf("default grpc flags = %#v, want all enabled", base)
	}
	if base.compileSourceMap != compileDefaults.SourceMap || base.compileMinify != compileDefaults.Minify || base.compileTreeShaking != compileDefaults.TreeShaking {
		t.Fatalf("default compile flags = %#v, want defaults from config", base)
	}

	override := newRuntimeOptions(
		scope.PathsRuntimeOptions{ModulesPath: "/workspace/modules", DistPath: "/workspace/dist", DefaultChoysumPath: "/workspace/.choysum"},
		true,
		scope.AuthRuntimeOptions{
			GrpcAuthentication: false,
			GrpcMethodAccess:   false,
			GrpcRecordRule:     true,
			GrpcCompanyFilter:  false,
			GrpcFieldRule:      true,
			AuthzDecisionLog:   "deny",
			AuthzDecisionAudit: true,
		},
		true,
		scope.TaskRuntimeOptions{Task: &config.TaskConfig{Dispatch: &config.TaskDispatchConfig{DefaultMaxAttempts: 11}}},
		true,
		scope.CompileRuntimeOptions{SourceMap: false, Minify: true, TreeShaking: false},
		true,
		scope.RuntimeEnvironmentOptions{BackendEnv: map[string]any{"FEATURE_FLAG": "on"}},
		true,
	)
	if override.modulesPath != "/workspace/modules" || override.distPath != "/workspace/dist" || override.defaultChoysumPath != "/workspace/.choysum" {
		t.Fatalf("override path fields = %#v", override)
	}
	if override.grpcAuthentication || override.grpcMethodAccess || !override.grpcRecordRule || override.grpcCompanyFilter || !override.grpcFieldRule {
		t.Fatalf("override grpc flags = %#v", override)
	}
	if override.authzDecisionLog != "deny" || !override.authzDecisionAudit {
		t.Fatalf("override authz fields = %#v", override)
	}
	if override.taskDefaultMaxAttempt != 11 {
		t.Fatalf("taskDefaultMaxAttempt = %d, want 11", override.taskDefaultMaxAttempt)
	}
	if override.compileSourceMap || !override.compileMinify || override.compileTreeShaking {
		t.Fatalf("override compile flags = %#v", override)
	}
	if override.backendEnv["FEATURE_FLAG"] != "on" {
		t.Fatalf("backend env not propagated: %#v", override.backendEnv)
	}

	nilEnv := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{BackendEnv: nil}, true)
	if nilEnv.backendEnv == nil || len(nilEnv.backendEnv) != 0 {
		t.Fatalf("nil BackendEnv override should keep initialized empty map, got %#v", nilEnv.backendEnv)
	}

	nilTask := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, true, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false)
	if nilTask.taskDefaultMaxAttempt != 0 {
		t.Fatalf("taskDefaultMaxAttempt with nil task = %d, want 0", nilTask.taskDefaultMaxAttempt)
	}
	noDispatch := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{Task: &config.TaskConfig{}}, true, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false)
	if noDispatch.taskDefaultMaxAttempt != 0 {
		t.Fatalf("taskDefaultMaxAttempt with nil dispatch = %d, want 0", noDispatch.taskDefaultMaxAttempt)
	}

	cases := []struct {
		name string
		opts runtimeOptions
		msg  string
	}{
		{name: "missing modules", opts: runtimeOptions{distPath: "/dist", defaultChoysumPath: "/root"}, msg: "modulesPath"},
		{name: "missing dist", opts: runtimeOptions{modulesPath: "/modules", defaultChoysumPath: "/root"}, msg: "distPath"},
		{name: "missing default path", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist"}, msg: "defaultChoysumPath"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := (runtimeOptions{modulesPath: "/modules", distPath: "/dist", defaultChoysumPath: "/root"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func TestBackendRuntimeOptionsFromScopeAndResolved(t *testing.T) {
	t.Parallel()

	nilScope := runtimeOptionsFromScope(nil)
	if !nilScope.grpcAuthentication || !nilScope.grpcMethodAccess {
		t.Fatalf("runtimeOptionsFromScope(nil) should preserve default grpc flags: %#v", nilScope)
	}

	runtimeScope := newBuilderTestScope()
	fromScope := runtimeOptionsFromScope(runtimeScope)
	if fromScope.modulesPath != runtimeScope.cfg.ModulesPath || fromScope.distPath != runtimeScope.cfg.DistPath || fromScope.defaultChoysumPath != runtimeScope.cfg.DefaultChoysumPath {
		t.Fatalf("runtimeOptionsFromScope(paths) = %#v", fromScope)
	}
	if fromScope.authzDecisionLog != runtimeScope.cfg.Auth.AuthzDecisionLog || fromScope.taskDefaultMaxAttempt != runtimeScope.cfg.Task.Dispatch.DefaultMaxAttempts {
		t.Fatalf("runtimeOptionsFromScope(auth/task) = %#v", fromScope)
	}
	if fromScope.backendEnv["CUSTOM_FLAG"] != "present" {
		t.Fatalf("runtimeOptionsFromScope(env) = %#v", fromScope.backendEnv)
	}

	explicit := runtimeOptions{distPath: "/explicit/dist", modulesPath: "/explicit/modules", defaultChoysumPath: "/explicit/root"}
	builder := &ModuleBuilder{runtimeOptions: explicit, runtimeScope: runtimeScope}
	gotExplicit := builder.resolvedRuntimeOptions()
	if gotExplicit.distPath != explicit.distPath || gotExplicit.modulesPath != explicit.modulesPath || gotExplicit.defaultChoysumPath != explicit.defaultChoysumPath {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v", gotExplicit)
	}

	builder = &ModuleBuilder{runtimeScope: runtimeScope}
	gotScope := builder.resolvedRuntimeOptions()
	if gotScope.distPath != runtimeScope.cfg.DistPath || gotScope.modulesPath != runtimeScope.cfg.ModulesPath {
		t.Fatalf("resolvedRuntimeOptions(scope) = %#v", gotScope)
	}

	builder = &ModuleBuilder{runtimeOptions: runtimeOptions{modulesPath: "/only-runtime-field"}}
	gotRuntimeOnly := builder.resolvedRuntimeOptions()
	if gotRuntimeOnly.modulesPath != "/only-runtime-field" {
		t.Fatalf("resolvedRuntimeOptions(runtime-only) = %#v", gotRuntimeOnly)
	}

	var nilBuilder *ModuleBuilder
	gotNil := nilBuilder.resolvedRuntimeOptions()
	if !gotNil.grpcAuthentication || !gotNil.grpcMethodAccess {
		t.Fatalf("resolvedRuntimeOptions(nil builder) = %#v", gotNil)
	}

	if hasRuntimeOptions(runtimeOptions{distPath: "  "}) {
		t.Fatal("hasRuntimeOptions(blank distPath) should be false")
	}
	if !hasRuntimeOptions(runtimeOptions{distPath: "/dist"}) {
		t.Fatal("hasRuntimeOptions(non-blank distPath) should be true")
	}
}
