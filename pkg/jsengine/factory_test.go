// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsengine

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (e *stubScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *stubScope) Session() *scope.Session { return nil }

func (e *stubScope) WithContext(ctx context.Context) scope.Scope {
	return &stubScope{ctx: ctx, cfg: e.cfg, logger: e.logger}
}

func (e *stubScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *stubScope) Logger() *slog.Logger { return e.logger }

func (e *stubScope) Config() *config.Config { return e.cfg }
func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

type stubAuthenticator struct{}

func (stubAuthenticator) ValidateToken(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
	return nil, nil
}

func (stubAuthenticator) CreateTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}

func (stubAuthenticator) RefreshTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}

func (stubAuthenticator) RevokeToken(context.Context, string, string) error { return nil }

func (stubAuthenticator) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 0, nil
}

func (stubAuthenticator) Close() error { return nil }

func snapshotFactories() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()
	clone := make(map[string]Factory, len(factories))
	for name, factory := range factories {
		clone[name] = factory
	}
	return clone
}

func restoreFactories(snapshot map[string]Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories = snapshot
}

func newStubScope() scope.Scope {
	return &stubScope{
		ctx:    context.Background(),
		cfg:    &config.Config{Server: &config.ServerConfig{JsEngineFactory: "chosen"}},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestRegisterExistsKeysAndNewByName(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	alpha := func(ScopeProvider, auth.Authenticator, ...JsEngineOption) JsEngineFactory {
		return func() (JsEngine, error) { return nil, nil }
	}
	beta := func(ScopeProvider, auth.Authenticator, ...JsEngineOption) JsEngineFactory {
		return func() (JsEngine, error) { return nil, nil }
	}

	Register("alpha", alpha)
	Register("beta", beta)

	if !Exists("alpha") || !Exists("beta") {
		t.Fatal("expected registered factories to exist")
	}
	if Exists("missing") {
		t.Fatal("did not expect missing factory to exist")
	}

	keys := Keys()
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"alpha", "beta"}) {
		t.Fatalf("Keys() = %#v, want alpha/beta", keys)
	}

	if got := NewByName("alpha"); reflect.ValueOf(got).Pointer() != reflect.ValueOf(alpha).Pointer() {
		t.Fatal("expected NewByName(alpha) to return the registered factory")
	}
	if got := NewByName("missing"); got != nil {
		t.Fatalf("expected missing factory lookup to return nil, got %#v", got)
	}
}

func TestNewJsEngineFactoryUsesConfiguredFactory(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	runtimeScope := newStubScope()
	authenticator := stubAuthenticator{}
	called := 0
	var gotProvider ScopeProvider
	var gotAuthenticator auth.Authenticator
	wantFactory := func() (JsEngine, error) { return nil, nil }

	Register("chosen", func(passedProvider ScopeProvider, passedAuth auth.Authenticator, options ...JsEngineOption) JsEngineFactory {
		called++
		gotProvider = passedProvider
		gotAuthenticator = passedAuth
		if len(options) != 0 {
			t.Fatalf("unexpected options: %#v", options)
		}
		return wantFactory
	})

	gotFactory := NewJsEngineFactory(runtimeScope, authenticator)
	if gotFactory == nil {
		t.Fatal("expected non-nil JsEngineFactory")
	}
	if called != 1 {
		t.Fatalf("factory call count = %d, want 1", called)
	}
	if gotProvider == nil {
		t.Fatal("expected scope provider to be passed to factory")
	}
	if gotScope := ResolveScope(gotProvider, nil); gotScope != runtimeScope {
		t.Fatal("expected nil runtime ctx to resolve the original scope")
	}
	runtimeCtx := context.WithValue(context.Background(), struct{}{}, "runtime")
	resolvedScope := ResolveScope(gotProvider, runtimeCtx)
	runtimeStub, ok := resolvedScope.(*stubScope)
	if !ok {
		t.Fatalf("expected resolved runtime scope to be *stubScope, got %T", resolvedScope)
	}
	if runtimeStub == runtimeScope {
		t.Fatal("expected runtime ctx resolution to clone the scope")
	}
	if runtimeStub.Context() != runtimeCtx {
		t.Fatal("expected runtime ctx to be rebound through WithContext")
	}
	if _, ok := gotAuthenticator.(stubAuthenticator); !ok {
		t.Fatalf("unexpected authenticator passed to factory: %#v", gotAuthenticator)
	}
	if reflect.ValueOf(gotFactory).Pointer() != reflect.ValueOf(wantFactory).Pointer() {
		t.Fatal("expected returned JsEngineFactory to match factory result")
	}
	if _, err := gotFactory(); err != nil {
		t.Fatalf("returned JsEngineFactory error: %v", err)
	}
}

func TestBuildRuntimePluginOptionsWithProviderUsesRuntimeContext(t *testing.T) {
	baseScope, ok := newStubScope().(*stubScope)
	if !ok {
		t.Fatalf("expected *stubScope, got %T", newStubScope())
	}
	runtimeCtx := context.WithValue(context.Background(), struct{}{}, "runtime")
	var gotScope scope.Scope

	plugins := []RuntimePlugin{
		NewRuntimePluginWithProvider("ctx-aware", func(provider ScopeProvider, authenticator auth.Authenticator) []JsEngineOption {
			gotScope = ResolveScope(provider, runtimeCtx)
			return []JsEngineOption{func(JsEngine) error { return nil }}
		}),
	}

	options := BuildRuntimePluginOptionsWithProvider(StaticScopeProvider(baseScope), stubAuthenticator{}, plugins...)
	if len(options) != 1 {
		t.Fatalf("runtime option count = %d, want 1", len(options))
	}
	gotStub, ok := gotScope.(*stubScope)
	if !ok {
		t.Fatalf("expected resolved runtime scope to be *stubScope, got %T", gotScope)
	}
	if gotStub == baseScope {
		t.Fatal("expected provider resolution to clone the base scope")
	}
	if gotStub.Context() != runtimeCtx {
		t.Fatal("expected provider resolution to preserve the runtime ctx")
	}
}

