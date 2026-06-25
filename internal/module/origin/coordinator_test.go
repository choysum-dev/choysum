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
	"sync/atomic"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/datatypes"
)

type sourceTestScope struct {
	ctx context.Context
	cfg *config.Config
}

type sourceTestScopeWithFactoryInput struct {
	*sourceTestScope
	input scope.FactoryInput
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

func (e *sourceTestScopeWithFactoryInput) FactoryInput() scope.FactoryInput {
	if e == nil {
		return nil
	}
	if e.input != nil {
		return e.input
	}
	if e.sourceTestScope == nil {
		return nil
	}
	return e.sourceTestScope.FactoryInput()
}

type fallbackToggleFactoryInput struct {
	cfg     *config.Config
	enabled bool
}

func (i fallbackToggleFactoryInput) Environment() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.Environment
}

func (i fallbackToggleFactoryInput) ModulesPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ModulesPath
}

func (i fallbackToggleFactoryInput) DistPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DistPath
}

func (i fallbackToggleFactoryInput) TmpPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.TmpPath
}

func (i fallbackToggleFactoryInput) DefaultChoysumPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DefaultChoysumPath
}

func (i fallbackToggleFactoryInput) ConfigPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ConfigPath
}

func (i fallbackToggleFactoryInput) ModuleCatalogIndexURL() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ModuleCatalogIndexURL
}

