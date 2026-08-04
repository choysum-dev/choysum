// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Test hooks (production defaults). Override in *_test.go to force error paths.
var (
	lockLogicalKeyFn                     = lockLogicalKey
	expandModelsAlongExtendsFn           = ExpandModelsAlongExtends
	mergeSameNameModelsByExtensionChainFn = MergeSameNameModelsByExtensionChain
	deleteWhereFn                        = func(db *gorm.DB, value interface{}, query interface{}, args ...interface{}) error {
		return db.Where(query, args...).Delete(value).Error
	}
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
// Delete+persist runs in one transaction so a mid-rebuild failure cannot leave a hole.
func RecomputeEffective(db *gorm.DB, application, name string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	key := LogicalKey{Application: application, Name: name}.Normalized()
	if !key.Valid() {
		return fmt.Errorf("recompute requires non-empty application and name")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return recomputeEffectiveTx(tx, key)
	})
}

func recomputeEffectiveTx(tx *gorm.DB, key LogicalKey) error {
	// Serialize concurrent recomputes for the same logical name (best-effort on SQLite).
	if err := lockLogicalKeyFn(tx, key); err != nil {
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
	if err := expandModelsAlongExtendsFn(tx, models); err != nil {
		return fmt.Errorf("expand extends for %s/%s: %w", key.Application, key.Name, err)
	}
	merged, err := mergeSameNameModelsByExtensionChainFn(models)
	if err != nil {
		return fmt.Errorf("E2 merge %s/%s: %w", key.Application, key.Name, err)
	}
	if merged == nil {
		return fmt.Errorf("E2 merge %s/%s returned nil", key.Application, key.Name)
	}

	effID := ""
	var tip Model
	if len(existing) > 0 {
		// Prefer tip-like id: newest UpdatedAt then Id (stable across sibling IMD leftovers).
		tip = existing[0]
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

	// Capture tip service ids before subtree delete so Auth ACL (meta_service_id) stays valid.
	reuseServiceIDs, err := loadEffectiveServiceIDsByName(tx, tip.Id.String)
	if err != nil {
		return err
	}

	for _, row := range existing {
		if row.Id.Valid {
			if err := DeleteEffectiveModelTree(tx, row.Id.String); err != nil {
				return err
			}
		}
	}

	if err := persistEffectiveProjection(tx, merged, effID, reuseServiceIDs); err != nil {
		return fmt.Errorf("persist effective %s/%s: %w", key.Application, key.Name, err)
	}
	return nil
}

func loadEffectiveServiceIDsByName(tx *gorm.DB, modelID string) (map[string]string, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, nil
	}
	var services []Service
	if err := tx.Select("id", "name").Where("model_id = ?", modelID).Find(&services).Error; err != nil {
		return nil, fmt.Errorf("load prior effective services: %w", err)
	}
	out := make(map[string]string, len(services))
	for _, s := range services {
		name := strings.TrimSpace(s.Name)
		if name == "" || !s.Id.Valid || strings.TrimSpace(s.Id.String) == "" {
			continue
		}
		out[name] = s.Id.String
	}
	return out, nil
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
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("application = ? AND name = ?", key.Application, key.Name).
			Take(&m).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("lock effective row for %s/%s: %w", key.Application, key.Name, err)
		}
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
		if err := deleteWhereFn(fresh(), &Argument{}, "decorator_id IN ?", decoratorIDs); err != nil {
			return fmt.Errorf("delete effective arguments: %w", err)
		}
		if err := deleteWhereFn(fresh(), &Decorator{}, "id IN ?", decoratorIDs); err != nil {
			return fmt.Errorf("delete effective decorators: %w", err)
		}
	}
	if len(serviceIDs) > 0 {
		if err := deleteWhereFn(fresh(), &TypeParameter{}, "service_id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete effective type parameters: %w", err)
		}
		if err := deleteWhereFn(fresh(), &Parameter{}, "service_id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete effective parameters: %w", err)
		}
		if err := deleteWhereFn(fresh(), &Service{}, "id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete effective services: %w", err)
		}
	}
	if len(fieldIDs) > 0 {
		if err := deleteWhereFn(fresh(), &Field{}, "id IN ?", fieldIDs); err != nil {
			return fmt.Errorf("delete effective fields: %w", err)
		}
	}
	if err := deleteWhereFn(fresh(), &Model{}, "id = ?", modelID); err != nil {
		return fmt.Errorf("delete effective model: %w", err)
	}
	return nil
}

// PersistEffectiveProjection writes one E2-merged model as the effective tree.
// Exported for migrate helpers and metaeff. Service ids are minted fresh unless
// callers use persistEffectiveProjection with a reuse map (RecomputeEffective).
func PersistEffectiveProjection(db *gorm.DB, merged *Model, effectiveID string) error {
	return persistEffectiveProjection(db, merged, effectiveID, nil)
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
		if err := deleteWhereFn(fresh(), &RawArgument{}, "decorator_id IN ?", decoratorIDs); err != nil {
			return fmt.Errorf("delete raw arguments: %w", err)
		}
		if err := deleteWhereFn(fresh(), &RawDecorator{}, "id IN ?", decoratorIDs); err != nil {
			return fmt.Errorf("delete raw decorators: %w", err)
		}
	}
	if len(serviceIDs) > 0 {
		if err := deleteWhereFn(fresh(), &RawTypeParameter{}, "service_id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete raw type parameters: %w", err)
		}
		if err := deleteWhereFn(fresh(), &RawParameter{}, "service_id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete raw parameters: %w", err)
		}
		if err := deleteWhereFn(fresh(), &RawService{}, "id IN ?", serviceIDs); err != nil {
			return fmt.Errorf("delete raw services: %w", err)
		}
	}
	if len(fieldIDs) > 0 {
		if err := deleteWhereFn(fresh(), &RawField{}, "id IN ?", fieldIDs); err != nil {
			return fmt.Errorf("delete raw fields: %w", err)
		}
	}
	if err := deleteWhereFn(fresh(), &RawModel{}, "id IN ?", modelIDs); err != nil {
		return fmt.Errorf("delete raw models: %w", err)
	}
	return nil
}
