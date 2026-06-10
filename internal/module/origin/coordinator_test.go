// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	peekFn  func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
	fetchFn func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
}

func (p *fakeRegistryProvider) PeekManifest(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p.peekFn != nil {
		return p.peekFn(ctx, registryURL, moduleName, packageName, version)
	}
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, packageName, version)
	}
	return nil, nil
}

func (p *fakeRegistryProvider) Fetch(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p.fetchFn != nil {
		return p.fetchFn(ctx, registryURL, moduleName, packageName, version)
	}
	return nil, nil
}

func startCoordinatorCatalogServer(t *testing.T, npmPackage, sourceRegistry, sourceIntegrity string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.URL.Path) != "/v1/index.json" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		source := map[string]any{
			"type":     "npm",
			"registry": sourceRegistry,
			"package":  npmPackage,
		}
		if strings.TrimSpace(sourceIntegrity) != "" {
			source["integrity"] = strings.TrimSpace(sourceIntegrity)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"modules": map[string]any{
				"auth": map[string]any{
					"moduleId":      "auth",
					"latestVersion": "v2.0.0",
					"package":       npmPackage,
					"versions": map[string]any{
						"v2.0.0": map[string]any{
							"source": source,
						},
					},
				},
			},
		})
	})
	return httptest.NewServer(mux)
}