func (i fallbackToggleFactoryInput) ModuleInstallRegistryFallbackEnabled() bool {
	return i.enabled
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
		cfg: &config.Config{
			ModulesPath:           modulesPath,
			ConfigPath:            filepath.Join(workspaceRoot, "config.yaml"),
			DefaultChoysumPath:    t.TempDir(),
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
		},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))

	t.Run("resolve local module and persist local binding", func(t *testing.T) {
		writeSourceTestPackageJSON(t, modulesPath, "auth", &meta.IrModule{Version: "1.2.3", ApplicationStr: "auth"})
		coordinator := NewCoordinator(
			runtimeScope,
			WithLockStore(lockStore),
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
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "sha512-catalog-auth-v2")
		defer catalog.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalog.URL + "/v1/index.json"

		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			if registryURL != "https://registry.npmjs.org" || moduleName != "auth" || packageName != "@acme/choysum-auth" || version != "v2.0.0" {
				t.Fatalf("unexpected provider input: url=%s module=%s package=%s version=%s", registryURL, moduleName, packageName, version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.0.0", Integrity: "sha512-provider-auth-v2", Path: filepath.Join(modulesPath, "auth")}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryProvider(provider))

		mod, err := coordinator.ResolveInstallModule(context.Background(), "auth@v2.0.0")
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
		if !ok || binding.OriginType != "registry" || binding.OriginRef != "auth@v2.0.0" {
			t.Fatalf("unexpected registry binding: ok=%v binding=%#v", ok, binding)
		}
		if binding.ResolvedVersion != "v2.0.0" || binding.Integrity != "sha512-provider-auth-v2" {
			t.Fatalf("unexpected registry lock details: %#v", binding)
		}
	})

	t.Run("resolve registry latest pins origin ref to resolved version", func(t *testing.T) {
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "")
		defer catalog.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalog.URL + "/v1/index.json"

		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			if version != "latest" {
				t.Fatalf("expected provider to receive latest, got %s", version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.1.0", Integrity: "sha512-provider-auth-v210", Path: filepath.Join(modulesPath, "auth")}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryProvider(provider))

		mod, err := coordinator.ResolveInstallModule(context.Background(), "auth@latest")
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
		if binding.OriginRef != "auth@v2.1.0" || binding.ResolvedVersion != "v2.1.0" {
			t.Fatalf("expected latest ref to pin resolved version, got %#v", binding)
		}
		if binding.Integrity != "sha512-provider-auth-v210" {
			t.Fatalf("unexpected integrity for latest install: %#v", binding)
		}
	})

	t.Run("resolve local name falls back to registry when local is missing", func(t *testing.T) {
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "sha512-catalog-auth-v3")
		defer catalog.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalog.URL + "/v1/index.json"

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			fetchCalls++
			return &meta.IrModule{Name: moduleName, Version: "v3.0.0", Path: filepath.Join(modulesPath, moduleName)}, nil
		}}
		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryProvider(provider))

		if err := os.RemoveAll(filepath.Join(modulesPath, "auth")); err != nil {
			t.Fatalf("remove local module dir: %v", err)
		}

		mod, err := coordinator.ResolveInstallModule(context.Background(), "auth")
		if err != nil {
			t.Fatalf("expected registry fallback to succeed when local is missing, got %v", err)
		}
		if mod == nil || mod.Version != "v3.0.0" {
			t.Fatalf("unexpected fallback module: %#v", mod)
		}
		if fetchCalls != 1 {
			t.Fatalf("expected 1 registry fetch call during fallback, got %d", fetchCalls)
		}
	})

	t.Run("resolve local name does not fall back when registry fallback is disabled", func(t *testing.T) {
		catalog := startCoordinatorCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", "sha512-catalog-auth-v3")
		defer catalog.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalog.URL + "/v1/index.json"

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			fetchCalls++
			return &meta.IrModule{Name: moduleName, Version: "v3.0.0", Path: filepath.Join(modulesPath, moduleName)}, nil
		}}

		toggleInput := fallbackToggleFactoryInput{
			cfg:     runtimeScope.cfg,
			enabled: false,
		}
		runtimeScopeNoFallback := &sourceTestScopeWithFactoryInput{sourceTestScope: runtimeScope, input: toggleInput}
		coordinator := NewCoordinator(runtimeScopeNoFallback, WithLockStore(lockStore), WithRegistryProvider(provider))

		if err := os.RemoveAll(filepath.Join(modulesPath, "auth")); err != nil {
			t.Fatalf("remove local module dir: %v", err)
		}

		_, err := coordinator.ResolveInstallModule(context.Background(), "auth")
		if err == nil || !strings.Contains(err.Error(), "not found in modules path") {
			t.Fatalf("expected local missing error when fallback disabled, got %v", err)
		}
		if fetchCalls != 0 {
			t.Fatalf("expected 0 registry fetch calls when fallback is disabled, got %d", fetchCalls)
		}
	})

	t.Run("resolve registry ref rejects empty catalog npm package", func(t *testing.T) {
		catalog := startCoordinatorCatalogServer(t, "", "https://registry.npmjs.org", "")
		defer catalog.Close()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalog.URL + "/v1/index.json"

		fetchCalls := 0
		provider := &fakeRegistryProvider{fetchFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			fetchCalls++
			return &meta.IrModule{Name: moduleName, Version: version}, nil
		}}

		coordinator := NewCoordinator(runtimeScope, WithLockStore(lockStore), WithRegistryProvider(provider))
		_, err := coordinator.ResolveInstallModule(context.Background(), "auth@v2.0.0")
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
		cfg: &config.Config{
			ModulesPath:           modulesPath,
			ConfigPath:            filepath.Join(workspaceRoot, "config.yaml"),
			DefaultChoysumPath:    t.TempDir(),
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
		},
	}
	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))
	coordinator := NewCoordinator(
		runtimeScope,
		WithLockStore(lockStore),
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
	if got := canonicalRegistryOriginRef(ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "LATEST"}, ""); got != "auth@latest" {
		t.Fatalf("canonicalRegistryOriginRef(registry latest fallback) = %q, want %q", got, "auth@latest")
	}
	if got := canonicalRegistryOriginRef(ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "LaTeSt"}, "v2.0.1"); got != "auth@v2.0.1" {
		t.Fatalf("canonicalRegistryOriginRef(registry latest resolved) = %q, want %q", got, "auth@v2.0.1")
	}
	if got := canonicalRegistryOriginRef(ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "   "}, ""); got != "auth@latest" {
		t.Fatalf("canonicalRegistryOriginRef(registry blank version) = %q, want %q", got, "auth@latest")
	}
	if got := canonicalRegistryOriginRef(ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "latest"}, "  v2.0.2  "); got != "auth@v2.0.2" {
		t.Fatalf("canonicalRegistryOriginRef(registry latest resolved trimmed) = %q, want %q", got, "auth@v2.0.2")
	}
	if got := resolveBindingIntegrity("  sha512-catalog  ", &meta.IrModule{Integrity: ""}); got != "sha512-catalog" {
		t.Fatalf("resolveBindingIntegrity(catalog fallback) = %q, want %q", got, "sha512-catalog")
	}
}

