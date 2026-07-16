// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Registry holds per-application TermStores and resolves module → application.
type Registry struct {
	mu           sync.RWMutex
	runtimeScope scope.Scope
	stores       map[string]*TermStore
	moduleToApp  map[string]string
}

// NewRegistry creates a registry bound to a runtime scope (shared DB).
func NewRegistry(runtimeScope scope.Scope) *Registry {
	return &Registry{
		runtimeScope: runtimeScope,
		stores:       make(map[string]*TermStore),
		moduleToApp:  make(map[string]string),
	}
}

// StoreFor returns (creating if needed) the TermStore for an application.
func (r *Registry) StoreFor(application string) *TermStore {
	application = strings.TrimSpace(application)
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.stores[application]; ok {
		return s
	}
	s := NewTermStore(r.runtimeScope, application)
	r.stores[application] = s
	return s
}

// Lookup resolves the module's application, then looks up in that store's cache.
func (r *Registry) Lookup(module, lang, scopeKey, src, kind string) (string, bool) {
	app, ok := r.ApplicationForModule(module)
	if !ok || app == "" || app == "core" {
		return "", false
	}
	return r.StoreFor(app).Lookup(module, lang, scopeKey, src, kind)
}

// ApplicationForModule returns the ApplicationStr for a module name.
func (r *Registry) ApplicationForModule(module string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "" {
		return "", false
	}

	r.mu.RLock()
	if app, ok := r.moduleToApp[module]; ok {
		r.mu.RUnlock()
		return app, true
	}
	r.mu.RUnlock()

	app, ok := r.loadModuleApplication(module)
	if !ok {
		return "", false
	}

	r.mu.Lock()
	r.moduleToApp[module] = app
	r.mu.Unlock()
	return app, true
}

// RememberModuleApplication caches a module → application mapping (tests / install hooks).
func (r *Registry) RememberModuleApplication(module, application string) {
	module = strings.TrimSpace(module)
	application = strings.TrimSpace(application)
	if module == "" || application == "" {
		return
	}
	r.mu.Lock()
	r.moduleToApp[module] = application
	r.mu.Unlock()
}

func (r *Registry) loadModuleApplication(module string) (string, bool) {
	if r.runtimeScope == nil || r.runtimeScope.Session() == nil {
		return "", false
	}
	session := r.runtimeScope.Session()
	if !session.Migrator().HasTable((&meta.IrModule{}).TableName()) {
		return "", false
	}
	var mod meta.IrModule
	if err := session.Where("name = ?", module).Take(&mod).Error; err != nil {
		return "", false
	}
	app := strings.TrimSpace(mod.ApplicationStr)
	if app == "" {
		return "", false
	}
	return app, true
}
