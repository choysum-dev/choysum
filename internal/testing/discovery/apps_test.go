// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package discovery

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type testStubScope struct {
	ctx context.Context
	cfg *config.Config
}

type testPassthroughTransactor struct{ rootScope scope.Scope }

func (t testPassthroughTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	switch opts.Propagation {
	case scope.PropagationRequired:
		txScope := t.rootScope
		if ctx != nil {
			txScope = t.rootScope.WithContext(ctx)
		}
		return fn(txScope, nil)
	case scope.PropagationRequiresNew:
		return scope.ErrRequiresNewUnsupported
	case scope.PropagationNested:
		return scope.ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", scope.ErrInvalidTransactionPropagation, opts.Propagation)
	}
}

func (t testPassthroughTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (t testPassthroughTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (t testPassthroughTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (e *testStubScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *testStubScope) Transactor() scope.Transactor {
	return testPassthroughTransactor{rootScope: e}
}
func (e *testStubScope) Session() *scope.Session { return nil }
func (e *testStubScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *testStubScope) Context() context.Context { return e.ctx }
func (e *testStubScope) Logger() *slog.Logger     { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func (e *testStubScope) Config() *config.Config   { return e.cfg }
func (e *testStubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func TestResolveTestApps(t *testing.T) {
	modulesPath := t.TempDir()
	writeTestFile(t, filepath.Join(modulesPath, "auth", "service", "user.test.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "partner", "web", "user.spec.tsx"), "")
	writeTestFile(t, filepath.Join(modulesPath, "mixed", "service", "node_modules", "ignored.test.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "mixed", "web", "dist", "ignored.spec.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "empty", "service", "user.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, ".choysum", "service", "ignored.test.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "tmp", "web", "ignored.spec.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "README.md"), "ignored")

	runtimeScope := &testStubScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: modulesPath},
	}

	t.Run("requires initialized scope", func(t *testing.T) {
		_, err := ResolveTestApps(nil, "auth", true, false)
		if err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
			t.Fatalf("expected scope error, got %v", err)
		}
	})

	t.Run("requires modules path", func(t *testing.T) {
		_, err := ResolveTestApps(&testStubScope{ctx: context.Background(), cfg: &config.Config{}}, "auth", true, false)
		if err == nil || !strings.Contains(err.Error(), "config missing modules_path") {
			t.Fatalf("expected modules path error, got %v", err)
		}
	})

	t.Run("requires app argument", func(t *testing.T) {
		_, err := ResolveTestApps(runtimeScope, " ", true, false)
		if err == nil || !strings.Contains(err.Error(), "missing app") {
			t.Fatalf("expected missing app error, got %v", err)
		}
	})

	t.Run("single backend app is returned when tests exist", func(t *testing.T) {
		apps, err := ResolveTestApps(runtimeScope, "auth", true, false)
		if err != nil {
			t.Fatalf("ResolveTestApps returned error: %v", err)
		}
		if !reflect.DeepEqual(apps, []string{"auth"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("single frontend app is returned when tests exist", func(t *testing.T) {
		apps, err := ResolveTestApps(runtimeScope, "partner", false, true)
		if err != nil {
			t.Fatalf("ResolveTestApps returned error: %v", err)
		}
		if !reflect.DeepEqual(apps, []string{"partner"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("single unknown app returns error", func(t *testing.T) {
		_, err := ResolveTestApps(runtimeScope, "missing", true, true)
		if err == nil || !strings.Contains(err.Error(), `unknown app "missing"`) {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})

	t.Run("single app without matching tests returns nil slice", func(t *testing.T) {
		apps, err := ResolveTestApps(runtimeScope, "empty", true, true)
		if err != nil {
			t.Fatalf("ResolveTestApps returned error: %v", err)
		}
		if apps != nil {
			t.Fatalf("expected nil apps, got %#v", apps)
		}
	})

	t.Run("all returns runnable apps and skips internal dirs", func(t *testing.T) {
		apps, err := ResolveTestApps(runtimeScope, "all", true, true)
		if err != nil {
			t.Fatalf("ResolveTestApps returned error: %v", err)
		}
		if !reflect.DeepEqual(apps, []string{"auth", "partner"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("all propagates read dir errors", func(t *testing.T) {
		badRuntimeScope := &testStubScope{
			ctx: context.Background(),
			cfg: &config.Config{ModulesPath: filepath.Join(modulesPath, "missing")},
		}
		_, err := ResolveTestApps(badRuntimeScope, "all", true, true)
		if err == nil || !strings.Contains(err.Error(), "read modules dir") {
			t.Fatalf("expected read modules dir error, got %v", err)
		}
	})
}

func TestHasAnyBackendTests(t *testing.T) {
	modulesPath := t.TempDir()
	writeTestFile(t, filepath.Join(modulesPath, "auth", "service", "nested", "user.test.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "ignored", "service", "node_modules", "ignored.test.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "plain", "service", "user.ts"), "")

	t.Run("finds backend tests recursively", func(t *testing.T) {
		has, err := HasAnyBackendTests(modulesPath, "auth")
		if err != nil {
			t.Fatalf("HasAnyBackendTests returned error: %v", err)
		}
		if !has {
			t.Fatal("expected backend tests to be found")
		}
	})

	t.Run("skips ignored directories", func(t *testing.T) {
		has, err := HasAnyBackendTests(modulesPath, "ignored")
		if err != nil {
			t.Fatalf("HasAnyBackendTests returned error: %v", err)
		}
		if has {
			t.Fatal("expected ignored node_modules tests to be skipped")
		}
	})

	t.Run("returns false when no backend tests exist", func(t *testing.T) {
		has, err := HasAnyBackendTests(modulesPath, "plain")
		if err != nil {
			t.Fatalf("HasAnyBackendTests returned error: %v", err)
		}
		if has {
			t.Fatal("expected no backend tests")
		}
	})

	t.Run("returns false when service dir is missing", func(t *testing.T) {
		has, err := HasAnyBackendTests(modulesPath, "missing")
		if err != nil {
			t.Fatalf("HasAnyBackendTests returned error: %v", err)
		}
		if has {
			t.Fatal("expected missing service dir to return false")
		}
	})
}

func TestHasAnyFrontendTests(t *testing.T) {
	modulesPath := t.TempDir()
	writeTestFile(t, filepath.Join(modulesPath, "portal", "web", "nested", "screen.test.tsx"), "")
	writeTestFile(t, filepath.Join(modulesPath, "portal2", "web", "nested", "screen.spec.ts"), "")
	writeTestFile(t, filepath.Join(modulesPath, "ignored", "web", "dist", "screen.spec.tsx"), "")
	writeTestFile(t, filepath.Join(modulesPath, "plain", "web", "screen.tsx"), "")

	tests := []struct {
		name string
		app  string
		want bool
	}{
		{name: "finds tsx tests", app: "portal", want: true},
		{name: "finds spec tests", app: "portal2", want: true},
		{name: "skips ignored dist directory", app: "ignored", want: false},
		{name: "returns false for non-test files", app: "plain", want: false},
		{name: "returns false for missing web dir", app: "missing", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, err := HasAnyFrontendTests(modulesPath, tt.app)
			if err != nil {
				t.Fatalf("HasAnyFrontendTests returned error: %v", err)
			}
			if has != tt.want {
				t.Fatalf("HasAnyFrontendTests(%q) = %v, want %v", tt.app, has, tt.want)
			}
		})
	}
}

func TestHasAnyTestFilesAndSkips(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "tmp", "ignored.test.ts"), "")
	writeTestFile(t, filepath.Join(root, "ok", "match.test.ts"), "")

	t.Run("returns true when match is found", func(t *testing.T) {
		has, err := hasAnyTestFiles(root, func(name string) bool {
			return strings.HasSuffix(name, ".test.ts")
		})
		if err != nil {
			t.Fatalf("hasAnyTestFiles returned error: %v", err)
		}
		if !has {
			t.Fatal("expected match to be found")
		}
	})

	t.Run("propagates walk errors", func(t *testing.T) {
		_, err := hasAnyTestFiles(filepath.Join(root, "missing"), func(name string) bool { return true })
		if err == nil {
			t.Fatal("expected walk error, got nil")
		}
	})

	if !shouldSkipTestScanDir("node_modules") || !shouldSkipTestScanDir("dist") || !shouldSkipTestScanDir(".choysum") || !shouldSkipTestScanDir("tmp") {
		t.Fatal("expected known directories to be skipped")
	}
	if shouldSkipTestScanDir("service") {
		t.Fatal("did not expect service directory to be skipped")
	}
	if !shouldSkipAppName(".choysum") || !shouldSkipAppName("tmp") {
		t.Fatal("expected internal app names to be skipped")
	}
	if shouldSkipAppName("auth") {
		t.Fatal("did not expect auth app to be skipped")
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
