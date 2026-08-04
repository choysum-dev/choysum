// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"fmt"
	"strings"

	"github.com/rs/xid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LogicalKey identifies one effective catalog row (application + name).
type LogicalKey struct {
	Application string
	Name        string
}

func (k LogicalKey) Normalized() LogicalKey {
	return LogicalKey{
		Application: strings.TrimSpace(k.Application),
		Name:        strings.TrimSpace(k.Name),
	}
}

func (k LogicalKey) Valid() bool {
	n := k.Normalized()
	return n.Application != "" && n.Name != ""
}

// RecomputeKeys rebuilds effective projections for each logical key (deduped).
// Safe to call inside an outer transaction.
func RecomputeKeys(tx *gorm.DB, keys []LogicalKey) error {
	if tx == nil {
		return fmt.Errorf("db is nil")
	}
	seen := map[string]LogicalKey{}
	for _, key := range keys {
		n := key.Normalized()
		if !n.Valid() {
			continue
		}
		seen[n.Application+"\x00"+n.Name] = n
	}
	for _, key := range seen {
		if err := RecomputeEffective(tx, key.Application, key.Name); err != nil {
			return err
		}
	}
	return nil
}

// RecomputeEffective rebuilds one effective meta_model* tree from live meta_raw_*
// for (application, name). Preserves existing effective id when present (EDS5).
// When no live raw remains, hard-deletes the effective tree.
func RecomputeEffective(tx *gorm.DB, application, name string) error {
	if tx == nil {
		return fmt.Errorf("db is nil")
	}
	key := LogicalKey{Application: application, Name: name}.Normalized()
	if !key.Valid() {
		return fmt.Errorf("recompute requires non-empty application and name")
	}

	// Serialize concurrent recomputes for the same logical name (best-effort on SQLite).
	_ = tx.Exec("SELECT 1").Error
	if err := lockLogicalKey(tx, key); err != nil {
		return err
	}

	var raws []*RawModel
	if err := tx.
		Preload("Fields").
		Preload("Fields.Decorators").
		Preload("Fields.Decorators.Arguments").
		Preload("Services").
		Preload("Services.Parameters").
		Preload("Services.TypeParameters").
		Preload("Services.Decorators").
		Preload("Services.Decorators.Arguments").
		Preload("Decorators").
		Preload("Decorators.Arguments").
		Where("application = ? AND name = ?", key.Application, key.Name).
		Find(&raws).Error; err != nil {
		return fmt.Errorf("load meta_raw_model for %s/%s: %w", key.Application, key.Name, err)
	}

	var existing []Model
	if err := tx.Where("application = ? AND name = ?", key.Application, key.Name).Find(&existing).Error; err != nil {
		return fmt.Errorf("load effective meta_model for %s/%s: %w", key.Application, key.Name, err)
	}

	if len(raws) == 0 {
		for _, row := range existing {
			if row.Id.Valid {
				if err := DeleteEffectiveModelTree(tx, row.Id.String); err != nil {
					return err
				}
			}
		}
		return nil
	}

	models := RawModelsAsModels(raws)
	// Pull differently-named Extends parents (e.g. BaseModel) into each declaration
	// before E2 same-name union — replaces pre-persist materialize (EDS4).
	if err := ExpandModelsAlongExtends(tx, models); err != nil {
		return fmt.Errorf("expand extends for %s/%s: %w", key.Application, key.Name, err)
	}
	merged, err := MergeSameNameModelsByExtensionChain(models)
	if err != nil {
		return fmt.Errorf("E2 merge %s/%s: %w", key.Application, key.Name, err)
	}
	if merged == nil {
		return fmt.Errorf("E2 merge %s/%s returned nil", key.Application, key.Name)
	}

	effID := ""
	if len(existing) > 0 {
		// Prefer tip-like id: newest UpdatedAt then Id (stable across sibling IMD leftovers).
		tip := existing[0]
		for i := 1; i < len(existing); i++ {
			row := existing[i]
			if row.UpdatedAt.After(tip.UpdatedAt) ||
				(row.UpdatedAt.Equal(tip.UpdatedAt) && row.Id.String > tip.Id.String) {
				tip = row
			}
		}
		if tip.Id.Valid && tip.Id.String != "" {
			effID = tip.Id.String
		}
	}
	if effID == "" {
		rawTip := pickTipRaw(raws)
		if rawTip != nil && rawTip.Id.Valid && rawTip.Id.String != "" {
			effID = rawTip.Id.String
		} else {
			effID = xid.New().String()
		}
	}

	for _, row := range existing {
		if row.Id.Valid {
			if err := DeleteEffectiveModelTree(tx, row.Id.String); err != nil {
				return err
			}
		}
	}

	if err := PersistEffectiveProjection(tx, merged, effID); err != nil {
		return fmt.Errorf("persist effective %s/%s: %w", key.Application, key.Name, err)
	}
	return nil
}

