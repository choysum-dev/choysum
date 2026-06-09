// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestNewCliRuntimeOptionsConstructors(t *testing.T) {
	t.Parallel()

	empty := newCliRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	if empty != (cliRuntimeOptions{}) {
		t.Fatalf("newCliRuntimeOptions(hasPathOpts=false) = %#v, want zero value", empty)
	}

	pathOpts := scope.PathsRuntimeOptions{
		DefaultChoysumPath: "/workspace/.choysum",
		ModulesPath:        "/workspace/modules",
		NpmPath:            "/workspace/node_modules",
		TmpPath:            "/workspace/.choysum/tmp",
	}
	fromPath := newCliRuntimeOptions(pathOpts, true)
	if fromPath.defaultChoysumPath != pathOpts.DefaultChoysumPath || fromPath.modulesPath != pathOpts.ModulesPath || fromPath.npmPath != pathOpts.NpmPath || fromPath.tmpPath != pathOpts.TmpPath {
		t.Fatalf("newCliRuntimeOptions(hasPathOpts=true) = %#v, want fields from path opts", fromPath)
	}

	if got := newCliRuntimeOptionsFromScopeInputOptions(nil); got != (cliRuntimeOptions{}) {
		t.Fatalf("newCliRuntimeOptionsFromScopeInputOptions(nil) = %#v, want zero value", got)
	}

	fromScopeOptions := newCliRuntimeOptionsFromScopeInputOptions(&scopeInputConfigOptions{
		DefaultChoysumPath: "/default",
		ModulesPath:        "/modules",
		NpmPath:            "/npm",
		TmpPath:            "/tmp",
	})
	if fromScopeOptions.defaultChoysumPath != "/default" || fromScopeOptions.modulesPath != "/modules" || fromScopeOptions.npmPath != "/npm" || fromScopeOptions.tmpPath != "/tmp" {
		t.Fatalf("newCliRuntimeOptionsFromScopeInputOptions() = %#v, want values copied from scope options", fromScopeOptions)
	}
}