func TestNewJsEngineFactoryReturnsNilWhenFactoryMissing(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	runtimeScope := &stubScope{
		ctx:    context.Background(),
		cfg:    &config.Config{Server: &config.ServerConfig{JsEngineFactory: "missing"}},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if got := NewJsEngineFactory(runtimeScope, stubAuthenticator{}); got != nil {
		t.Fatalf("expected nil JsEngineFactory when config points to missing factory, got %#v", got)
	}
}

func TestMergeRuntimePluginsCompatibilityKeepsMixedInputBehavior(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) RuntimePlugin {
		return NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []JsEngineOption {
			called = append(called, marker)
			return []JsEngineOption{func(JsEngine) error { return nil }}
		})
	}

	merged := MergeRuntimePlugins(
		[]RuntimePlugin{makePlugin("console", "base-console"), makePlugin("auth", "base-auth")},
		makePlugin("auth", "override-auth"),
		makePlugin("grpc", "extra-grpc"),
	)

	if len(merged) != 3 {
		t.Fatalf("merged plugin count = %d, want 3", len(merged))
	}
	BuildRuntimePluginOptions(newStubScope(), stubAuthenticator{}, merged...)
	if !reflect.DeepEqual(called, []string{"base-console", "override-auth", "extra-grpc"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestReplaceRuntimePluginsReplacesKnownBaseSlotsOnly(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) RuntimePlugin {
		return NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []JsEngineOption {
			called = append(called, marker)
			return []JsEngineOption{func(JsEngine) error { return nil }}
		})
	}

	replaced := ReplaceRuntimePlugins(
		[]RuntimePlugin{makePlugin("console", "base-console"), makePlugin("auth", "base-auth")},
		makePlugin("auth", "replacement-auth"),
		makePlugin("grpc", "ignored-grpc"),
	)

	if len(replaced) != 2 {
		t.Fatalf("replaced plugin count = %d, want 2", len(replaced))
	}
	BuildRuntimePluginOptions(newStubScope(), stubAuthenticator{}, replaced...)
	if !reflect.DeepEqual(called, []string{"base-console", "replacement-auth"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestAssembleRuntimePluginsSeparatesReplacementsFromExtras(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) RuntimePlugin {
		return NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []JsEngineOption {
			called = append(called, marker)
			return []JsEngineOption{func(JsEngine) error { return nil }}
		})
	}

	assembled := AssembleRuntimePlugins(
		[]RuntimePlugin{makePlugin("console", "base-console"), makePlugin("auth", "base-auth")},
		[]RuntimePlugin{makePlugin("auth", "replacement-auth"), makePlugin("grpc", "ignored-grpc")},
		[]RuntimePlugin{nil, makePlugin("grpc", "extra-grpc")},
	)

	if len(assembled) != 3 {
		t.Fatalf("assembled plugin count = %d, want 3", len(assembled))
	}
	BuildRuntimePluginOptions(newStubScope(), stubAuthenticator{}, assembled...)
	if !reflect.DeepEqual(called, []string{"base-console", "replacement-auth", "extra-grpc"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestRuntimePluginAssemblyApplySeparatesReplacementsFromExtras(t *testing.T) {
	called := []string{}
	makePlugin := func(name string, marker string) RuntimePlugin {
		return NewRuntimePlugin(name, func(scope.Scope, auth.Authenticator) []JsEngineOption {
			called = append(called, marker)
			return []JsEngineOption{func(JsEngine) error { return nil }}
		})
	}

	assembled := (RuntimePluginAssembly{
		Replacements: []RuntimePlugin{makePlugin("auth", "replacement-auth")},
		Extras:       []RuntimePlugin{makePlugin("grpc", "extra-grpc")},
	}).Apply([]RuntimePlugin{makePlugin("console", "base-console"), makePlugin("auth", "base-auth")})

	if len(assembled) != 3 {
		t.Fatalf("assembled plugin count = %d, want 3", len(assembled))
	}
	BuildRuntimePluginOptions(newStubScope(), stubAuthenticator{}, assembled...)
	if !reflect.DeepEqual(called, []string{"base-console", "replacement-auth", "extra-grpc"}) {
		t.Fatalf("plugin execution order = %#v", called)
	}
}

func TestBuildRuntimePluginOptionsSkipsNilPlugins(t *testing.T) {
	called := 0
	plugins := []RuntimePlugin{
		nil,
		NewRuntimePlugin("console", func(scope.Scope, auth.Authenticator) []JsEngineOption {
			called++
			return []JsEngineOption{func(JsEngine) error { return nil }}
		}),
	}

	options := BuildRuntimePluginOptions(newStubScope(), stubAuthenticator{}, plugins...)
	if called != 1 {
		t.Fatalf("plugin call count = %d, want 1", called)
	}
	if len(options) != 1 {
		t.Fatalf("runtime option count = %d, want 1", len(options))
	}
}
