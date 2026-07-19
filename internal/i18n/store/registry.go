// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"sort"
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
	// hostAppsCache is a sorted list of non-core host applications. nil means dirty.
	hostAppsCache []string
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
	r.hostAppsCache = nil
	return s
}

// Lookup resolves the module's application, then looks up in that store's cache.
// Framework module "core" is hosted in each real application's table (Scheme A);
// Lookup probes host app stores until a hit (terms are identical across hosts).
func (r *Registry) Lookup(module, lang, scopeKey, src, kind string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "core" {
		return r.lookupFrameworkModule(module, lang, scopeKey, src, kind)
	}
	app, ok := r.ApplicationForModule(module)
	if !ok || app == "" || app == "core" {
		return "", false
	}
	return r.StoreFor(app).Lookup(module, lang, scopeKey, src, kind)
}

func (r *Registry) lookupFrameworkModule(module, lang, scopeKey, src, kind string) (string, bool) {
	apps := r.listHostApplications()
	for _, app := range apps {
		if val, ok := r.StoreFor(app).Lookup(module, lang, scopeKey, src, kind); ok {
			return val, true
		}
	}
	return "", false
}

func (r *Registry) listHostApplications() []string {
	r.mu.RLock()
	if r.hostAppsCache != nil {
		out := append([]string(nil), r.hostAppsCache...)
		r.mu.RUnlock()
		return out
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hostAppsCache != nil {
		return append([]string(nil), r.hostAppsCache...)
	}

	seen := map[string]struct{}{}
	for _, app := range r.moduleToApp {
		app = strings.TrimSpace(app)
		if app == "" || app == "core" {
			continue
		}
		seen[app] = struct{}{}
	}
	for app := range r.stores {
		app = strings.TrimSpace(app)
		if app == "" || app == "core" {
			continue
		}
		seen[app] = struct{}{}
	}

	// Prefer in-memory hosts for hot Lookup paths. Fall back to IrModule only
	// when the registry has not observed any host application yet.
	if len(seen) == 0 && r.runtimeScope != nil && r.runtimeScope.Session() != nil {
		session := r.runtimeScope.Session()
		if session.Migrator().HasTable((&meta.IrModule{}).TableName()) {
			var modules []meta.IrModule
			if err := session.Where("status = ?", meta.Installed).Find(&modules).Error; err == nil {
				for _, mod := range modules {
					app := strings.TrimSpace(mod.ApplicationStr)
					if app == "" || app == "core" {
						continue
					}
					seen[app] = struct{}{}
				}
			}
		}
	}

	out := make([]string, 0, len(seen))
	for app := range seen {
		out = append(out, app)
	}
	sort.Strings(out)
	r.hostAppsCache = append([]string(nil), out...)
	return append([]string(nil), out...)
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
	r.hostAppsCache = nil
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