func TestCommandRuntimeScopeInputPathPriorityAndNilOptions(t *testing.T) {
	t.Parallel()

	input := newCommandRuntimeScopeInput(
		&scopeInputConfigOptions{
			ModulesPath:        "/options/modules",
			DistPath:           "/options/dist",
			TmpPath:            "/options/tmp",
			DefaultChoysumPath: "/options/default",
			ConfigPath:         "/options/config.yaml",
			NpmPath:            "/options/npm",
			NPMRegistryURL:     "https://registry.example.com",
			Server:             &config.ServerConfig{Environment: "production"},
		},
		cliRuntimeOptions{modulesPath: "/runtime/modules", tmpPath: "/runtime/tmp"},
	)

	if got := input.Environment(); got != "production" {
		t.Fatalf("Environment() = %q, want %q", got, "production")
	}
	if got := input.ModulesPath(); got != "/runtime/modules" {
		t.Fatalf("ModulesPath() runtime override = %q, want %q", got, "/runtime/modules")
	}
	if got := input.TmpPath(); got != "/runtime/tmp" {
		t.Fatalf("TmpPath() runtime override = %q, want %q", got, "/runtime/tmp")
	}
	if got := input.DistPath(); got != "/options/dist" {
		t.Fatalf("DistPath() = %q, want %q", got, "/options/dist")
	}
	if got := input.DefaultChoysumPath(); got != "/options/default" {
		t.Fatalf("DefaultChoysumPath() = %q, want %q", got, "/options/default")
	}
	if got := input.ConfigPath(); got != "/options/config.yaml" {
		t.Fatalf("ConfigPath() = %q, want %q", got, "/options/config.yaml")
	}
	if got := input.NpmPath(); got != "/options/npm" {
		t.Fatalf("NpmPath() = %q, want %q", got, "/options/npm")
	}
	if got := input.NpmRegistryURL(); got != "https://registry.example.com" {
		t.Fatalf("NpmRegistryURL() = %q, want %q", got, "https://registry.example.com")
	}

	fallback := newCommandRuntimeScopeInput(
		&scopeInputConfigOptions{ModulesPath: "/fallback/modules", TmpPath: "/fallback/tmp"},
		cliRuntimeOptions{},
	)
	if got := fallback.ModulesPath(); got != "/fallback/modules" {
		t.Fatalf("ModulesPath() fallback = %q, want %q", got, "/fallback/modules")
	}
	if got := fallback.TmpPath(); got != "/fallback/tmp" {
		t.Fatalf("TmpPath() fallback = %q, want %q", got, "/fallback/tmp")
	}

	nilOptionsInput := newCommandRuntimeScopeInput(nil, cliRuntimeOptions{modulesPath: "/runtime/modules", tmpPath: "/runtime/tmp"})
	if got := nilOptionsInput.ModulesPath(); got != "/runtime/modules" {
		t.Fatalf("ModulesPath() with nil options = %q, want %q", got, "/runtime/modules")
	}
	if got := nilOptionsInput.TmpPath(); got != "/runtime/tmp" {
		t.Fatalf("TmpPath() with nil options = %q, want %q", got, "/runtime/tmp")
	}
	if got := nilOptionsInput.Environment(); got != "" {
		t.Fatalf("Environment() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.DistPath(); got != "" {
		t.Fatalf("DistPath() with nil options = %q, want empty", got)
	}
	if got := nilOptionsInput.NpmRegistryURL(); got != "" {
		t.Fatalf("NpmRegistryURL() with nil options = %q, want empty", got)
	}
}

func TestRunRuntimeScopeInputPathFallbackAndRegistryURL(t *testing.T) {
	t.Parallel()

	input := newRunRuntimeScopeInput(
		&scopeInputConfigOptions{
			ModulesPath:    "/options/modules",
			TmpPath:        "/options/tmp",
			NPMRegistryURL: "https://registry.options.example",
		},
		cliRuntimeOptions{},
		runServerRuntimeOptions{},
		runDBRuntimeOptions{},
	)

	if got := input.ModulesPath(); got != "/options/modules" {
		t.Fatalf("ModulesPath() fallback = %q, want %q", got, "/options/modules")
	}
	if got := input.NpmRegistryURL(); got != "https://registry.options.example" {
		t.Fatalf("NpmRegistryURL() fallback = %q, want %q", got, "https://registry.options.example")
	}

	nilOptions := newRunRuntimeScopeInput(nil, cliRuntimeOptions{}, runServerRuntimeOptions{}, runDBRuntimeOptions{})
	if got := nilOptions.ModulesPath(); got != "" {
		t.Fatalf("ModulesPath() with nil options = %q, want empty", got)
	}
	if got := nilOptions.NpmRegistryURL(); got != "" {
		t.Fatalf("NpmRegistryURL() with nil options = %q, want empty", got)
	}
}

func TestRequireCliRuntimeOptionsAndValidate(t *testing.T) {
	t.Parallel()

	if _, err := requireCliRuntimeOptions(nil); err == nil || !strings.Contains(err.Error(), "getter is not initialized") {
		t.Fatalf("requireCliRuntimeOptions(nil) error = %v, want getter initialization error", err)
	}

	if _, err := requireCliRuntimeOptions(func() cliRuntimeOptions { return cliRuntimeOptions{} }); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath") {
		t.Fatalf("requireCliRuntimeOptions(invalid) error = %v, want defaultChoysumPath validation error", err)
	}

	valid := cliRuntimeOptions{
		defaultChoysumPath: "/workspace/.choysum",
		modulesPath:        "/workspace/modules",
		npmPath:            "/workspace/node_modules",
		tmpPath:            "/workspace/.choysum/tmp",
	}
	resolved, err := requireCliRuntimeOptions(func() cliRuntimeOptions { return valid })
	if err != nil {
		t.Fatalf("requireCliRuntimeOptions(valid) error = %v", err)
	}
	if resolved != valid {
		t.Fatalf("requireCliRuntimeOptions(valid) = %#v, want %#v", resolved, valid)
	}

	cases := []struct {
		name string
		opts cliRuntimeOptions
		msg  string
	}{
		{name: "missing default path", opts: cliRuntimeOptions{modulesPath: "/modules", npmPath: "/npm", tmpPath: "/tmp"}, msg: "defaultChoysumPath"},
		{name: "missing modules", opts: cliRuntimeOptions{defaultChoysumPath: "/root", npmPath: "/npm", tmpPath: "/tmp"}, msg: "modulesPath"},
		{name: "missing npm", opts: cliRuntimeOptions{defaultChoysumPath: "/root", modulesPath: "/modules", tmpPath: "/tmp"}, msg: "npmPath"},
		{name: "missing tmp", opts: cliRuntimeOptions{defaultChoysumPath: "/root", modulesPath: "/modules", npmPath: "/npm"}, msg: "tmpPath"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
}
