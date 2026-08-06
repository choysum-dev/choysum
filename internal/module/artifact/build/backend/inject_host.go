// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	"gorm.io/gorm"
)

func (b *ModuleBuilder) injectBuildCtx() injectappmodel.BuildCtx {
	ctx := injectappmodel.BuildCtx{}
	if b == nil {
		return ctx
	}
	ctx.Module = b.module
	ctx.ModulesPath = b.resolvedRuntimeOptions().modulesPath
	if b.runtimeScope != nil && b.runtimeScope.Session() != nil {
		ctx.DB = b.runtimeScope.Session().DB
	}
	return ctx
}

func (b *ModuleBuilder) ensureInjectSession() *injectappmodel.Session {
	if b == nil {
		return nil
	}
	ctx := b.injectBuildCtx()
	if b.injectSession == nil {
		reg := b.injectRegistry
		if reg == nil {
			reg = injectappmodel.DefaultRegistry()
		}
		b.injectSession = injectappmodel.NewSession(ctx, reg)
	} else {
		*b.injectSession.Context() = ctx
	}
	return b.injectSession
}

func (b *ModuleBuilder) releaseInjectSchedules() {
	if b == nil || b.injectSession == nil {
		return
	}
	b.injectSession.ReleaseSchedules()
}

// applyInjectEffects registers virtual sources and merges entry-point imports.
func (b *ModuleBuilder) applyInjectEffects(fx injectappmodel.Effects) {
	if b == nil {
		return
	}
	for _, f := range fx.Files {
		path := strings.TrimSpace(f.Path)
		if path == "" {
			continue
		}
		if registrar, ok := b.buildPlugin.(interface {
			RegisterVirtualSource(path string, contents string)
		}); ok {
			registrar.RegisterVirtualSource(path, f.Contents)
		}
		if registrar, ok := b.prebuildPlugin.(interface {
			RegisterVirtualSource(path string, contents string)
		}); ok {
			registrar.RegisterVirtualSource(path, f.Contents)
		}
	}
	if len(fx.Imports) == 0 {
		return
	}
	imports := mergeImportPaths(b.entryPointImports(), fx.Imports)
	if setter, ok := b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if setter, ok := b.prebuildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
}

func mergeImportPaths(existing, add []string) []string {
	out := make([]string, 0, len(existing)+len(add))
	seen := make(map[string]struct{}, len(existing)+len(add))
	for _, group := range [][]string{existing, add} {
		for _, p := range group {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

// sessionDB is used by tests that previously went through injectHost.
func (b *ModuleBuilder) sessionDB() *gorm.DB {
	if b == nil || b.runtimeScope == nil || b.runtimeScope.Session() == nil {
		return nil
	}
	return b.runtimeScope.Session().DB
}