func writeSourceTestPackageJSON(t *testing.T, modulesPath string, name string, mod *meta.IrModule) {
	t.Helper()
	moduleDir := filepath.Join(modulesPath, name)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	version := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(mod.Version), "v"))
	if version == "" {
		version = "0.1.0"
	}
	application := strings.TrimSpace(mod.ApplicationStr)
	if application == "" {
		application = name
	}
	payloadObj := map[string]any{
		"name":    "@acme/choysum-" + name,
		"version": version,
		"choysum": map[string]any{
			"moduleName":  name,
			"application": application,
		},
	}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		t.Fatalf("marshal package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), payload, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func TestCoordinatorResolveLocalAndRegistry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: modulesPath, ConfigPath: filepath.Join(workspaceRoot, "config.yaml"), DefaultChoysumPath: t.TempDir()},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))

	t.Run("resolve local module and persist local binding", func(t *testing.T) {
		writeSourceTestPackageJSON(t, modulesPath, "auth", &meta.IrModule{Version: "1.2.3", ApplicationStr: "auth"})
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
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "sha512-catalog-auth-v2")
		defer catalog.Close()
		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{IndexURL: catalog.URL + "/v1/index.json"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			if registryURL != "https://registry.npmjs.org" || moduleName != "auth" || packageName != "@acme/choysum-auth" || version != "v2.0.0" {
				t.Fatalf("unexpected provider input: url=%s module=%s package=%s version=%s", registryURL, moduleName, packageName, version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.0.0", Integrity: "sha512-provider-auth-v2", Path: filepath.Join(modulesPath, "auth")}, nil
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
		if binding.ResolvedVersion != "v2.0.0" || binding.Integrity != "sha512-provider-auth-v2" {
			t.Fatalf("unexpected registry lock details: %#v", binding)
		}
	})

	t.Run("resolve registry latest pins origin ref to resolved version", func(t *testing.T) {
		home := t.TempDir()
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "")
		defer catalog.Close()
		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{IndexURL: catalog.URL + "/v1/index.json"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			if version != "latest" {
				t.Fatalf("expected provider to receive latest, got %s", version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.1.0", Integrity: "sha512-provider-auth-v210", Path: filepath.Join(modulesPath, "auth")}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryStore(store), WithRegistryProvider(provider))

		mod, err := coordinator.ResolveInstallModule(context.Background(), "corp/auth")
		if err != nil {
			t.Fatalf("ResolveInstallModule(registry latest) error = %v", err)
		}
		if mod == nil || mod.Version != "v2.1.0" {
			t.Fatalf("unexpected resolved registry module: %#v", mod)
		}
		binding, ok, err := lockStore.LookupBinding(WorkspaceRoot(runtimeScope), "auth")
		if err != nil {
			t.Fatalf("LookupBinding(registry latest) error = %v", err)
		}
		if !ok {
			t.Fatalf("expected registry binding to exist after latest install")
		}
		if binding.OriginRef != "corp/auth@v2.1.0" || binding.ResolvedVersion != "v2.1.0" {
			t.Fatalf("expected latest ref to pin resolved version, got %#v", binding)
		}
		if binding.Integrity != "sha512-provider-auth-v210" {
			t.Fatalf("unexpected integrity for latest install: %#v", binding)
		}
	})

	t.Run("resolve local name does not fallback to registry binding", func(t *testing.T) {
		home := t.TempDir()
		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{IndexURL: "https://index.acme.dev/v1/index.json"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
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

		if err := os.RemoveAll(filepath.Join(modulesPath, "auth")); err != nil {
			t.Fatalf("remove local module dir: %v", err)
		}

		_, err = coordinator.ResolveInstallModule(context.Background(), "auth")
		if err == nil || !strings.Contains(err.Error(), "not found in modules path") {
			t.Fatalf("expected local missing error, got %v", err)
		}
		if fetchCalls != 0 {
			t.Fatalf("expected no registry fetch fallback, got %d calls", fetchCalls)
		}
	})

	t.Run("resolve registry ref rejects empty catalog npm package", func(t *testing.T) {
		home := t.TempDir()
		catalog := startCoordinatorCatalogServer(t, "", "https://registry.npmjs.org", "")
		defer catalog.Close()

		store := registry.NewStore(registry.WithHomeDir(home), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{IndexURL: catalog.URL + "/v1/index.json"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			fetchCalls++
			return &meta.IrModule{Name: moduleName, Version: version}, nil
		}}

		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryStore(store), WithRegistryProvider(provider))
		_, err = coordinator.ResolveInstallModule(context.Background(), "corp/auth@v2.0.0")
		if err == nil || !strings.Contains(err.Error(), "empty npm package source") {
			t.Fatalf("expected empty npm package source error, got %v", err)
		}
		if fetchCalls != 0 {
			t.Fatalf("expected provider fetch not called, got %d calls", fetchCalls)
		}
	})
}

func TestCoordinatorPurgeEndToEnd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: modulesPath, ConfigPath: filepath.Join(workspaceRoot, "config.yaml"), DefaultChoysumPath: t.TempDir()},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
	coordinator := NewCoordinator(
		runtimeScope,
		WithLockStore(lockStore),
		WithRegistryStore(registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))),
		WithRegistryProvider(&fakeRegistryProvider{}),
	)

	writeSourceTestPackageJSON(t, modulesPath, "auth", &meta.IrModule{Version: "1.2.3", ApplicationStr: "auth"})
	if _, err := coordinator.ResolveInstallModule(context.Background(), "auth"); err != nil {
		t.Fatalf("ResolveInstallModule(local) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "auth", "package.json")); err != nil {
		t.Fatalf("expected local module files to exist before purge: %v", err)
	}

	if err := coordinator.Purge(context.Background(), "auth"); err != nil {
		t.Fatalf("Purge(auth) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "auth")); !os.IsNotExist(err) {
		t.Fatalf("expected module dir removed after purge, stat err=%v", err)
	}
	if _, ok, err := lockStore.LookupBinding(WorkspaceRoot(runtimeScope), "auth"); err != nil {
		t.Fatalf("LookupBinding(after purge) error = %v", err)
	} else if ok {
		t.Fatal("expected binding to be removed after purge")
	}
}

func TestCoordinatorHelperBranchFunctions(t *testing.T) {
	t.Parallel()

	if err := validateAndNormalizeModuleSemVer(&meta.IrModule{Name: "auth", Version: "   "}, "source.json"); err == nil || !strings.Contains(err.Error(), "empty module version") {
		t.Fatalf("expected empty module version error, got %v", err)
	}
	if err := validateAndNormalizeModuleSemVer(&meta.IrModule{Name: "auth", Version: "not-semver"}, "source.json"); err == nil || !strings.Contains(err.Error(), "invalid module version") {
		t.Fatalf("expected invalid module version error, got %v", err)
	}

	mod := &meta.IrModule{Name: "auth", Version: "1.2.3"}
	if err := validateAndNormalizeModuleSemVer(mod, "source.json"); err != nil {
		t.Fatalf("validateAndNormalizeModuleSemVer(valid) error = %v", err)
	}
	if mod.Version != "v1.2.3" {
		t.Fatalf("normalized version = %q, want %q", mod.Version, "v1.2.3")
	}

	if got := canonicalRegistryOriginRef(ParsedInput{Kind: InputKindLocal, LocalName: " auth "}, "v1.0.0"); got != "auth" {
		t.Fatalf("canonicalRegistryOriginRef(local) = %q, want %q", got, "auth")
	}
	if got := resolveBindingIntegrity("  sha512-catalog  ", &meta.IrModule{Integrity: ""}); got != "sha512-catalog" {
		t.Fatalf("resolveBindingIntegrity(catalog fallback) = %q, want %q", got, "sha512-catalog")
	}
}

func TestCoordinatorPeekLocalModuleErrorBranches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	t.Run("empty module name", func(t *testing.T) {
		runtimeScope := &sourceTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DefaultChoysumPath: t.TempDir()}}
		coordinator := NewCoordinator(runtimeScope)

		if _, err := coordinator.peekLocalModule("   "); err == nil || !strings.Contains(err.Error(), "module name is empty") {
			t.Fatalf("expected empty module name error, got %v", err)
		}
	})

	t.Run("stat package json failure", func(t *testing.T) {
		modulesPathFile := filepath.Join(t.TempDir(), "modules-file")
		if err := os.WriteFile(modulesPathFile, []byte("not-a-dir"), 0o644); err != nil {
			t.Fatalf("write modules path file: %v", err)
		}
		runtimeScope := &sourceTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPathFile, DefaultChoysumPath: t.TempDir()}}
		coordinator := NewCoordinator(runtimeScope)

		if _, err := coordinator.peekLocalModule("auth"); err == nil || !strings.Contains(err.Error(), "stat local module package.json failed") {
			t.Fatalf("expected stat failure, got %v", err)
		}
	})

	t.Run("read package json failure", func(t *testing.T) {
		modulesPath := t.TempDir()
		packageJSONDir := filepath.Join(modulesPath, "auth", "package.json")
		if err := os.MkdirAll(packageJSONDir, 0o755); err != nil {
			t.Fatalf("mkdir package.json dir: %v", err)
		}
		runtimeScope := &sourceTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath, DefaultChoysumPath: t.TempDir()}}
		coordinator := NewCoordinator(runtimeScope)

		if _, err := coordinator.peekLocalModule("auth"); err == nil || !strings.Contains(err.Error(), "read package.json") {
			t.Fatalf("expected read package.json error, got %v", err)
		}
	})

	t.Run("parse package json failure", func(t *testing.T) {
		modulesPath := t.TempDir()
		moduleDir := filepath.Join(modulesPath, "auth")
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			t.Fatalf("mkdir module dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(`{"name":`), 0o644); err != nil {
			t.Fatalf("write invalid package.json: %v", err)
		}
		runtimeScope := &sourceTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: modulesPath, DefaultChoysumPath: t.TempDir()}}
		coordinator := NewCoordinator(runtimeScope)

		if _, err := coordinator.peekLocalModule("auth"); err == nil || !strings.Contains(err.Error(), "parse package.json") {
			t.Fatalf("expected parse package.json error, got %v", err)
		}
	})
}

