// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultengine

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type defaultEngineTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (e *defaultEngineTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *defaultEngineTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *defaultEngineTestScope) Session() *scope.Session { return nil }
func (e *defaultEngineTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *defaultEngineTestScope) Context() context.Context { return e.ctx }
func (e *defaultEngineTestScope) Logger() *slog.Logger     { return e.logger }
func (e *defaultEngineTestScope) Config() *config.Config   { return e.cfg }
func (e *defaultEngineTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func TestQuickjsFactoryRegisteredAndConstructible(t *testing.T) {
	if !jsengine.Exists("quickjs") {
		t.Fatal("expected quickjs engine factory to be registered")
	}

	runtimeScope := &defaultEngineTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			Db:     &config.DbConfig{Dialect: "sqlite"},
			Server: config.NewDefaultServerConfig(),
		},
	}

	factory := QuickjsFactory(runtimeScope, nil)
	if factory == nil {
		t.Fatal("expected QuickjsFactory to return a non-nil factory")
	}

	byName := jsengine.NewByName("quickjs")
	if byName == nil {
		t.Fatal("expected quickjs factory lookup by name to succeed")
	}
	if built := byName(jsengine.StaticScopeProvider(runtimeScope), nil); built == nil {
		t.Fatal("expected named quickjs factory to be constructible")
	}
	if jsengine.NewByName("missing") != nil {
		t.Fatal("expected missing engine lookup to return nil")
	}
}

func TestDefaultQuickjsRuntimePluginsMatchPublishedNames(t *testing.T) {
	plugins := defaultQuickjsRuntimePlugins()
	names := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			t.Fatal("expected non-nil runtime plugin")
		}
		names = append(names, plugin.Name())
	}

	if !reflect.DeepEqual(names, quickjsengine.DefaultRuntimePluginNames()) {
		t.Fatalf("default quickjs runtime plugin names = %#v, want %#v", names, quickjsengine.DefaultRuntimePluginNames())
	}
}

func TestDefaultQuickjsRuntimePluginPolicies(t *testing.T) {
	for _, plugin := range defaultQuickjsReplaceableRuntimePlugins() {
		if !quickjsengine.IsReplaceableRuntimePlugin(plugin.Name()) {
			t.Fatalf("expected %q to be replaceable", plugin.Name())
		}
		if quickjsengine.IsDefaultOnlyRuntimePlugin(plugin.Name()) {
			t.Fatalf("expected %q to not be default-only", plugin.Name())
		}
	}

	for _, plugin := range defaultQuickjsDefaultOnlyRuntimePlugins() {
		if !quickjsengine.IsDefaultOnlyRuntimePlugin(plugin.Name()) {
			t.Fatalf("expected %q to be default-only", plugin.Name())
		}
		if quickjsengine.IsReplaceableRuntimePlugin(plugin.Name()) {
			t.Fatalf("expected %q to not be replaceable", plugin.Name())
		}
	}
}

func TestQuickjsRuntimePluginsCompatibilityKeepsMixedInputBehavior(t *testing.T) {
	plugins := QuickjsRuntimePlugins(
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginConsole, nil),
		jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginScriptVueSFC, nil),
		jsengine.NewRuntimePlugin("custom:analytics", nil),
	)

	names := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			t.Fatal("expected non-nil runtime plugin")
		}
		names = append(names, plugin.Name())
	}

	defaults := quickjsengine.DefaultRuntimePluginNames()
	if !reflect.DeepEqual(names[:len(defaults)], defaults) {
		t.Fatalf("default quickjs runtime plugin prefix = %#v, want %#v", names[:len(defaults)], defaults)
	}
	if !reflect.DeepEqual(names[len(defaults):], []string{quickjsengine.RuntimePluginScriptVueSFC, "custom:analytics"}) {
		t.Fatalf("override suffix = %#v, want default-only override plus custom plugin", names[len(defaults):])
	}
}

func TestQuickjsRuntimePluginAssemblySeparatesReplacementsAndExtras(t *testing.T) {
	plugins := QuickjsRuntimePluginAssembly(jsengine.RuntimePluginAssembly{
		Replacements: []jsengine.RuntimePlugin{
			jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginConsole, nil),
			jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginScriptVueSFC, nil),
		},
		Extras: []jsengine.RuntimePlugin{
			jsengine.NewRuntimePlugin(quickjsengine.RuntimePluginScriptVueSFC, nil),
			jsengine.NewRuntimePlugin("custom:analytics", nil),
		},
	})

	names := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin == nil {
			t.Fatal("expected non-nil runtime plugin")
		}
		names = append(names, plugin.Name())
	}

	defaults := quickjsengine.DefaultRuntimePluginNames()
	if !reflect.DeepEqual(names[:len(defaults)], defaults) {
		t.Fatalf("default quickjs runtime plugin prefix = %#v, want %#v", names[:len(defaults)], defaults)
	}
	if !reflect.DeepEqual(names[len(defaults):], []string{quickjsengine.RuntimePluginScriptVueSFC, "custom:analytics"}) {
		t.Fatalf("assembly suffix = %#v, want extra plugins only", names[len(defaults):])
	}
}

func TestQuickjsFactoryWithRuntimePluginAssemblyBuildsFactory(t *testing.T) {
	runtimeScope := &defaultEngineTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			Db:     &config.DbConfig{Dialect: "sqlite"},
			Server: config.NewDefaultServerConfig(),
		},
	}

	factory := QuickjsFactoryWithRuntimePluginAssembly(
		runtimeScope,
		nil,
		jsengine.RuntimePluginAssembly{Extras: []jsengine.RuntimePlugin{
			jsengine.NewRuntimePlugin("custom:marker", func(scope.Scope, auth.Authenticator) []jsengine.JsEngineOption {
				return []jsengine.JsEngineOption{quickjsengine.WithScript(&jsengine.JsScript{
					FileName: "scripts/custom/marker.js",
					Content: `(function() {
						globalThis.$choysum = globalThis.$choysum || {};
						globalThis.$choysum.__rpc__ = async function(req) {
							return {
								id: req.id,
								result: { service: req.service },
								context: { customMarker: true }
							};
						};
					})();`,
				})}
			}),
		}},
	)
	if factory == nil {
		t.Fatal("expected QuickjsFactoryWithRuntimePluginAssembly to return a non-nil factory")
	}

	engine, err := factory()
	if err != nil {
		t.Fatalf("QuickjsFactoryWithRuntimePluginAssembly() error = %v", err)
	}
	defer func() { _ = engine.Close() }()

	resp, err := engine.Execute(context.Background(), &jsengine.JsRequest{Id: "req-1", Service: "demo"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if marked, ok := resp.Context["customMarker"].(bool); !ok || !marked {
		t.Fatalf("expected custom override marker in response context, got %#v", resp.Context)
	}
}
