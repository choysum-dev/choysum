// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubScope struct {
	cfg *config.Config
}

func (e *stubScope) Run(func(scope.Scope) error) error { return nil }
func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *stubScope) Session() *scope.Session                     { return nil }
func (e *stubScope) WithContext(ctx context.Context) scope.Scope { return e }
func (e *stubScope) Context() context.Context                    { return context.Background() }
func (e *stubScope) Logger() *slog.Logger                        { return nil }
func (e *stubScope) Config() *config.Config                      { return e.cfg }
func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func TestIsCoreModuleAndModuleSpecs(t *testing.T) {
	runtimeScope := &stubScope{cfg: &config.Config{ModulesPath: filepath.Join("tmp", "modules")}}

	if !IsCoreModule("core") {
		t.Fatal("expected core module to be recognized")
	}
	if IsCoreModule("auth") {
		t.Fatal("did not expect non-core module to be recognized as core")
	}

	checks := []struct {
		name      string
		path      string
		reference string
	}{
		{name: "ModelDecoratorModuleSpec", path: filepath.Join(runtimeScope.cfg.ModulesPath, "core", "service", "orm", "decorator", "model"), reference: "Model"},
		{name: "FieldDecoratorModuleSpec", path: filepath.Join(runtimeScope.cfg.ModulesPath, "core", "service", "orm", "decorator", "field"), reference: "Field"},
		{name: "ServiceDecoratorModuleSpec", path: filepath.Join(runtimeScope.cfg.ModulesPath, "core", "service", "orm", "decorator", "service"), reference: "Service"},
		{name: "XpathComponentModuleSpec", path: filepath.Join(runtimeScope.cfg.ModulesPath, "core", "web", "component", "xpath.vue"), reference: "default"},
		{name: "BaseModelModuleSpec", path: filepath.Join(runtimeScope.cfg.ModulesPath, "core", "service", "orm", "model", "model"), reference: "BaseModel"},
	}

	actual := []struct {
		path string
		ref  string
	}{
		func() struct{ path, ref string } {
			p, r := ModelDecoratorModuleSpec(runtimeScope)
			return struct{ path, ref string }{p, r}
		}(),
		func() struct{ path, ref string } {
			p, r := FieldDecoratorModuleSpec(runtimeScope)
			return struct{ path, ref string }{p, r}
		}(),
		func() struct{ path, ref string } {
			p, r := ServiceDecoratorModuleSpec(runtimeScope)
			return struct{ path, ref string }{p, r}
		}(),
		func() struct{ path, ref string } {
			p, r := XpathComponentModuleSpec(runtimeScope)
			return struct{ path, ref string }{p, r}
		}(),
		func() struct{ path, ref string } {
			p, r := BaseModelModuleSpec(runtimeScope)
			return struct{ path, ref string }{p, r}
		}(),
	}

	for index, check := range checks {
		if actual[index].path != check.path || actual[index].ref != check.reference {
			t.Fatalf("%s() = (%q, %q), want (%q, %q)", check.name, actual[index].path, actual[index].ref, check.path, check.reference)
		}
	}
}

func TestIsConventionalModelService(t *testing.T) {
	tests := []struct {
		name          string
		accessibility string
		isStatic      bool
		serviceName   string
		want          bool
	}{
		{name: "public static upper-case", accessibility: "public", isStatic: true, serviceName: "Search", want: true},
		{name: "public static lower-case", accessibility: "public", isStatic: true, serviceName: "search", want: false},
		{name: "private static upper-case", accessibility: "private", isStatic: true, serviceName: "Search", want: false},
		{name: "public instance upper-case", accessibility: "public", isStatic: false, serviceName: "Search", want: false},
		{name: "empty name", accessibility: "public", isStatic: true, serviceName: "", want: false},
	}

	for _, tt := range tests {
		if got := IsConventionalModelService(tt.accessibility, tt.isStatic, tt.serviceName); got != tt.want {
			t.Fatalf("%s: IsConventionalModelService(%q, %v, %q) = %v, want %v", tt.name, tt.accessibility, tt.isStatic, tt.serviceName, got, tt.want)
		}
	}
}
