// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"reflect"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestNewFactoryWithRuntimePluginsAppliesPlugins(t *testing.T) {
	plugins := []jsengine.RuntimePlugin{
		jsengine.NewRuntimePlugin(RuntimePluginXid, func(scope.Scope, auth.Authenticator) []jsengine.JsEngineOption {
			return []jsengine.JsEngineOption{WithXid()}
		}),
	}

	factory := NewFactoryWithRuntimePlugins(nil, nil, plugins)
	engine, err := factory()
	if err != nil {
		t.Fatalf("NewFactoryWithRuntimePlugins() error = %v", err)
	}
	defer func() { _ = engine.Close() }()

	quickjs, ok := engine.(*QuickjsEngine)
	if !ok {
		t.Fatalf("expected *QuickjsEngine, got %T", engine)
	}
	if !evalBool(t, quickjs, `typeof $choysum.xid.New === "function"`) {
		t.Fatal("expected plugin-installed xid helper")
	}
}

func TestRuntimePluginNamePolicies(t *testing.T) {
	defaultNames := DefaultRuntimePluginNames()
	replaceableNames := ReplaceableRuntimePluginNames()
	defaultOnlyNames := DefaultOnlyRuntimePluginNames()

	if !reflect.DeepEqual(defaultNames, []string{
		RuntimePluginConsole,
		RuntimePluginXid,
		RuntimePluginDB,
		RuntimePluginAuth,
		RuntimePluginCrypto,
		RuntimePluginHtml,
		RuntimePluginFS,
		RuntimePluginI18n,
		RuntimePluginImport,
		RuntimePluginExport,
		RuntimePluginGRPC,
		RuntimePluginBus,
		RuntimePluginDocumentStorage,
		RuntimePluginModuleManagement,
		RuntimePluginScriptVueSFC,
		RuntimePluginScriptChoysumRPC,
	}) {
		t.Fatalf("DefaultRuntimePluginNames() = %#v", defaultNames)
	}

	if !reflect.DeepEqual(replaceableNames, []string{
		RuntimePluginConsole,
		RuntimePluginXid,
		RuntimePluginDB,
		RuntimePluginAuth,
		RuntimePluginCrypto,
		RuntimePluginHtml,
		RuntimePluginFS,
		RuntimePluginI18n,
		RuntimePluginImport,
		RuntimePluginExport,
		RuntimePluginGRPC,
		RuntimePluginBus,
		RuntimePluginDocumentStorage,
		RuntimePluginModuleManagement,
	}) {
		t.Fatalf("ReplaceableRuntimePluginNames() = %#v", replaceableNames)
	}

	if !reflect.DeepEqual(defaultOnlyNames, []string{
		RuntimePluginScriptVueSFC,
		RuntimePluginScriptChoysumRPC,
	}) {
		t.Fatalf("DefaultOnlyRuntimePluginNames() = %#v", defaultOnlyNames)
	}

	for _, name := range replaceableNames {
		if !IsReplaceableRuntimePlugin(name) {
			t.Fatalf("expected %q to be replaceable", name)
		}
		if IsDefaultOnlyRuntimePlugin(name) {
			t.Fatalf("expected %q to not be default-only", name)
		}
	}

	for _, name := range defaultOnlyNames {
		if !IsDefaultOnlyRuntimePlugin(name) {
			t.Fatalf("expected %q to be default-only", name)
		}
		if IsReplaceableRuntimePlugin(name) {
			t.Fatalf("expected %q to not be replaceable", name)
		}
	}

	if IsReplaceableRuntimePlugin("missing") {
		t.Fatal("unexpected replaceable policy for unknown plugin")
	}
	if IsDefaultOnlyRuntimePlugin("missing") {
		t.Fatal("unexpected default-only policy for unknown plugin")
	}
}

