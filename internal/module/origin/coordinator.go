// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Coordinator struct {
	runtimeScope     scope.Scope
	lockStore        *LockStore
	registryProvider registry.Provider
	resolutionCache  map[string]registrySourceResolution
	cacheMu          sync.RWMutex
}

type Option func(*Coordinator)

func WithLockStore(store *LockStore) Option {
	return func(c *Coordinator) {
		if store != nil {
			c.lockStore = store
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
		registryProvider: registry.NewProvider(runtimeScope),
		resolutionCache:  make(map[string]registrySourceResolution),
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
	registryURL      string
	packageName      string
	integrity        string
	preferredVersion string
}

func canonicalRegistryOriginRef(parsed ParsedInput, resolvedVersion string) string {
	if parsed.Kind != InputKindRegistry {
		return strings.TrimSpace(parsed.LocalName)
	}
	moduleName := strings.TrimSpace(parsed.ModuleName)
	version := strings.TrimSpace(parsed.Version)
	if strings.EqualFold(version, "latest") {
		version = "latest"
		if resolved := strings.TrimSpace(resolvedVersion); resolved != "" {
			return moduleName + "@" + resolved
		}
	}
	if version == "" {
		version = "latest"
	}
	return moduleName + "@" + version
}

func resolveBindingIntegrity(catalogIntegrity string, mod *meta.IrModule) string {
	if mod != nil {
		if integrity := strings.TrimSpace(mod.Integrity); integrity != "" {
			return integrity
		}
	}
	return strings.TrimSpace(catalogIntegrity)
}

func resolveRegistryRequestedVersion(requestedVersion, preferredVersion string) string {
	requestedVersion = strings.TrimSpace(requestedVersion)
	if requestedVersion != "" {
		if strings.EqualFold(requestedVersion, "latest") {
			if preferred := strings.TrimSpace(preferredVersion); preferred != "" {
				return preferred
			}
		}
		return requestedVersion
	}
	return strings.TrimSpace(preferredVersion)
}

func (c *Coordinator) peekRegistryModule(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	if c.registryProvider == nil {
		return nil, xfmt.Errorf("registry provider is nil")
	}
	resolved, err := c.resolveRegistrySource(ctx, parsed)
	if err != nil {
		return nil, err
	}
	effectiveVersion := resolveRegistryRequestedVersion(parsed.Version, resolved.preferredVersion)
	return c.registryProvider.PeekManifest(ctx, resolved.registryURL, parsed.ModuleName, resolved.packageName, effectiveVersion)
}

func (c *Coordinator) resolveRegistrySource(ctx context.Context, parsed ParsedInput) (registrySourceResolution, error) {
	runtimeOpts := runtimeOptionsFromScope(c.runtimeScope)
	indexURL := strings.TrimSpace(runtimeOpts.moduleCatalogIndexURL)
	if indexURL == "" {
		indexURL = config.DefaultModuleCatalogIndexURL
	}
	moduleName := strings.TrimSpace(parsed.ModuleName)
	cacheKey := registrySourceResolutionCacheKey(indexURL, moduleName)
	if resolved, ok := c.lookupRegistrySourceResolution(cacheKey); ok {
		return resolved, nil
	}
	resolved := registrySourceResolution{}

	catalog := registry.NewCatalog(c.runtimeScope)
	item, err := catalog.Info(ctx, indexURL, moduleName)
	if err != nil {
		return registrySourceResolution{}, xfmt.Errorf("resolve catalog source failed (indexURL=%s module=%s): %w", indexURL, moduleName, err)
	}
	resolved.packageName = item.ResolvedNPMPackage()
	if resolved.packageName == "" {
		return registrySourceResolution{}, xfmt.Errorf("catalog module %q has empty npm package source", moduleName)
	}
	if sourceRegistry := item.ResolvedNPMRegistry(""); sourceRegistry != "" {
		resolved.registryURL = sourceRegistry
	}
	if item.Source != nil {
		resolved.integrity = strings.TrimSpace(item.Source.Integrity)
		if sourceVersion := strings.TrimSpace(item.Source.Version); sourceVersion != "" {
			resolved.preferredVersion = sourceVersion
		}
	}
	if resolved.preferredVersion == "" {
		resolved.preferredVersion = strings.TrimSpace(item.LatestVersion)
	}
	c.cacheRegistrySourceResolution(cacheKey, resolved)

	return resolved, nil
}

func registrySourceResolutionCacheKey(indexURL, moduleName string) string {
	indexURL = strings.TrimSpace(indexURL)
	moduleName = strings.TrimSpace(moduleName)
	if indexURL == "" || moduleName == "" {
		return ""
	}
	return indexURL + "|" + moduleName
}

func (c *Coordinator) lookupRegistrySourceResolution(cacheKey string) (registrySourceResolution, bool) {
	if c == nil || cacheKey == "" {
		return registrySourceResolution{}, false
	}
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	resolved, ok := c.resolutionCache[cacheKey]
	return resolved, ok
}

func (c *Coordinator) cacheRegistrySourceResolution(cacheKey string, resolved registrySourceResolution) {
	if c == nil || cacheKey == "" {
		return
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.resolutionCache == nil {
		c.resolutionCache = make(map[string]registrySourceResolution)
	}
	c.resolutionCache[cacheKey] = resolved
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
		mod, err := c.resolveRegistry(ctx, parsed)
		c.logResolveInstallOutcome(parsed, "registry", false, err)
		return mod, err
	case InputKindLocal:
		mod, localErr := c.resolveLocal(ctx, parsed)
		if localErr == nil {
			c.logResolveInstallOutcome(parsed, "local", false, nil)
			return mod, nil
		}
		// When the module is not found locally, fall back to registry resolution.
		if isModuleNotFoundError(localErr) {
			if !runtimeOptionsFromScope(c.runtimeScope).moduleInstallRegistryFallbackEnabled {
				c.logResolveInstallOutcome(parsed, "local", false, localErr)
				return nil, localErr
			}
			mod, registryErr := c.resolveRegistry(ctx, parsed)
			if registryErr == nil {
				c.logResolveInstallOutcome(parsed, "registry", true, nil)
				return mod, nil
			}
			wrapped := xfmt.Errorf("module %s not found locally and registry fallback failed: %w", parsed.LocalName, registryErr)
			c.logResolveInstallOutcome(parsed, "registry", true, wrapped)
			return nil, wrapped
		}
		c.logResolveInstallOutcome(parsed, "local", false, localErr)
		return nil, localErr
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
	effectiveVersion := resolveRegistryRequestedVersion(parsed.Version, resolved.preferredVersion)
	mod, err := c.registryProvider.Fetch(ctx, resolved.registryURL, parsed.ModuleName, resolved.packageName, effectiveVersion)
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

// isModuleNotFoundError reports whether the error indicates the module was not
// found in the local modules path (as opposed to a disk I/O or permission error).
func isModuleNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "not found in modules path")
}

func resolveInstallModuleName(parsed ParsedInput) string {
	if name := strings.TrimSpace(parsed.ModuleName); name != "" {
		return name
	}
	if name := strings.TrimSpace(parsed.LocalName); name != "" {
		return name
	}
	return ""
}

func (c *Coordinator) logResolveInstallOutcome(parsed ParsedInput, resolvedOrigin string, fallback bool, err error) {
	if c == nil || c.runtimeScope == nil || c.runtimeScope.Logger() == nil {
		return
	}
	attrs := []any{
		"module", resolveInstallModuleName(parsed),
		"input_kind", string(parsed.Kind),
		"resolved_origin", strings.TrimSpace(resolvedOrigin),
		"fallback", fallback,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		c.runtimeScope.Logger().Warn("origin install resolve failed", attrs...)
		return
	}
	c.runtimeScope.Logger().Debug("origin install resolve succeeded", attrs...)
}
