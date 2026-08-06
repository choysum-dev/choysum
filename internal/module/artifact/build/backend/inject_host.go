// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// injectHost adapts ModuleBuilder to injectappmodel.Host.
type injectHost struct {
	b *ModuleBuilder
}

var _ injectappmodel.Host = injectHost{}

func (h injectHost) Module() *meta.Module {
	if h.b == nil {
		return nil
	}
	return h.b.module
}

func (h injectHost) SessionDB() *gorm.DB {
	if h.b == nil || h.b.runtimeScope == nil || h.b.runtimeScope.Session() == nil {
		return nil
	}
	return h.b.runtimeScope.Session().DB
}

func (h injectHost) ModulesPath() string {
	if h.b == nil {
		return ""
	}
	return h.b.resolvedRuntimeOptions().modulesPath
}

func (h injectHost) EntryPointImports() []string {
	if h.b == nil {
		return nil
	}
	return h.b.entryPointImports()
}

func (h injectHost) SetEntryPointImports(imports []string) {
	if h.b == nil {
		return
	}
	if setter, ok := h.b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if setter, ok := h.b.prebuildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
}

func (h injectHost) RegisterVirtualSource(path, contents string) {
	if h.b == nil {
		return
	}
	if registrar, ok := h.b.buildPlugin.(interface {
		RegisterVirtualSource(path string, contents string)
	}); ok {
		registrar.RegisterVirtualSource(path, contents)
	}
	if registrar, ok := h.b.prebuildPlugin.(interface {
		RegisterVirtualSource(path string, contents string)
	}); ok {
		registrar.RegisterVirtualSource(path, contents)
	}
}

func (b *ModuleBuilder) ensureInjectSession() *injectappmodel.Session {
	if b == nil {
		return nil
	}
	if b.injectSession == nil {
		reg := b.injectRegistry
		if reg == nil {
			reg = injectappmodel.DefaultRegistry()
		}
		b.injectSession = injectappmodel.NewSession(injectHost{b}, reg)
	}
	return b.injectSession
}

func (b *ModuleBuilder) releaseInjectSchedules() {
	if b == nil || b.injectSession == nil {
		return
	}
	b.injectSession.ReleaseSchedules()
}
