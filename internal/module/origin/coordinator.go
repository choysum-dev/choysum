// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/mod/semver"
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

var strictSemVerV = regexp.MustCompile(`^v\d+\.\d+\.\d+([\-\+].+)?$`)
var strictSemVerNoV = regexp.MustCompile(`^\d+\.\d+\.\d+([\-\+].+)?$`)

func decodeLocalManifest(r io.Reader) (*meta.IrModule, error) {
	module := &meta.IrModule{}
	if err := json.NewDecoder(r).Decode(module); err != nil {
		return nil, err
	}
	module.Status = meta.ToInstall
	return module, nil
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

func validateAndNormalizeManifestSemVer(mod *meta.IrModule, manifestHint string) error {
	if mod == nil {
		return nil
	}
	ver := strings.TrimSpace(mod.Version)
	if ver == "" {
		return xfmt.Errorf("empty manifest version (module=%q, manifest=%q)", strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
	}
	if strings.HasPrefix(ver, "v") {
		if !strictSemVerV.MatchString(ver) || !semver.IsValid(ver) {
			return xfmt.Errorf("invalid manifest version %q (module=%q, manifest=%q); expected SemVer like v0.1.0", ver, strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
		}
		return nil
	}
	if strictSemVerNoV.MatchString(ver) {
		v := "v" + ver
		if !strictSemVerV.MatchString(v) || !semver.IsValid(v) {
			return xfmt.Errorf("invalid manifest version %q (module=%q, manifest=%q); expected SemVer like v0.1.0", ver, strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
		}
		mod.Version = v
		return nil
	}
	return xfmt.Errorf("invalid manifest version %q (module=%q, manifest=%q); expected SemVer like v0.1.0", ver, strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
}

func (c *Coordinator) peekLocalModule(moduleName string) (*meta.IrModule, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, xfmt.Errorf("module name is empty")
	}
	moduleDir := filepath.Join(runtimeOptionsFromScope(c.runtimeScope).addonsPath, moduleName)
	manifestPath := filepath.Join(moduleDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, xfmt.Errorf("stat local module manifest failed: %w", err)
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, xfmt.Errorf("open manifest: %w", err)
	}
	defer f.Close()

	module, err := decodeLocalManifest(f)
	if err != nil {
		return nil, xfmt.Errorf("decode manifest: %w", err)
	}
	module.Name = moduleName
	module.Path = moduleDir
	if err := applyEntryPoints(module); err != nil {
		return nil, err
	}
	if err := validateAndNormalizeManifestSemVer(module, manifestPath); err != nil {
		return nil, err
	}
	return module, nil
}

func (c *Coordinator) peekRegistryModule(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	if c.registryProvider == nil {
		return nil, xfmt.Errorf("registry provider is nil")
	}
	entry, err := c.registryStore.Resolve(parsed.RegistryAlias)
	if err != nil {
		return nil, err
	}
	return c.registryProvider.PeekManifest(ctx, entry.URL, parsed.ModuleName, parsed.Version)
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
		return nil, xfmt.Errorf("module %s not found in addons path", moduleName)
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

	moduleDir := filepath.Join(runtimeOptionsFromScope(c.runtimeScope).addonsPath, parsed.LocalName)
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
	return nil, xfmt.Errorf("module %s not found in addons path", moduleName)
}

func (c *Coordinator) resolveRegistry(ctx context.Context, parsed ParsedInput) (*meta.IrModule, error) {
	if c.registryProvider == nil {
		return nil, xfmt.Errorf("registry provider is nil")
	}
	entry, err := c.registryStore.Resolve(parsed.RegistryAlias)
	if err != nil {
		return nil, err
	}
	mod, err := c.registryProvider.Fetch(ctx, entry.URL, parsed.ModuleName, parsed.Version)
	if err != nil {
		return nil, err
	}
	if err := c.lockStore.UpsertBinding(WorkspaceRoot(c.runtimeScope), Binding{
		ModuleName:      strings.TrimSpace(parsed.ModuleName),
		OriginType:      OriginTypeRegistry,
		OriginRef:       parsed.CanonicalRef(),
		ResolvedVersion: strings.TrimSpace(mod.Version),
		LocalPath:       strings.TrimSpace(mod.Path),
	}); err != nil {
		return nil, err
	}
	return mod, nil
}
