// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Coordinator struct {
	runtimeScope     scope.Scope
	lockStore        *LockStore
	registryStore    *registry.Store
	registryProvider registry.Provider
}

type Option func(*Coordinator)

func WithLockStore(store *LockStore) Option {
	return func(c *Coordinator) {
		if store != nil {
			c.lockStore = store
		}
	}
}

func WithRegistryStore(store *registry.Store) Option {
	return func(c *Coordinator) {
		if store != nil {
			c.registryStore = store
		}
	}
}

func WithRegistryProvider(provider registry.Provider) Option {
	return func(c *Coordinator) {
		if provider != nil {
			c.registryProvider = provider
		}
	}
}

func NewCoordinator(runtimeScope scope.Scope, opts ...Option) *Coordinator {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	defaultChoysumPath := strings.TrimSpace(runtimeOpts.defaultChoysumPath)
	c := &Coordinator{
		runtimeScope:     runtimeScope,
		lockStore:        NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath)),
		registryStore:    registry.NewStore(registry.WithDefaultChoysumPath(defaultChoysumPath)),
		registryProvider: registry.NewProvider(runtimeScope),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func applyEntryPoints(module *meta.IrModule) error {
	if module == nil {
		return nil
	}
	entryPointsMap := make(map[string]string)
	if module.EntryPoints != nil {
		if err := json.Unmarshal(module.EntryPoints, &entryPointsMap); err != nil {
			return xfmt.Errorf("error unmarshalling entry points: %w", err)
		}
		if webEntryPoint, ok := entryPointsMap["web"]; ok {
			module.WebEntryPoint = webEntryPoint
		}
		if serviceEntryPoint, ok := entryPointsMap["service"]; ok {
			module.ServiceEntryPoint = serviceEntryPoint
		}
	}
	return nil
}

func validateAndNormalizeModuleSemVer(mod *meta.IrModule, sourceHint string) error {
	if mod == nil {
		return nil
	}
	ver := strings.TrimSpace(mod.Version)
	if ver == "" {
		return xfmt.Errorf("empty module version (module=%q, source=%q)", strings.TrimSpace(mod.Name), strings.TrimSpace(sourceHint))
	}
	normalized, err := contract.NormalizeVersion(ver)
	if err != nil {
		return xfmt.Errorf("invalid module version %q (module=%q, source=%q); expected SemVer like v0.1.0", ver, strings.TrimSpace(mod.Name), strings.TrimSpace(sourceHint))
	}
	mod.Version = normalized
	return nil
}

func (c *Coordinator) peekLocalModule(moduleName string) (*meta.IrModule, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, xfmt.Errorf("module name is empty")
	}
	moduleDir := filepath.Join(runtimeOptionsFromScope(c.runtimeScope).modulesPath, moduleName)
	packageJSONPath := filepath.Join(moduleDir, "package.json")
	if _, err := os.Stat(packageJSONPath); err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, xfmt.Errorf("stat local module package.json failed: %w", err)
	}

	raw, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, xfmt.Errorf("read package.json: %w", err)
	}

	result, err := contract.ParsePackageJSONToIrModule(raw, moduleDir, nil)
	if err != nil {
		return nil, xfmt.Errorf("parse package.json: %w", err)
	}
	if result == nil || result.Module == nil {
		return nil, xfmt.Errorf("parse package.json: empty module result")
	}
	module := result.Module
	module.Name = moduleName
	module.Path = moduleDir
	if err := applyEntryPoints(module); err != nil {
		return nil, err
	}
	if err := validateAndNormalizeModuleSemVer(module, packageJSONPath); err != nil {
		return nil, err
	}
	return module, nil
}

type registrySourceResolution struct {
	registryURL string
	packageName string
	integrity   string
}

