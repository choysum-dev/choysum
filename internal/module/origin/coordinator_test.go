// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type sourceTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *sourceTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *sourceTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *sourceTestScope) Session() *scope.Session { return &scope.Session{} }
func (e *sourceTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *sourceTestScope) Context() context.Context { return e.ctx }
func (e *sourceTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *sourceTestScope) Config() *config.Config { return e.cfg }

func (e *sourceTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type fakeRegistryProvider struct {
	peekFn  func(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error)
	fetchFn func(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error)
}

func (p *fakeRegistryProvider) PeekManifest(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
	if p.peekFn != nil {
		return p.peekFn(ctx, registryURL, moduleName, version)
	}
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, version)
	}
	return nil, nil
}

func (p *fakeRegistryProvider) Fetch(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, version)
	}
	return nil, nil
}

func writeSourceTestManifest(t *testing.T, addonsPath string, name string, mod *meta.IrModule) {
	t.Helper()
	moduleDir := filepath.Join(addonsPath, name)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	payload, err := json.Marshal(mod)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.json"), payload, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestCoordinatorResolveLocalAndRegistry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	addonsPath := filepath.Join(workspaceRoot, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{AddonsPath: addonsPath, ConfigPath: filepath.Join(workspaceRoot, "config.yaml"), DefaultChoysumPath: t.TempDir()},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))

	t.Run("resolve local module and persist local binding", func(t *testing.T) {
		writeSourceTestManifest(t, addonsPath, "auth", &meta.IrModule{Version: "1.2.3", ApplicationStr: "auth"})
		coordinator := NewCoordinator(
			runtimeScope,
			WithLockStore(lockStore),
			WithRegistryStore(registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))),
			WithRegistryProvider(&fakeRegistryProvider{}),
		)

		mod, err := coordinator.ResolveInstallModule(context.Background(), "auth")
		if err != nil {
			t.Fatalf("ResolveInstallModule(local) error = %v", err)
		}
		if mod == nil || mod.Name != "auth" || mod.Version != "v1.2.3" {
			t.Fatalf("unexpected resolved local module: %#v", mod)
		}
		binding, ok, err := lockStore.LookupBinding(WorkspaceRoot(runtimeScope), "auth")
		if err != nil {
			t.Fatalf("LookupBinding(local) error = %v", err)
		}
		if !ok || binding.OriginType != "local" || binding.OriginRef != "auth" {
			t.Fatalf("unexpected local binding: ok=%v binding=%#v", ok, binding)
		}
	})

	t.Run("resolve registry ref and persist registry binding", func(t *testing.T) {
		home := t.TempDir()
		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{URL: "https://github.com/acme/registry"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
			if registryURL != "https://github.com/acme/registry" || moduleName != "auth" || version != "v2.0.0" {
				t.Fatalf("unexpected provider input: url=%s module=%s version=%s", registryURL, moduleName, version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.0.0", Path: filepath.Join(addonsPath, "auth")}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryStore(store), WithRegistryProvider(provider))

		mod, err := coordinator.ResolveInstallModule(context.Background(), "corp/auth@v2.0.0")
		if err != nil {
			t.Fatalf("ResolveInstallModule(registry) error = %v", err)
		}
		if mod == nil || mod.Name != "auth" || mod.Version != "v2.0.0" {
			t.Fatalf("unexpected resolved registry module: %#v", mod)
		}
		binding, ok, err := lockStore.LookupBinding(WorkspaceRoot(runtimeScope), "auth")
		if err != nil {
			t.Fatalf("LookupBinding(registry) error = %v", err)
		}
		if !ok || binding.OriginType != "registry" || binding.OriginRef != "corp/auth@v2.0.0" {
			t.Fatalf("unexpected registry binding: ok=%v binding=%#v", ok, binding)
		}
	})

	t.Run("resolve local name does not fallback to registry binding", func(t *testing.T) {
		home := t.TempDir()
		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{URL: "https://github.com/acme/registry"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
			fetchCalls++
			return &meta.IrModule{Name: moduleName, Version: version}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryStore(store), WithRegistryProvider(provider))

		if err := lockStore.UpsertBinding(WorkspaceRoot(runtimeScope), Binding{
			ModuleName:      "auth",
			OriginType:      OriginTypeRegistry,
			OriginRef:       "corp/auth@v2.0.0",
			ResolvedVersion: "v2.0.0",
		}); err != nil {
			t.Fatalf("upsert lock binding: %v", err)
		}

		if err := os.RemoveAll(filepath.Join(addonsPath, "auth")); err != nil {
			t.Fatalf("remove local module dir: %v", err)
		}

		_, err = coordinator.ResolveInstallModule(context.Background(), "auth")
		if err == nil || !strings.Contains(err.Error(), "not found in addons path") {
			t.Fatalf("expected local missing error, got %v", err)
		}
		if fetchCalls != 0 {
			t.Fatalf("expected no registry fetch fallback, got %d calls", fetchCalls)
		}
	})
}

func TestCoordinatorPurgeEndToEnd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	addonsPath := filepath.Join(workspaceRoot, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{AddonsPath: addonsPath, ConfigPath: filepath.Join(workspaceRoot, "config.yaml"), DefaultChoysumPath: t.TempDir()},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
	coordinator := NewCoordinator(
		runtimeScope,
		WithLockStore(lockStore),
		WithRegistryStore(registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))),
		WithRegistryProvider(&fakeRegistryProvider{}),
	)

	writeSourceTestManifest(t, addonsPath, "auth", &meta.IrModule{Version: "1.2.3", ApplicationStr: "auth"})
	if _, err := coordinator.ResolveInstallModule(context.Background(), "auth"); err != nil {
		t.Fatalf("ResolveInstallModule(local) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(addonsPath, "auth", "manifest.json")); err != nil {
		t.Fatalf("expected local module files to exist before purge: %v", err)
	}

	if err := coordinator.Purge(context.Background(), "auth"); err != nil {
		t.Fatalf("Purge(auth) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(addonsPath, "auth")); !os.IsNotExist(err) {
		t.Fatalf("expected module dir removed after purge, stat err=%v", err)
	}
	if _, ok, err := lockStore.LookupBinding(WorkspaceRoot(runtimeScope), "auth"); err != nil {
		t.Fatalf("LookupBinding(after purge) error = %v", err)
	} else if ok {
		t.Fatal("expected binding to be removed after purge")
	}
}