func TestCoordinatorHelperAndGuardCoverage(t *testing.T) {
	t.Parallel()

	if err := applyEntryPoints(nil); err != nil {
		t.Fatalf("applyEntryPoints(nil) error = %v", err)
	}

	invalidEntryPoints := &meta.IrModule{EntryPoints: datatypes.JSON([]byte(`{"web":`))}
	if err := applyEntryPoints(invalidEntryPoints); err == nil || !strings.Contains(err.Error(), "error unmarshalling entry points") {
		t.Fatalf("expected entry points unmarshal error, got %v", err)
	}

	modWithEntryPoints := &meta.IrModule{EntryPoints: datatypes.JSON([]byte(`{"web":"./web/index.ts","service":"./service/main.ts"}`))}
	if err := applyEntryPoints(modWithEntryPoints); err != nil {
		t.Fatalf("applyEntryPoints(valid) error = %v", err)
	}
	if modWithEntryPoints.WebEntryPoint != "./web/index.ts" || modWithEntryPoints.ServiceEntryPoint != "./service/main.ts" {
		t.Fatalf("unexpected entry point mapping: %#v", modWithEntryPoints)
	}

	if err := validateAndNormalizeModuleSemVer(nil, ""); err != nil {
		t.Fatalf("validateAndNormalizeModuleSemVer(nil) error = %v", err)
	}

	if got := resolveBindingIntegrity("  sha512-catalog  ", nil); got != "sha512-catalog" {
		t.Fatalf("resolveBindingIntegrity(nil module) = %q, want %q", got, "sha512-catalog")
	}
	if got := resolveBindingIntegrity("sha512-catalog", &meta.IrModule{Integrity: "  sha512-provider  "}); got != "sha512-provider" {
		t.Fatalf("resolveBindingIntegrity(provider integrity) = %q, want %q", got, "sha512-provider")
	}

	if got := registrySourceResolutionCacheKey("", "auth"); got != "" {
		t.Fatalf("registrySourceResolutionCacheKey(empty index) = %q, want empty", got)
	}
	if got := registrySourceResolutionCacheKey("https://index.acme.dev/v1/index.json", "   "); got != "" {
		t.Fatalf("registrySourceResolutionCacheKey(empty module) = %q, want empty", got)
	}
	if got := registrySourceResolutionCacheKey(" https://index.acme.dev/v1/index.json ", " auth "); got != "https://index.acme.dev/v1/index.json|auth" {
		t.Fatalf("registrySourceResolutionCacheKey(trimmed) = %q, want %q", got, "https://index.acme.dev/v1/index.json|auth")
	}

	var nilCoordinator *Coordinator
	if _, ok := nilCoordinator.lookupRegistrySourceResolution("index|auth"); ok {
		t.Fatal("expected nil coordinator cache lookup miss")
	}
	nilCoordinator.cacheRegistrySourceResolution("index|auth", registrySourceResolution{packageName: "@acme/choysum-auth"})

	coordinator := &Coordinator{}
	if _, ok := coordinator.lookupRegistrySourceResolution(""); ok {
		t.Fatal("expected empty cache key lookup miss")
	}
	coordinator.cacheRegistrySourceResolution("", registrySourceResolution{packageName: "@acme/choysum-auth"})
	if coordinator.resolutionCache != nil {
		t.Fatal("expected empty cache key write to be ignored")
	}

	cacheKey := "https://index.acme.dev/v1/index.json|auth"
	wantResolved := registrySourceResolution{
		registryURL: "https://registry.npmjs.org",
		packageName: "@acme/choysum-auth",
		integrity:   "sha512-auth",
	}
	coordinator.cacheRegistrySourceResolution(cacheKey, wantResolved)
	gotResolved, ok := coordinator.lookupRegistrySourceResolution(cacheKey)
	if !ok {
		t.Fatal("expected cache hit after write")
	}
	if gotResolved != wantResolved {
		t.Fatalf("cached resolution = %#v, want %#v", gotResolved, wantResolved)
	}

	if _, err := nilCoordinator.Fetch(context.Background(), "auth"); err == nil || !strings.Contains(err.Error(), "origin coordinator env is nil") {
		t.Fatalf("expected nil coordinator fetch error, got %v", err)
	}
	if err := nilCoordinator.Purge(context.Background(), "auth"); err == nil || !strings.Contains(err.Error(), "origin coordinator env is nil") {
		t.Fatalf("expected nil coordinator purge error, got %v", err)
	}

	runtimeScope := &sourceTestScope{ctx: context.Background(), cfg: &config.Config{ModulesPath: t.TempDir(), DefaultChoysumPath: t.TempDir()}}
	coordinatorWithScope := NewCoordinator(runtimeScope)
	if _, err := coordinatorWithScope.Fetch(context.Background(), "   "); err == nil || !strings.Contains(err.Error(), "empty module input") {
		t.Fatalf("expected fetch parse error for empty input, got %v", err)
	}
	if err := coordinatorWithScope.Purge(context.Background(), "auth@latest"); err == nil || !strings.Contains(err.Error(), "purge accepts local module name only") {
		t.Fatalf("expected purge local-name guard error, got %v", err)
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
			ModulesPath:           t.TempDir(),
			ConfigPath:            filepath.Join(t.TempDir(), "config.yaml"),
			DefaultChoysumPath:    t.TempDir(),
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
		},
	}

	t.Run("catalog info error is wrapped", func(t *testing.T) {
		catalogServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		}))
		defer catalogServer.Close()

		runtimeScope.cfg.ModuleCatalogIndexURL = catalogServer.URL + "/v1/index.json"

		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		_, err := coordinator.resolveRegistrySource(context.Background(), ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "latest"})
		if err == nil || !strings.Contains(err.Error(), "resolve catalog source failed") {
			t.Fatalf("expected wrapped catalog source error, got %v", err)
		}
	})
}