func canonicalRegistryOriginRef(parsed ParsedInput, resolvedVersion string) string {
	if parsed.Kind != InputKindRegistry {
		return strings.TrimSpace(parsed.LocalName)
	}
	if strings.EqualFold(strings.TrimSpace(parsed.Version), "latest") {
		if resolvedVersion = strings.TrimSpace(resolvedVersion); resolvedVersion != "" {
			return strings.TrimSpace(parsed.RegistryAlias) + "/" + strings.TrimSpace(parsed.ModuleName) + "@" + resolvedVersion
		}
	}
	return parsed.CanonicalRef()
}

func resolveBindingIntegrity(catalogIntegrity string, mod *meta.IrModule) string {
	if mod != nil {
		if integrity := strings.TrimSpace(mod.Integrity); integrity != "" {
			return integrity
		}
	}
	return strings.TrimSpace(catalogIntegrity)
}

func (c *Coordinator) peekRegistryModule(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	if c.registryProvider == nil {
		return nil, xfmt.Errorf("registry provider is nil")
	}
	resolved, err := c.resolveRegistrySource(ctx, parsed)
	if err != nil {
		return nil, err
	}
	return c.registryProvider.PeekManifest(ctx, resolved.registryURL, parsed.ModuleName, resolved.packageName, parsed.Version)
}

func looksLikeCatalogRegistryURL(registryURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(registryURL))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	pathLower := strings.ToLower(parsed.Path)
	if strings.HasSuffix(pathLower, ".json") || strings.Contains(pathLower, "/api/") || strings.HasSuffix(pathLower, "/api") {
		return true
	}

	hostLower := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if hostLower == "" {
		return false
	}
	if strings.Contains(hostLower, "registry.npmjs.org") ||
		strings.Contains(hostLower, "registry.npmmirror.com") ||
		strings.Contains(hostLower, "registry.yarnpkg.com") ||
		strings.Contains(hostLower, "npm.pkg.github.com") ||
		hostLower == "localhost" ||
		hostLower == "127.0.0.1" ||
		hostLower == "::1" {
		return false
	}
	if strings.Contains(hostLower, "catalog.") {
		return true
	}
	if hostLower == "github.com" {
		return true
	}
	return true
}

func (c *Coordinator) resolveRegistrySource(ctx context.Context, parsed ParsedInput) (registrySourceResolution, error) {
	entry, err := c.registryStore.Resolve(parsed.RegistryAlias)
	if err != nil {
		return registrySourceResolution{}, err
	}
	registryURL := strings.TrimSpace(entry.URL)
	moduleName := strings.TrimSpace(parsed.ModuleName)
	resolved := registrySourceResolution{registryURL: registryURL, packageName: moduleName}

	if !looksLikeCatalogRegistryURL(registryURL) {
		return resolved, nil
	}

	catalog := registry.NewCatalog(c.runtimeScope)
	item, err := catalog.Info(ctx, registryURL, moduleName)
	if err != nil {
		return registrySourceResolution{}, xfmt.Errorf("resolve catalog source failed (registry=%s module=%s): %w", strings.TrimSpace(parsed.RegistryAlias), moduleName, err)
	}
	resolved.packageName = item.ResolvedNPMPackage()
	if resolved.packageName == "" {
		return registrySourceResolution{}, xfmt.Errorf("catalog module %q in registry %q has empty npm package source", moduleName, strings.TrimSpace(parsed.RegistryAlias))
	}
	if sourceRegistry := item.ResolvedNPMRegistry(registryURL); sourceRegistry != "" {
		resolved.registryURL = sourceRegistry
	}
	if item.Source != nil {
		resolved.integrity = strings.TrimSpace(item.Source.Integrity)
	}

	return resolved, nil
}

