// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"context"
	"fmt"
	"sort"
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

// replaceModuleApplications atomically replaces the module→application map.
// WarmRuntimeCache uses this so uninstalled / soft-deleted modules stop resolving.
func (r *Registry) replaceModuleApplications(next map[string]string) {
	if next == nil {
		next = make(map[string]string)
	}
	r.mu.Lock()
	r.moduleToApp = next
	r.mu.Unlock()
}

// WarmRuntimeCache pre-fills module→application maps and loads terminology for
// installed applications. Call before serving RPCs so Lookup never hits the DB
// on the JS thread (SQLite pool deadlock with open RPC transactions).
//
// Languages come from base_language.Code when that table exists, unioned with
// distinct lang values already stored in each application's term table.
// ctx is applied to DB sessions so startup cancellation can abort warm queries.
func (r *Registry) WarmRuntimeCache(ctx context.Context) error {
	if r == nil || r.runtimeScope == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	session := r.runtimeScope.WithContext(ctx).Session()
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
		q := session.Table((&meta.Module{}).TableName()).
			Select("name", "application_str").
			Where("status = ?", meta.Installed)
		if session.Migrator().HasColumn((&meta.Module{}).TableName(), "deleted_at") {
			q = q.Where("deleted_at IS NULL")
		}
		if err := q.Order("id ASC").Find(&mods).Error; err != nil {
			return fmt.Errorf("list installed modules for terminology warm: %w", err)
		}
		next := make(map[string]string, len(mods))
		for i := range mods {
			name := strings.TrimSpace(mods[i].Name)
			app := strings.TrimSpace(mods[i].ApplicationStr)
			if name != "" && app != "" {
				next[name] = app
			}
			if app != "" {
				apps[app] = struct{}{}
			}
		}
		r.replaceModuleApplications(next)
	} else {
		// No catalog table: drop any Remember/prior-warm mappings so Lookup cannot
		// resolve modules that are no longer installable in this DB.
		r.replaceModuleApplications(nil)
	}

	langs := map[string]struct{}{}
	if session.Migrator().HasTable("base_language") {
		var codes []string
		lq := session.Table("base_language").Distinct("code")
		if session.Migrator().HasColumn("base_language", "deleted_at") {
			lq = lq.Where("deleted_at IS NULL")
		}
		if err := lq.Pluck("code", &codes).Error; err != nil {
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
	appNames := sortedMapKeys(apps)
	appsWithTermTable := make([]string, 0, len(appNames))
	for _, app := range appNames {
		tableName := i18nmodels.TranslationTermTableName(app)
		if !session.Migrator().HasTable(tableName) {
			continue
		}
		appsWithTermTable = append(appsWithTermTable, app)
		var termLangs []string
		tq := session.Table(tableName).Distinct("lang")
		if session.Migrator().HasColumn(tableName, "deleted_at") {
			tq = tq.Where("deleted_at IS NULL")
		}
		if err := tq.Pluck("lang", &termLangs).Error; err != nil {
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

	langCodes := sortedMapKeys(langs)
	for _, app := range appsWithTermTable {
		if err := ctx.Err(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return firstErr
		}
		ts := r.StoreFor(app)
		for _, lang := range langCodes {
			if err := ts.warmLanguage(ctx, lang); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("warm %s lang=%s: %w", app, lang, err)
			}
		}
	}
	return firstErr
}

func sortedMapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