func TestMergeRuntimePluginOverridesCompatibilityKeepsMixedInputBehavior(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) jsengine.RuntimePlugin {
		return jsengine.NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []jsengine.JsEngineOption {
			called = append(called, marker)
			return []jsengine.JsEngineOption{func(jsengine.JsEngine) error { return nil }}
		})
	}

	merged := MergeRuntimePluginOverrides(
		[]jsengine.RuntimePlugin{
			makePlugin(RuntimePluginConsole, "base-console"),
			makePlugin(RuntimePluginScriptVueSFC, "base-vuesfc"),
			makePlugin(RuntimePluginGRPC, "base-grpc"),
		},
		makePlugin(RuntimePluginConsole, "override-console"),
		makePlugin(RuntimePluginScriptVueSFC, "override-vuesfc"),
		makePlugin("custom:analytics", "extra-custom"),
	)

	if len(merged) != 5 {
		t.Fatalf("merged plugin count = %d, want 5", len(merged))
	}

	jsengine.BuildRuntimePluginOptions(nil, nil, merged...)
	if !reflect.DeepEqual(called, []string{"override-console", "base-vuesfc", "base-grpc", "override-vuesfc", "extra-custom"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestApplyRuntimePluginAssemblySeparatesReplacementsFromExtras(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) jsengine.RuntimePlugin {
		return jsengine.NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []jsengine.JsEngineOption {
			called = append(called, marker)
			return []jsengine.JsEngineOption{func(jsengine.JsEngine) error { return nil }}
		})
	}

	assembled := ApplyRuntimePluginAssembly(
		[]jsengine.RuntimePlugin{
			makePlugin(RuntimePluginConsole, "base-console"),
			makePlugin(RuntimePluginScriptVueSFC, "base-vuesfc"),
			makePlugin(RuntimePluginGRPC, "base-grpc"),
		},
		jsengine.RuntimePluginAssembly{
			Replacements: []jsengine.RuntimePlugin{
				makePlugin(RuntimePluginConsole, "replacement-console"),
				makePlugin(RuntimePluginScriptVueSFC, "ignored-vuesfc-replacement"),
				nil,
			},
			Extras: []jsengine.RuntimePlugin{
				nil,
				makePlugin(RuntimePluginScriptVueSFC, "extra-vuesfc"),
				makePlugin("custom:analytics", "extra-custom"),
			},
		},
	)

	if len(assembled) != 5 {
		t.Fatalf("assembled plugin count = %d, want 5", len(assembled))
	}

	jsengine.BuildRuntimePluginOptions(nil, nil, assembled...)
	if !reflect.DeepEqual(called, []string{"replacement-console", "base-vuesfc", "base-grpc", "extra-vuesfc", "extra-custom"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestAssembleRuntimePluginsCompatibilityKeepsSliceAssemblyBehavior(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) jsengine.RuntimePlugin {
		return jsengine.NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []jsengine.JsEngineOption {
			called = append(called, marker)
			return []jsengine.JsEngineOption{func(jsengine.JsEngine) error { return nil }}
		})
	}

	assembled := AssembleRuntimePlugins(
		[]jsengine.RuntimePlugin{
			makePlugin(RuntimePluginConsole, "base-console"),
			makePlugin(RuntimePluginScriptVueSFC, "base-vuesfc"),
			makePlugin(RuntimePluginGRPC, "base-grpc"),
		},
		[]jsengine.RuntimePlugin{
			makePlugin(RuntimePluginConsole, "replacement-console"),
			makePlugin(RuntimePluginScriptVueSFC, "ignored-vuesfc-replacement"),
			nil,
		},
		[]jsengine.RuntimePlugin{
			nil,
			makePlugin(RuntimePluginScriptVueSFC, "extra-vuesfc"),
			makePlugin("custom:analytics", "extra-custom"),
		},
	)

	if len(assembled) != 5 {
		t.Fatalf("assembled plugin count = %d, want 5", len(assembled))
	}

	jsengine.BuildRuntimePluginOptions(nil, nil, assembled...)
	if !reflect.DeepEqual(called, []string{"replacement-console", "base-vuesfc", "base-grpc", "extra-vuesfc", "extra-custom"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}
