// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsexecutor

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type factoryTestEngine struct{}

type factoryTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

type factoryTestBackend struct {
	mu      sync.Mutex
	scripts []*jsengine.JsScript
}

func (e *factoryTestEngine) Load(_ []*jsengine.JsScript) error {
	return nil
}

func (e *factoryTestEngine) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
}

func (e *factoryTestEngine) Close() error {
	return nil
}

func (b *factoryTestBackend) GetJsScripts() []*jsengine.JsScript {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*jsengine.JsScript, len(b.scripts))
	copy(out, b.scripts)
	return out
}

func (b *factoryTestBackend) SetJsScripts(scripts []*jsengine.JsScript) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.scripts = make([]*jsengine.JsScript, len(scripts))
	copy(b.scripts, scripts)
}

func (b *factoryTestBackend) AppendJsScripts(scripts ...*jsengine.JsScript) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.scripts = append(b.scripts, scripts...)
}

func (b *factoryTestBackend) Start() error {
	return nil
}

func (b *factoryTestBackend) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: req.Id, Result: "ok"}, nil
}

func (b *factoryTestBackend) Stop() error {
	return nil
}

func (b *factoryTestBackend) Reload(scripts ...*jsengine.JsScript) error {
	if scripts != nil {
		b.SetJsScripts(scripts)
	}
	return nil
}

func (s *factoryTestScope) Run(fn func(scope.Scope) error) error {
	return fn(s)
}

func (s *factoryTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}

func (s *factoryTestScope) Session() *scope.Session {
	return nil
}

func (s *factoryTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &factoryTestScope{ctx: ctx, cfg: s.cfg, logger: s.logger}
}

func (s *factoryTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *factoryTestScope) Logger() *slog.Logger {
	return s.logger
}

func (s *factoryTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.cfg)
}

func newFactoryTestScope(factoryName string) scope.Scope {
	return &factoryTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Server: &config.ServerConfig{
				Environment:       "default",
				JsExecutorFactory: factoryName,
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func snapshotExecutorFactories() (map[string]RuntimeFactory, map[string]CompilerFactory) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	runtimeSnapshot := make(map[string]RuntimeFactory, len(runtimeFactories))
	for name, factory := range runtimeFactories {
		runtimeSnapshot[name] = factory
	}
	compilerSnapshot := make(map[string]CompilerFactory, len(compilerFactories))
	for name, factory := range compilerFactories {
		compilerSnapshot[name] = factory
	}
	return runtimeSnapshot, compilerSnapshot
}

func restoreExecutorFactories(runtimeSnapshot map[string]RuntimeFactory, compilerSnapshot map[string]CompilerFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	runtimeFactories = runtimeSnapshot
	compilerFactories = compilerSnapshot
}

func TestCompilerExecutorLifecycle(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	RegisterRuntimeFactory(defaultRuntimeFactoryName, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})
	RegisterCompilerFactory(defaultCompilerFactoryName, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})

	exec, err := NewCompilerExecutor(
		nil,
		WithJsEngine(func() (jsengine.JsEngine, error) { return &factoryTestEngine{}, nil }),
		WithMinPoolSize(1),
		WithMaxPoolSize(1),
		WithQueueSize(2),
	)
	if err != nil {
		t.Fatalf("NewCompilerExecutor() error = %v", err)
	}

	if err := exec.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = exec.Stop() })

	resp, err := exec.Execute(context.Background(), &jsengine.JsRequest{Id: "req-1", Service: "svc"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp == nil || resp.Id != "req-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestFactoryRegistry_DefaultFactoriesPresent(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	RegisterRuntimeFactory(defaultRuntimeFactoryName, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})
	RegisterCompilerFactory(defaultCompilerFactoryName, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})

	if !RuntimeFactoryExists(defaultRuntimeFactoryName) {
		t.Fatalf("runtime default factory %q is missing", defaultRuntimeFactoryName)
	}
	if !CompilerFactoryExists(defaultCompilerFactoryName) {
		t.Fatalf("compiler default factory %q is missing", defaultCompilerFactoryName)
	}
	if !slices.Contains(RuntimeFactoryKeys(), defaultRuntimeFactoryName) {
		t.Fatalf("runtime factory keys do not include %q", defaultRuntimeFactoryName)
	}
	if !slices.Contains(CompilerFactoryKeys(), defaultCompilerFactoryName) {
		t.Fatalf("compiler factory keys do not include %q", defaultCompilerFactoryName)
	}
}