func (c *Coordinator) Peek(ctx context.Context, input string) (*meta.IrModule, error) {
	if c == nil || c.runtimeScope == nil {
		return nil, xfmt.Errorf("origin coordinator env is nil")
	}
	parsed, err := ParseInput(input)
	if err != nil {
		return nil, err
	}

	switch parsed.Kind {
	case InputKindRegistry:
		return c.peekRegistryModule(ctx, parsed)
	case InputKindLocal:
		moduleName := strings.TrimSpace(parsed.LocalName)
		mod, err := c.peekLocalModule(moduleName)
		if err == nil {
			return mod, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, xfmt.Errorf("module %s not found in modules path", moduleName)
	default:
		return nil, xfmt.Errorf("unsupported origin input kind: %s", parsed.Kind)
	}
}

func (c *Coordinator) ResolveInstallModule(ctx context.Context, input string) (*meta.IrModule, error) {
	return c.Fetch(ctx, input)
}

func (c *Coordinator) Fetch(ctx context.Context, input string) (*meta.IrModule, error) {
	if c == nil || c.runtimeScope == nil {
		return nil, xfmt.Errorf("origin coordinator env is nil")
	}
	parsed, err := ParseInput(input)
	if err != nil {
		return nil, err
	}

	switch parsed.Kind {
	case InputKindRegistry:
		return c.resolveRegistry(ctx, parsed)
	case InputKindLocal:
		return c.resolveLocal(ctx, parsed)
	default:
		return nil, xfmt.Errorf("unsupported origin input kind: %s", parsed.Kind)
	}
}

func (c *Coordinator) Purge(ctx context.Context, moduleName string) error {
	if c == nil || c.runtimeScope == nil {
		return xfmt.Errorf("origin coordinator env is nil")
	}
	parsed, err := ParseInput(moduleName)
	if err != nil {
		return err
	}
	if parsed.Kind != InputKindLocal {
		return xfmt.Errorf("purge accepts local module name only: %s", moduleName)
	}

	workspaceRoot := WorkspaceRoot(c.runtimeScope)
	if err := c.lockStore.DeleteBinding(workspaceRoot, parsed.LocalName); err != nil {
		return err
	}

	moduleDir := filepath.Join(runtimeOptionsFromScope(c.runtimeScope).modulesPath, parsed.LocalName)
	if err := os.RemoveAll(moduleDir); err != nil {
		return xfmt.Errorf("purge module %s failed: %w", parsed.LocalName, err)
	}
	return nil
}

func (c *Coordinator) resolveLocal(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	moduleName := strings.TrimSpace(parsed.LocalName)
	if mod, err := c.peekLocalModule(moduleName); err == nil {
		if err := c.lockStore.UpsertBinding(WorkspaceRoot(c.runtimeScope), Binding{
			ModuleName:      moduleName,
			OriginType:      OriginTypeLocal,
			OriginRef:       moduleName,
			ResolvedVersion: strings.TrimSpace(mod.Version),
			LocalPath:       strings.TrimSpace(mod.Path),
		}); err != nil {
			return nil, err
		}
		return mod, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, xfmt.Errorf("load local module %s failed: %w", moduleName, err)
	}
	return nil, xfmt.Errorf("module %s not found in modules path", moduleName)
}

func (c *Coordinator) resolveRegistry(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	if c.registryProvider == nil {
		return nil, xfmt.Errorf("registry provider is nil")
	}
	resolved, err := c.resolveRegistrySource(ctx, parsed)
	if err != nil {
		return nil, err
	}
	mod, err := c.registryProvider.Fetch(ctx, resolved.registryURL, parsed.ModuleName, resolved.packageName, parsed.Version)
	if err != nil {
		return nil, err
	}
	resolvedVersion := strings.TrimSpace(mod.Version)
	if err := c.lockStore.UpsertBinding(WorkspaceRoot(c.runtimeScope), Binding{
		ModuleName:      strings.TrimSpace(parsed.ModuleName),
		OriginType:      OriginTypeRegistry,
		OriginRef:       canonicalRegistryOriginRef(parsed, resolvedVersion),
		ResolvedVersion: resolvedVersion,
		Integrity:       resolveBindingIntegrity(resolved.integrity, mod),
		LocalPath:       strings.TrimSpace(mod.Path),
	}); err != nil {
		return nil, err
	}
	return mod, nil
}
