// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"fmt"
	"strings"
	"sync"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
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

// ExistingStore returns the TermStore for an application without creating one.
func (r *Registry) ExistingStore(application string) (*TermStore, bool) {
	application = strings.TrimSpace(application)
	if application == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.stores[application]
	return s, ok
}

// Lookup resolves the module's application, then looks up in that store's cache.
// Framework module "core" uses StoreFor("core") (Scheme B).
func (r *Registry) Lookup(module, lang, scopeKey, src, kind string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "core" {
		return r.StoreFor("core").Lookup(module, lang, scopeKey, src, kind)
	}
	app, ok := r.ApplicationForModule(module)
	if !ok || app == "" {
		return "", false
	}
	return r.StoreFor(app).Lookup(module, lang, scopeKey, src, kind)
}

// ApplicationForModule returns the cached ApplicationStr for a module name.
// Memory-only: callers must populate via WarmRuntimeCache or
// RememberModuleApplication. Lookup must not sync-query meta_module (SQLite
// pool deadlock when an RPC already holds a connection on the JS thread).
func (r *Registry) ApplicationForModule(module string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "" {
		return "", false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.moduleToApp[module]
	return app, ok
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

// WarmRuntimeCache pre-fills module→application maps and loads terminology for
// installed applications. Call before serving RPCs so Lookup never hits the DB
// on the JS thread (SQLite pool deadlock with open RPC transactions).
//
// Languages come from base_language.Code when that table exists, unioned with
// distinct lang values already stored in each application's term table.
func (r *Registry) WarmRuntimeCache() error {
	if r == nil || r.runtimeScope == nil {
		return nil
	}
	session := r.runtimeScope.Session()
	if session == nil || session.DB == nil {
		return nil
	}

	apps := map[string]struct{}{"core": {}}
	if session.Migrator().HasTable((&meta.Module{}).TableName()) {
		// Table scan (not meta.Module Find) so soft-delete columns are optional in tests.
		type installedModuleRow struct {
			Name           string
			ApplicationStr string `gorm:"column:application_str"`
		}
		var mods []installedModuleRow
		if err := session.Table((&meta.Module{}).TableName()).
			Select("name", "application_str").
			Where("status = ?", meta.Installed).
			Order("id ASC").
			Find(&mods).Error; err != nil {
			return fmt.Errorf("list installed modules for terminology warm: %w", err)
		}
		for i := range mods {
			name := strings.TrimSpace(mods[i].Name)
			app := strings.TrimSpace(mods[i].ApplicationStr)
			if name != "" && app != "" {
				r.RememberModuleApplication(name, app)
			}
			if app != "" {
				apps[app] = struct{}{}
			}
		}
	}

	langs := map[string]struct{}{}
	if session.Migrator().HasTable("base_language") {
		var codes []string
		if err := session.Table("base_language").Distinct("code").Pluck("code", &codes).Error; err != nil {
			return fmt.Errorf("list language codes for terminology warm: %w", err)
		}
		for _, code := range codes {
			code = strings.TrimSpace(code)
			if code != "" {
				langs[code] = struct{}{}
			}
		}
	}

	var firstErr error
	for app := range apps {
		tableName := i18nmodels.TranslationTermTableName(app)
		if !session.Migrator().HasTable(tableName) {
			continue
		}
		var termLangs []string
		if err := session.Table(tableName).Distinct("lang").Pluck("lang", &termLangs).Error; err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("list term langs for %s: %w", app, err)
			}
			continue
		}
		for _, code := range termLangs {
			code = strings.TrimSpace(code)
			if code != "" {
				langs[code] = struct{}{}
			}
		}
	}
	if len(langs) == 0 {
		return firstErr
	}

	for app := range apps {
		ts := r.StoreFor(app)
		for lang := range langs {
			if err := ts.WarmLanguage(lang); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("warm %s lang=%s: %w", app, lang, err)
			}
		}
	}
	return firstErr
}
