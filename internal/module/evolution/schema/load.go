// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

// Overridable helpers for tests.
var (
	listDeclarationsFn          = modmeta.ListDeclarations
	expandModelsAlongExtendsFn  = modmeta.ExpandModelsAlongExtends
	loadEffectiveModelsByKeysFn = loadEffectiveModelsByKeys
)

// loadModelsForSchema loads models for DDL.
// Trigger scope is the current module's declarations; desired shape prefers
// effective tip rows (IMD), falling back to expanded declarations.
func loadModelsForSchema(runtimeScope scope.Scope, module *meta.Module) ([]*meta.Model, error) {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil, fmt.Errorf("runtime scope is nil")
	}
	if module == nil {
		return nil, fmt.Errorf("module is nil")
	}
	if !module.Id.Valid || strings.TrimSpace(module.Id.String) == "" {
		return []*meta.Model{}, nil
	}
	db := runtimeScope.Session().DB

	absFalse := false
	decls, err := listDeclarationsFn(db, modmeta.DeclarationQuery{
		ModuleID:    module.Id.String,
		Abstract:    &absFalse,
		PreloadTree: true,
	})
	if err != nil {
		return nil, xfmt.Errorf("error getting models by module id: %w", err)
	}
	if err := expandModelsAlongExtendsFn(db, decls); err != nil {
		return nil, xfmt.Errorf("error expanding model extends for schema: %w", err)
	}

	keys := make([]modmeta.LogicalKey, 0, len(decls))
	for _, d := range decls {
		if d == nil {
			continue
		}
		k := modmeta.LogicalKey{Application: d.Application, Name: d.Name}.Normalized()
		if !k.Valid() {
			// Path-only / incomplete identity: keep declaration as-is later.
			continue
		}
		keys = append(keys, k)
	}

	effectiveByKey, err := loadEffectiveModelsByKeysFn(db, keys)
	if err != nil {
		return nil, err
	}

	// Prefer effective shape; fall back to expanded declaration.
	candidates := make([]*meta.Model, 0, len(decls))
	seenKey := map[string]struct{}{}
	for _, d := range decls {
		if d == nil {
			continue
		}
		k := modmeta.LogicalKey{Application: d.Application, Name: d.Name}.Normalized()
		key := k.Application + "\x00" + k.Name
		if k.Valid() {
			if _, ok := seenKey[key]; ok {
				continue
			}
			seenKey[key] = struct{}{}
			if eff, ok := effectiveByKey[key]; ok && eff != nil {
				candidates = append(candidates, eff)
				continue
			}
		}
		candidates = append(candidates, d)
	}

	// Effective tips may omit Extends-only ancestors in Fields; expand again.
	if err := expandModelsAlongExtendsFn(db, candidates); err != nil {
		return nil, xfmt.Errorf("error expanding effective model extends for schema: %w", err)
	}

	if err := rejectConflictingModelTables(candidates); err != nil {
		return nil, err
	}

	filtered := make([]*meta.Model, 0, len(candidates))
	for _, model := range candidates {
		if model == nil || model.Abstract {
			continue
		}
		if model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered, nil
}

func loadEffectiveModelsByKeys(db *gorm.DB, keys []modmeta.LogicalKey) (map[string]*meta.Model, error) {
	out := make(map[string]*meta.Model, len(keys))
	if db == nil || len(keys) == 0 {
		return out, nil
	}

	type pair struct{ app, name string }
	want := make(map[string]pair, len(keys))
	for _, key := range keys {
		n := key.Normalized()
		if !n.Valid() {
			continue
		}
		want[n.Application+"\x00"+n.Name] = pair{n.Application, n.Name}
	}
	if len(want) == 0 {
		return out, nil
	}

	placeholders := make([]string, 0, len(want))
	args := make([]any, 0, len(want)*2)
	for _, p := range want {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, p.app, p.name)
	}

	orderID := func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }
	var rows []meta.Model
	err := db.Model(&meta.Model{}).
		Where("(module_id IS NULL OR module_id = '')").
		Where("(application, name) IN ("+strings.Join(placeholders, ", ")+")", args...).
		Preload("Fields", orderID).
		Preload("Fields.Decorators", orderID).
		Preload("Fields.Decorators.Arguments", orderID).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, xfmt.Errorf("load effective models: %w", err)
	}

	for i := range rows {
		m := rows[i]
		n := modmeta.LogicalKey{Application: m.Application, Name: m.Name}.Normalized()
		key := n.Application + "\x00" + n.Name
		if _, ok := out[key]; ok {
			continue // already have highest id for this key
		}
		copied := m
		out[key] = &copied
	}
	return out, nil
}

func rejectConflictingModelTables(models []*meta.Model) error {
	byTable := map[string]string{} // table → application\x00name
	for _, m := range models {
		if m == nil {
			continue
		}
		table := strings.TrimSpace(m.ModelTable)
		if table == "" {
			continue
		}
		key := strings.TrimSpace(m.Application) + "\x00" + strings.TrimSpace(m.Name)
		if prev, ok := byTable[table]; ok && prev != key {
			return fmt.Errorf("conflicting ModelTable %q for %q and %q", table, prev, key)
		}
		byTable[table] = key
	}
	return nil
}