func TestCoordinatorResolveRegistrySourceCaching(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	newRuntimeScope := func() *sourceTestScope {
		return &sourceTestScope{
			ctx: context.Background(),
			cfg: &config.Config{
				ModulesPath:           t.TempDir(),
				ConfigPath:            filepath.Join(t.TempDir(), "config.yaml"),
				DefaultChoysumPath:    t.TempDir(),
				ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
			},
		}
	}

	startCacheCatalogServer := func(t *testing.T, npmPackage, sourceRegistry string, hits *int32) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(r.URL.Path) != "/v1/index.json" {
				http.NotFound(w, r)
				return
			}
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			atomic.AddInt32(hits, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modules": map[string]any{
					"auth": map[string]any{
						"moduleId":      "auth",
						"latestVersion": "v2.0.0",
						"package":       npmPackage,
						"versions": map[string]any{
							"v2.0.0": map[string]any{
								"source": map[string]any{
									"type":     "npm",
									"registry": sourceRegistry,
									"package":  npmPackage,
								},
							},
						},
					},
				},
			})
		}))
	}

	parsed := ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "latest"}

	t.Run("same module under same index uses cache", func(t *testing.T) {
		var hits int32
		catalogServer := startCacheCatalogServer(t, "@acme/choysum-auth", "https://registry.npmjs.org", &hits)
		defer catalogServer.Close()

		runtimeScope := newRuntimeScope()
		runtimeScope.cfg.ModuleCatalogIndexURL = catalogServer.URL + "/v1/index.json"
		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}))

		for i := 0; i < 3; i++ {
			resolved, err := coordinator.resolveRegistrySource(context.Background(), parsed)
			if err != nil {
				t.Fatalf("resolveRegistrySource() error = %v", err)
			}
			if resolved.packageName != "@acme/choysum-auth" {
				t.Fatalf("packageName = %q, want %q", resolved.packageName, "@acme/choysum-auth")
			}
			if resolved.registryURL != "https://registry.npmjs.org" {
				t.Fatalf("registryURL = %q, want %q", resolved.registryURL, "https://registry.npmjs.org")
			}
		}

		if got := atomic.LoadInt32(&hits); got != 1 {
			t.Fatalf("catalog server hit count = %d, want 1", got)
		}
	})

	t.Run("cache key includes index url", func(t *testing.T) {
		var hitsA int32
		catalogA := startCacheCatalogServer(t, "@acme/choysum-auth-a", "https://registry-a.example.dev", &hitsA)
		defer catalogA.Close()

		var hitsB int32
		catalogB := startCacheCatalogServer(t, "@acme/choysum-auth-b", "https://registry-b.example.dev", &hitsB)
		defer catalogB.Close()

		runtimeScope := newRuntimeScope()
		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}))

		runtimeScope.cfg.ModuleCatalogIndexURL = catalogA.URL + "/v1/index.json"
		resolvedA, err := coordinator.resolveRegistrySource(context.Background(), parsed)
		if err != nil {
			t.Fatalf("resolveRegistrySource(indexA) error = %v", err)
		}
		if resolvedA.packageName != "@acme/choysum-auth-a" || resolvedA.registryURL != "https://registry-a.example.dev" {
			t.Fatalf("unexpected resolution from indexA: %#v", resolvedA)
		}

		runtimeScope.cfg.ModuleCatalogIndexURL = catalogB.URL + "/v1/index.json"
		resolvedB, err := coordinator.resolveRegistrySource(context.Background(), parsed)
		if err != nil {
			t.Fatalf("resolveRegistrySource(indexB) error = %v", err)
		}
		if resolvedB.packageName != "@acme/choysum-auth-b" || resolvedB.registryURL != "https://registry-b.example.dev" {
			t.Fatalf("unexpected resolution from indexB: %#v", resolvedB)
		}

		if got := atomic.LoadInt32(&hitsA); got != 1 {
			t.Fatalf("catalogA hit count = %d, want 1", got)
		}
		if got := atomic.LoadInt32(&hitsB); got != 1 {
			t.Fatalf("catalogB hit count = %d, want 1", got)
		}
	})

	t.Run("blank index url falls back to default and can hit cache", func(t *testing.T) {
		runtimeScope := newRuntimeScope()
		runtimeScope.cfg.ModuleCatalogIndexURL = "   "
		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}))

		cacheKey := registrySourceResolutionCacheKey(config.DefaultModuleCatalogIndexURL, "auth")
		wantResolved := registrySourceResolution{
			registryURL: "https://registry.npmjs.org",
			packageName: "@acme/choysum-auth",
			integrity:   "sha512-auth",
		}
		coordinator.cacheRegistrySourceResolution(cacheKey, wantResolved)

		resolved, err := coordinator.resolveRegistrySource(context.Background(), parsed)
		if err != nil {
			t.Fatalf("resolveRegistrySource(blank index fallback cache hit) error = %v", err)
		}
		if resolved != wantResolved {
			t.Fatalf("resolveRegistrySource(blank index fallback cache hit) = %#v, want %#v", resolved, wantResolved)
		}
	})

	t.Run("nil runtime scope falls back to default index and can hit cache", func(t *testing.T) {
		coordinator := NewCoordinator(nil, WithRegistryProvider(&fakeRegistryProvider{}))

		cacheKey := registrySourceResolutionCacheKey(config.DefaultModuleCatalogIndexURL, "auth")
		wantResolved := registrySourceResolution{
			registryURL: "https://registry.npmjs.org",
			packageName: "@acme/choysum-auth",
			integrity:   "sha512-auth",
		}
		coordinator.cacheRegistrySourceResolution(cacheKey, wantResolved)

		resolved, err := coordinator.resolveRegistrySource(context.Background(), parsed)
		if err != nil {
			t.Fatalf("resolveRegistrySource(nil scope default index fallback cache hit) error = %v", err)
		}
		if resolved != wantResolved {
			t.Fatalf("resolveRegistrySource(nil scope default index fallback cache hit) = %#v, want %#v", resolved, wantResolved)
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
			ModulesPath:           modulesPath,
			ConfigPath:            filepath.Join(workspaceRoot, "config.yaml"),
			DefaultChoysumPath:    t.TempDir(),
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
		},
	}
	catalogServer := startCoordinatorCatalogServer(t, "auth", "https://registry.npmjs.org", "")
	defer catalogServer.Close()
	runtimeScope.cfg.ModuleCatalogIndexURL = catalogServer.URL + "/v1/index.json"

	t.Run("registry input uses provider PeekManifest", func(t *testing.T) {
		peekCalls := 0
		provider := &fakeRegistryProvider{peekFn: func(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
			peekCalls++
			if registryURL != "https://registry.npmjs.org" || moduleName != "auth" || packageName != "auth" || version != "v2.0.0" {
				t.Fatalf("unexpected provider peek args: registry=%s module=%s package=%s version=%s", registryURL, moduleName, packageName, version)
			}
			return &meta.IrModule{Name: "auth", Version: "v2.0.0"}, nil
		}}

		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(provider), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		mod, err := coordinator.Peek(context.Background(), "auth@v2.0.0")
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
		parsed, err := ParseInput("auth@v2.0.0")
		if err != nil {
			t.Fatalf("ParseInput() error = %v", err)
		}
		coordinator := &Coordinator{runtimeScope: runtimeScope}
		if _, err := coordinator.peekRegistryModule(context.Background(), parsed); err == nil || !strings.Contains(err.Error(), "registry provider is nil") {
			t.Fatalf("expected registry provider nil error, got %v", err)
		}
	})

	t.Run("local module missing", func(t *testing.T) {
		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
		if _, err := coordinator.Peek(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), "not found in modules path") {
			t.Fatalf("expected local missing error, got %v", err)
		}
	})

	t.Run("local module success", func(t *testing.T) {
		writeSourceTestPackageJSON(t, modulesPath, "auth", &meta.IrModule{Version: "2.1.0", ApplicationStr: "auth"})
		coordinator := NewCoordinator(runtimeScope, WithRegistryProvider(&fakeRegistryProvider{}), WithLockStore(NewLockStore(WithLockStoreDefaultChoysumPath(runtimeScope.cfg.DefaultChoysumPath))))
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