func TestFactorySelection_DefaultConfigured(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	runtimeCalls := 0
	compilerCalls := 0
	RegisterRuntimeFactory(defaultRuntimeFactoryName, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		runtimeCalls++
		return newTestExecutorWrapper(nil), nil
	})
	RegisterCompilerFactory(defaultCompilerFactoryName, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		compilerCalls++
		return newTestExecutorWrapper(nil), nil
	})

	runtimeScope := newFactoryTestScope(defaultRuntimeFactoryName)
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err != nil {
		t.Fatalf("NewRuntimeExecutor() error = %v", err)
	}
	if _, err := NewCompilerExecutor(runtimeScope); err != nil {
		t.Fatalf("NewCompilerExecutor() error = %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("runtime factory call count = %d, want 1", runtimeCalls)
	}
	if compilerCalls != 1 {
		t.Fatalf("compiler factory call count = %d, want 1", compilerCalls)
	}
}

func TestFactorySelection_CustomConfigured(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	const customFactory = "ee-custom"
	runtimeCalls := 0
	compilerCalls := 0
	RegisterRuntimeFactory(customFactory, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		runtimeCalls++
		return newTestExecutorWrapper(nil), nil
	})
	RegisterCompilerFactory(customFactory, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		compilerCalls++
		return newTestExecutorWrapper(nil), nil
	})

	runtimeScope := newFactoryTestScope(customFactory)
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err != nil {
		t.Fatalf("NewRuntimeExecutor() error = %v", err)
	}
	if _, err := NewCompilerExecutor(runtimeScope); err != nil {
		t.Fatalf("NewCompilerExecutor() error = %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("runtime factory call count = %d, want 1", runtimeCalls)
	}
	if compilerCalls != 1 {
		t.Fatalf("compiler factory call count = %d, want 1", compilerCalls)
	}
}

func TestFactorySelection_MissingConfiguredFailsFast(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	runtimeScope := newFactoryTestScope("missing-factory")
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err == nil {
		t.Fatal("expected NewRuntimeExecutor() to fail when configured factory is missing")
	} else if !strings.Contains(err.Error(), "factory-not-registered: runtime executor factory is not registered: missing-factory") {
		t.Fatalf("unexpected runtime missing-factory error: %v", err)
	}
	if _, err := NewCompilerExecutor(runtimeScope); err == nil {
		t.Fatal("expected NewCompilerExecutor() to fail when configured factory is missing")
	} else if !strings.Contains(err.Error(), "factory-not-registered: compiler executor factory is not registered: missing-factory") {
		t.Fatalf("unexpected compiler missing-factory error: %v", err)
	}
}

func TestFactorySelection_EmptyConfiguredFallsBackToDefault(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	runtimeCalls := 0
	compilerCalls := 0
	RegisterRuntimeFactory(defaultRuntimeFactoryName, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		runtimeCalls++
		return newTestExecutorWrapper(nil), nil
	})
	RegisterCompilerFactory(defaultCompilerFactoryName, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		compilerCalls++
		return newTestExecutorWrapper(nil), nil
	})

	runtimeScope := newFactoryTestScope(" ")
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err != nil {
		t.Fatalf("NewRuntimeExecutor() error = %v", err)
	}
	if _, err := NewCompilerExecutor(runtimeScope); err != nil {
		t.Fatalf("NewCompilerExecutor() error = %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("runtime default factory call count = %d, want 1", runtimeCalls)
	}
	if compilerCalls != 1 {
		t.Fatalf("compiler default factory call count = %d, want 1", compilerCalls)
	}
}

func TestFactorySelection_RuntimePathFailsWhenCompilerSideMissing(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	const halfRegistered = "half-registered"
	RegisterRuntimeFactory(halfRegistered, func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})

	runtimeScope := newFactoryTestScope(halfRegistered)
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err == nil {
		t.Fatal("expected NewRuntimeExecutor() to fail when compiler side is not registered")
	} else if !strings.Contains(err.Error(), "factory-pair-incomplete: runtime executor factory pair is incomplete: compiler executor factory is not registered: half-registered") {
		t.Fatalf("unexpected runtime pair-incomplete error: %v", err)
	}
}

func TestFactorySelection_CompilerPathFailsWhenRuntimeSideMissing(t *testing.T) {
	runtimeSnapshot, compilerSnapshot := snapshotExecutorFactories()
	t.Cleanup(func() { restoreExecutorFactories(runtimeSnapshot, compilerSnapshot) })

	const halfRegistered = "half-registered"
	RegisterCompilerFactory(halfRegistered, func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
		return newTestExecutorWrapper(&factoryTestBackend{}), nil
	})

	runtimeScope := newFactoryTestScope(halfRegistered)
	if _, err := NewCompilerExecutor(runtimeScope); err == nil {
		t.Fatal("expected NewCompilerExecutor() to fail when runtime side is not registered")
	} else if !strings.Contains(err.Error(), "factory-pair-incomplete: compiler executor factory pair is incomplete: runtime executor factory is not registered: half-registered") {
		t.Fatalf("unexpected compiler pair-incomplete error: %v", err)
	}
}
