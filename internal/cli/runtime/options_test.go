// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type optionsScopeStub struct {
	ctx   context.Context
	input scope.FactoryInput
}

func (s *optionsScopeStub) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *optionsScopeStub) Session() *scope.Session              { return nil }
func (s *optionsScopeStub) Transactor() scope.Transactor         { return nil }
func (s *optionsScopeStub) WithContext(ctx context.Context) scope.Scope {
	return &optionsScopeStub{ctx: ctx, input: s.input}
}
func (s *optionsScopeStub) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *optionsScopeStub) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (s *optionsScopeStub) FactoryInput() scope.FactoryInput {
	return s.input
}

func TestNewOptions(t *testing.T) {
	t.Parallel()

	empty := NewOptions(scope.PathsRuntimeOptions{}, false)
	if empty != (Options{}) {
		t.Fatalf("NewOptions(hasPathOpts=false) = %#v, want zero value", empty)
	}

	pathOpts := scope.PathsRuntimeOptions{
		DefaultChoysumPath:    "/workspace/.choysum",
		ModulesPath:           "/workspace/modules",
		TmpPath:               "/workspace/.choysum/tmp",
		ModuleCatalogIndexURL: " https://index.example.com/v1/index.json ",
	}
	fromPath := NewOptions(pathOpts, true)
	if fromPath.DefaultChoysumPath != pathOpts.DefaultChoysumPath || fromPath.ModulesPath != pathOpts.ModulesPath || fromPath.TmpPath != pathOpts.TmpPath || fromPath.ModuleCatalogIndexURL != "https://index.example.com/v1/index.json" {
		t.Fatalf("NewOptions(hasPathOpts=true) = %#v, want fields from path opts", fromPath)
	}
}

func TestOptionsFromScopeNil(t *testing.T) {
	t.Parallel()

	if got := OptionsFromScope(nil); got != (Options{}) {
		t.Fatalf("OptionsFromScope(nil) = %#v, want zero value", got)
	}
}

func TestOptionsFromScope(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DefaultChoysumPath:    "/workspace/.choysum",
		ModulesPath:           "/workspace/modules",
		TmpPath:               "/workspace/.choysum/tmp",
		ModuleCatalogIndexURL: "https://index.example.com/v1/index.json",
	}
	configOptions := NewScopeInputConfigOptions(snapshot.New(cfg))
	runtimeOptions := Options{
		DefaultChoysumPath:    configOptions.DefaultChoysumPath,
		ModulesPath:           configOptions.ModulesPath,
		TmpPath:               configOptions.TmpPath,
		ModuleCatalogIndexURL: strings.TrimSpace(configOptions.ModuleCatalogIndexURL),
	}
	runtimeScope := &optionsScopeStub{input: NewCommandScopeInput(configOptions, runtimeOptions)}

	got := OptionsFromScope(runtimeScope)
	if got != runtimeOptions {
		t.Fatalf("OptionsFromScope() = %#v, want %#v", got, runtimeOptions)
	}
}

func TestValidateAndRequireOptions(t *testing.T) {
	t.Parallel()

	if _, err := RequireOptions(nil); err == nil || !strings.Contains(err.Error(), "getter is not initialized") {
		t.Fatalf("RequireOptions(nil) error = %v, want getter initialization error", err)
	}

	if _, err := RequireOptions(func() Options { return Options{} }); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath") {
		t.Fatalf("RequireOptions(invalid) error = %v, want defaultChoysumPath validation error", err)
	}

	valid := Options{
		DefaultChoysumPath: "/workspace/.choysum",
		ModulesPath:        "/workspace/modules",
		TmpPath:            "/workspace/.choysum/tmp",
	}
	resolved, err := RequireOptions(func() Options { return valid })
	if err != nil {
		t.Fatalf("RequireOptions(valid) error = %v", err)
	}
	if resolved != valid {
		t.Fatalf("RequireOptions(valid) = %#v, want %#v", resolved, valid)
	}

	cases := []struct {
		name string
		opts Options
		msg  string
	}{
		{name: "missing default path", opts: Options{ModulesPath: "/modules", TmpPath: "/tmp"}, msg: "defaultChoysumPath"},
		{name: "missing modules", opts: Options{DefaultChoysumPath: "/root", TmpPath: "/tmp"}, msg: "modulesPath"},
		{name: "missing tmp", opts: Options{DefaultChoysumPath: "/root", ModulesPath: "/modules"}, msg: "tmpPath"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opts.Validate(); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("Validate() expected %q error, got %v", tc.msg, err)
			}
		})
	}

	invalidCatalog := valid
	invalidCatalog.ModuleCatalogIndexURL = "https://index.example.dev/v1/catalog.json"
	if err := invalidCatalog.Validate(); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("Validate(invalid moduleCatalogIndexURL) error = %v, want index.json validation error", err)
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	if _, err := RequireOptionsForCommand("typecheck", nil); err == nil || !strings.Contains(err.Error(), "typecheck: invalid runtime options") {
		t.Fatalf("RequireOptionsForCommand(nil) error = %v, want prefixed runtime options error", err)
	}

	if _, err := RequireOptionsForCommand("", func() Options { return Options{} }); err == nil || !strings.Contains(err.Error(), "command: invalid runtime options") {
		t.Fatalf("RequireOptionsForCommand(empty command) error = %v, want fallback command prefix", err)
	}

	resolvedWithPrefix, err := RequireOptionsForCommand("e2e", func() Options { return valid })
	if err != nil {
		t.Fatalf("RequireOptionsForCommand(valid) error = %v", err)
	}
	if resolvedWithPrefix != valid {
		t.Fatalf("RequireOptionsForCommand(valid) = %#v, want %#v", resolvedWithPrefix, valid)
	}
}

func TestResolveModuleCatalogIndexURL(t *testing.T) {
	t.Parallel()

	defaultURL, err := ResolveModuleCatalogIndexURL(Options{})
	if err != nil {
		t.Fatalf("ResolveModuleCatalogIndexURL(default) error = %v", err)
	}
	if defaultURL != config.DefaultModuleCatalogIndexURL {
		t.Fatalf("ResolveModuleCatalogIndexURL(default) = %q, want %q", defaultURL, config.DefaultModuleCatalogIndexURL)
	}

	customURL, err := ResolveModuleCatalogIndexURL(Options{ModuleCatalogIndexURL: " https://index.acme.dev/v1/index.json "})
	if err != nil {
		t.Fatalf("ResolveModuleCatalogIndexURL(custom) error = %v", err)
	}
	if customURL != "https://index.acme.dev/v1/index.json" {
		t.Fatalf("ResolveModuleCatalogIndexURL(custom) = %q, want %q", customURL, "https://index.acme.dev/v1/index.json")
	}

	if _, err := ResolveModuleCatalogIndexURL(Options{ModuleCatalogIndexURL: "https://index.acme.dev/v1/catalog.json"}); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("ResolveModuleCatalogIndexURL(invalid) error = %v, want validation error", err)
	}
}