func pickTipRaw(raws []*RawModel) *RawModel {
	var tip *RawModel
	for _, raw := range raws {
		if raw == nil {
			continue
		}
		if tip == nil || rawIsNewerTip(raw, tip) {
			tip = raw
		}
	}
	return tip
}

func lockLogicalKey(tx *gorm.DB, key LogicalKey) error {
	switch tx.Dialector.Name() {
	case "postgres":
		// Advisory lock keyed by app+name hash would be ideal; fall back to FOR UPDATE
		// on any existing effective row (no-op when missing).
		var m Model
		_ = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("application = ? AND name = ?", key.Application, key.Name).
			Take(&m).Error
	default:
		// SQLite / MySQL: rely on UNIQUE(application, name) + caller transaction.
	}
	return nil
}

// DeleteEffectiveModelTree hard-deletes one effective model and its shape children.
// IDs are plucked first so SQLite does not reject DELETE … WHERE id IN (SELECT … same table).
func DeleteEffectiveModelTree(db *gorm.DB, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}
	root := db
	fresh := func() *gorm.DB { return root.Session(&gorm.Session{NewDB: true}).Unscoped() }

	var serviceIDs []string
	if err := fresh().Model(&Service{}).Where("model_id = ?", modelID).Pluck("id", &serviceIDs).Error; err != nil {
		return fmt.Errorf("load effective services: %w", err)
	}
	var fieldIDs []string
	if err := fresh().Model(&Field{}).Where("model_id = ?", modelID).Pluck("id", &fieldIDs).Error; err != nil {
		return fmt.Errorf("load effective fields: %w", err)
	}

	decoratorQ := fresh().Model(&Decorator{}).Where("model_id = ?", modelID)
	if len(serviceIDs) > 0 {
		decoratorQ = decoratorQ.Or("service_id IN ?", serviceIDs)
	}
	if len(fieldIDs) > 0 {
		decoratorQ = decoratorQ.Or("field_id IN ?", fieldIDs)
	}
	var decoratorIDs []string
	if err := decoratorQ.Pluck("id", &decoratorIDs).Error; err != nil {
		return fmt.Errorf("load effective decorators: %w", err)
	}

	if len(decoratorIDs) > 0 {
		if err := fresh().Where("decorator_id IN ?", decoratorIDs).Delete(&Argument{}).Error; err != nil {
			return fmt.Errorf("delete effective arguments: %w", err)
		}
		if err := fresh().Where("id IN ?", decoratorIDs).Delete(&Decorator{}).Error; err != nil {
			return fmt.Errorf("delete effective decorators: %w", err)
		}
	}
	if len(serviceIDs) > 0 {
		if err := fresh().Where("service_id IN ?", serviceIDs).Delete(&TypeParameter{}).Error; err != nil {
			return fmt.Errorf("delete effective type parameters: %w", err)
		}
		if err := fresh().Where("service_id IN ?", serviceIDs).Delete(&Parameter{}).Error; err != nil {
			return fmt.Errorf("delete effective parameters: %w", err)
		}
		if err := fresh().Where("id IN ?", serviceIDs).Delete(&Service{}).Error; err != nil {
			return fmt.Errorf("delete effective services: %w", err)
		}
	}
	if len(fieldIDs) > 0 {
		if err := fresh().Where("id IN ?", fieldIDs).Delete(&Field{}).Error; err != nil {
			return fmt.Errorf("delete effective fields: %w", err)
		}
	}
	if err := fresh().Where("id = ?", modelID).Delete(&Model{}).Error; err != nil {
		return fmt.Errorf("delete effective model: %w", err)
	}
	return nil
}

