// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestLifecycleRuntimeOptionsNewAndValidate(t *testing.T) {
	t.Parallel()

	defaults := config.NewDefaultCompileConfig()
	base := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false)
	if base.compileBundleMode != defaults.BundleMode {
		t.Fatalf("default compile bundle mode = %q, want %q", base.compileBundleMode, defaults.BundleMode)
	}

	override := newRuntimeOptions(
		scope.PathsRuntimeOptions{
			ModulesPath:        "/workspace/modules",
			DistPath:           "/workspace/dist",
			TmpPath:            "/workspace/tmp",
			DefaultChoysumPath: "/workspace/.choysum",
		},
		true,
		scope.CompileRuntimeOptions{BundleMode: "application"},
		true,
	)
	if override.modulesPath != "/workspace/modules" || override.distPath != "/workspace/dist" || override.tmpPath != "/workspace/tmp" || override.defaultChoysumPath != "/workspace/.choysum" || override.compileBundleMode != "application" {
		t.Fatalf("override runtime options = %#v, want path and compile overrides", override)
	}

	blankCompile := newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{BundleMode: "   "}, true)
	if blankCompile.compileBundleMode != defaults.BundleMode {
		t.Fatalf("blank compile bundle mode should keep default, got %q", blankCompile.compileBundleMode)
	}

	cases := []struct {
		name string
		opts runtimeOptions
		msg  string
	}{
		{name: "missing modules", opts: runtimeOptions{distPath: "/dist", tmpPath: "/tmp", defaultChoysumPath: "/root", compileBundleMode: "bundle"}, msg: "modulesPath"},
		{name: "missing dist", opts: runtimeOptions{modulesPath: "/modules", tmpPath: "/tmp", defaultChoysumPath: "/root", compileBundleMode: "bundle"}, msg: "distPath"},
		{name: "missing tmp", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", defaultChoysumPath: "/root", compileBundleMode: "bundle"}, msg: "tmpPath"},
		{name: "missing default path", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", tmpPath: "/tmp", compileBundleMode: "bundle"}, msg: "defaultChoysumPath"},
		{name: "missing compile mode", opts: runtimeOptions{modulesPath: "/modules", distPath: "/dist", tmpPath: "/tmp", defaultChoysumPath: "/root"}, msg: "compileBundleMode"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := (runtimeOptions{
		modulesPath:        "/modules",
		distPath:           "/dist",
		tmpPath:            "/tmp",
		defaultChoysumPath: "/root",
		compileBundleMode:  "bundle",
	}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}

func TestLifecycleRuntimeOptionsFromScopeAndResolved(t *testing.T) {
	t.Parallel()

	defaults := config.NewDefaultCompileConfig()
	nilScopeOpts := runtimeOptionsFromScope(nil)
	if nilScopeOpts.compileBundleMode != defaults.BundleMode {
		t.Fatalf("runtimeOptionsFromScope(nil).compileBundleMode = %q, want %q", nilScopeOpts.compileBundleMode, defaults.BundleMode)
	}

	runtimeScope := newModuleIndexSyncScope("/workspace/modules", nil)
	runtimeScope.cfg.DistPath = "/workspace/dist"
	runtimeScope.cfg.TmpPath = "/workspace/tmp"
	runtimeScope.cfg.DefaultChoysumPath = "/workspace/.choysum"
	runtimeScope.cfg.Compile = &config.CompileConfig{BundleMode: "application"}

	fromScope := runtimeOptionsFromScope(runtimeScope)
	if fromScope.modulesPath != "/workspace/modules" || fromScope.distPath != "/workspace/dist" || fromScope.tmpPath != "/workspace/tmp" || fromScope.defaultChoysumPath != "/workspace/.choysum" || fromScope.compileBundleMode != "application" {
		t.Fatalf("runtimeOptionsFromScope() = %#v, want values from scope config", fromScope)
	}

	explicit := runtimeOptions{compileBundleMode: "bundle", modulesPath: "/explicit/modules"}
	manager := &ModuleManager{runtimeOptions: explicit, runtimeScope: runtimeScope}
	if got := manager.resolvedRuntimeOptions(); got != explicit {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v, want %#v", got, explicit)
	}

	manager = &ModuleManager{runtimeScope: runtimeScope}
	if got := manager.resolvedRuntimeOptions(); got.modulesPath != "/workspace/modules" || got.compileBundleMode != "application" {
		t.Fatalf("resolvedRuntimeOptions(scope fallback) = %#v", got)
	}

	manager = &ModuleManager{runtimeOptions: runtimeOptions{modulesPath: "/only-runtime-field"}}
	if got := manager.resolvedRuntimeOptions(); got.modulesPath != "/only-runtime-field" {
		t.Fatalf("resolvedRuntimeOptions(non-nil manager without scope) = %#v", got)
	}

	var nilManager *ModuleManager
	if got := nilManager.resolvedRuntimeOptions(); got.compileBundleMode != defaults.BundleMode {
		t.Fatalf("resolvedRuntimeOptions(nil manager).compileBundleMode = %q, want %q", got.compileBundleMode, defaults.BundleMode)
	}

	if hasRuntimeOptions(runtimeOptions{compileBundleMode: "   "}) {
		t.Fatal("hasRuntimeOptions(blank) should be false")
	}
	if !hasRuntimeOptions(runtimeOptions{compileBundleMode: "bundle"}) {
		t.Fatal("hasRuntimeOptions(non-blank) should be true")
	}
}