func TestCoordinatorResolveRegistrySourceErrorBranches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath:        t.TempDir(),
			ConfigPath:         filepath.Join(t.TempDir(), "config.yaml"),
			DefaultChoysumPath: t.TempDir(),
		},
	}

	t.Run("registry alias resolve error", func(t *testing.T) {
		store := registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		coordinator := NewCoordinator(runtimeScope, WithRegistryStore(store), WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		if _, err := coordinator.resolveRegistrySource(context.Background(), ParsedInput{Kind: InputKindRegistry, RegistryAlias: "missing", ModuleName: "auth", Version: "latest"}); err == nil {
			t.Fatal("expected resolveRegistrySource to fail for missing alias")
		}
	})

	t.Run("catalog info error is wrapped", func(t *testing.T) {
		catalogServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		}))
		defer catalogServer.Close()

		store := registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("registry store load: %v", err)
		}
		cfg.Registries["corp"] = registry.Entry{IndexURL: catalogServer.URL + "/v1/index.json"}
		if err := store.Save(cfg); err != nil {
			t.Fatalf("registry store save: %v", err)
		}

		coordinator := NewCoordinator(runtimeScope, WithRegistryStore(store), WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		_, err = coordinator.resolveRegistrySource(context.Background(), ParsedInput{Kind: InputKindRegistry, RegistryAlias: "corp", ModuleName: "auth", Version: "latest"})
		if err == nil || !strings.Contains(err.Error(), "resolve catalog source failed") {
			t.Fatalf("expected wrapped catalog source error, got %v", err)
		}
	})
}