// PersistEffectiveProjection writes one E2-merged model as the effective tree.
// Exported for migrate helpers and metaeff.
func PersistEffectiveProjection(db *gorm.DB, merged *Model, effectiveID string) error {
	return persistEffectiveProjection(db, merged, effectiveID)
}

// PersistModelTreeAsRaw copies a declaration Model tree into meta_raw_*.
func PersistModelTreeAsRaw(db *gorm.DB, src *Model) error {
	return copyModelTreeToRaw(db, src)
}

// DeleteRawModelsForModule hard-deletes all raw catalog rows for a module id.
// IDs are plucked first so SQLite does not reject DELETE … WHERE id IN (SELECT … same table).
func DeleteRawModelsForModule(db *gorm.DB, moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil
	}
	root := db
	fresh := func() *gorm.DB { return root.Session(&gorm.Session{NewDB: true}).Unscoped() }

	var modelIDs []string
	if err := fresh().Model(&RawModel{}).Where("module_id = ?", moduleID).Pluck("id", &modelIDs).Error; err != nil {
		return fmt.Errorf("load raw models: %w", err)
	}
	if len(modelIDs) == 0 {
		return nil
	}

	var serviceIDs []string
	if err := fresh().Model(&RawService{}).Where("model_id IN ?", modelIDs).Pluck("id", &serviceIDs).Error; err != nil {
		return fmt.Errorf("load raw services: %w", err)
	}
	var fieldIDs []string
	if err := fresh().Model(&RawField{}).Where("model_id IN ?", modelIDs).Pluck("id", &fieldIDs).Error; err != nil {
		return fmt.Errorf("load raw fields: %w", err)
	}

	decoratorQ := fresh().Model(&RawDecorator{}).Where("model_id IN ?", modelIDs)
	if len(serviceIDs) > 0 {
		decoratorQ = decoratorQ.Or("service_id IN ?", serviceIDs)
	}
	if len(fieldIDs) > 0 {
		decoratorQ = decoratorQ.Or("field_id IN ?", fieldIDs)
	}
	var decoratorIDs []string
	if err := decoratorQ.Pluck("id", &decoratorIDs).Error; err != nil {
		return fmt.Errorf("load raw decorators: %w", err)
	}

	if len(decoratorIDs) > 0 {
		if err := fresh().Where("decorator_id IN ?", decoratorIDs).Delete(&RawArgument{}).Error; err != nil {
			return fmt.Errorf("delete raw arguments: %w", err)
		}
		if err := fresh().Where("id IN ?", decoratorIDs).Delete(&RawDecorator{}).Error; err != nil {
			return fmt.Errorf("delete raw decorators: %w", err)
		}
	}
	if len(serviceIDs) > 0 {
		if err := fresh().Where("service_id IN ?", serviceIDs).Delete(&RawTypeParameter{}).Error; err != nil {
			return fmt.Errorf("delete raw type parameters: %w", err)
		}
		if err := fresh().Where("service_id IN ?", serviceIDs).Delete(&RawParameter{}).Error; err != nil {
			return fmt.Errorf("delete raw parameters: %w", err)
		}
		if err := fresh().Where("id IN ?", serviceIDs).Delete(&RawService{}).Error; err != nil {
			return fmt.Errorf("delete raw services: %w", err)
		}
	}
	if len(fieldIDs) > 0 {
		if err := fresh().Where("id IN ?", fieldIDs).Delete(&RawField{}).Error; err != nil {
			return fmt.Errorf("delete raw fields: %w", err)
		}
	}
	if err := fresh().Where("id IN ?", modelIDs).Delete(&RawModel{}).Error; err != nil {
		return fmt.Errorf("delete raw models: %w", err)
	}
	return nil
}