func TestCoordinatorPeekBranches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}

	runtimeScope := &sourceTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath:        modulesPath,
			ConfigPath:         filepath.Join(workspaceRoot, "config.yaml"),
			DefaultChoysumPath: t.TempDir(),
		},
	}
	catalogServer := startCoordinatorCatalogServer(t, "auth", "https://registry.npmjs.org", "")
	defer catalogServer.Close()

	store := registry.NewStore(registry.WithHomeDir(t.TempDir()), registry.WithDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("registry store load: %v", err)
	}
	cfg.Registries["corp"] = registry.Entry{IndexURL: catalogServer.URL + "/v1/index.json"}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("registry store save: %v", err)
	}

	t.Run("registry input uses provider PeekManifest", func(t *testing.T) {
		peekCalls := 0
		provider := &fakeRegistryProvider{peekFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			peekCalls++
			if registryURL != "https://registry.npmjs.org" || moduleName != "auth" || packageName != "auth" || version != "v2.0.0" {
				t.Fatalf("unexpected provider peek args: registry=%s module=%s package=%s version=%s", registryURL, moduleName, packageName, version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.0.0"}, nil
		}}

		coordinator := NewCoordinator(runtimeScope, WithRegistryStore(store), WithRegistryProvider(provider), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		mod, err := coordinator.Peek(context.Background(), "corp/auth@v2.0.0")
		if err != nil {
			t.Fatalf("Peek(registry) error = %v", err)
		}
		if mod == nil || mod.Name != "auth" || mod.Version != "v2.0.0" {
			t.Fatalf("unexpected Peek(registry) module: %#v", mod)
		}
		if peekCalls != 1 {
			t.Fatalf("provider peek calls = %d, want 1", peekCalls)
		}
	})

	t.Run("registry peek requires provider", func(t *testing.T) {
		parsed, err := ParseInput("corp/auth@v2.0.0")
		if err != nil {
			t.Fatalf("ParseInput() error = %v", err)
		}
		coordinator := &Coordinator{runtimeScope: runtimeScope, registryStore: store}
		if _, err := coordinator.peekRegistryModule(context.Background(), parsed); err == nil || !strings.Contains(err.Error(), "registry provider is nil") {
			t.Fatalf("expected registry provider nil error, got %v", err)
		}
	})

	t.Run("local module missing", func(t *testing.T) {
		coordinator := NewCoordinator(runtimeScope, WithRegistryStore(store), WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		if _, err := coordinator.Peek(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), "not found in modules path") {
			t.Fatalf("expected local missing error, got %v", err)
		}
	})

	t.Run("local module success", func(t *testing.T) {
		writeSourceTestPackageJSON(t, modulesPath, "auth", &meta.IrModule{Version: "2.1.0", ApplicationStr: "auth"})
		coordinator := NewCoordinator(runtimeScope, WithRegistryStore(store), WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		mod, err := coordinator.Peek(context.Background(), "auth")
		if err != nil {
			t.Fatalf("Peek(local) error = %v", err)
		}
		if mod == nil || mod.Name != "auth" || mod.Version != "v2.1.0" {
			t.Fatalf("unexpected Peek(local) module: %#v", mod)
		}
	})

	t.Run("nil coordinator env", func(t *testing.T) {
		var nilCoordinator *Coordinator
		if _, err := nilCoordinator.Peek(context.Background(), "auth"); err == nil || !strings.Contains(err.Error(), "origin coordinator env is nil") {
			t.Fatalf("expected nil coordinator env error, got %v", err)
		}
	})
}
