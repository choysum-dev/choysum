// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"gorm.io/gorm"
)

const effectiveAppNameUniqueIndex = "uidx_meta_model_app_name_alive"

// EnsureDualStoreTables creates raw + effective catalog tables (idempotent).
func EnsureDualStoreTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	ents := append(DualStoreEffectiveEntities(), DualStoreRawEntities()...)
	if err := db.AutoMigrate(ents...); err != nil {
		return fmt.Errorf("ensure dual-store catalog tables: %w", err)
	}
	return nil
}

// MigrateIMDCatalogToDualStore copies today's meta_model* IMD rows into meta_raw_*,
// then rebuilds meta_model* as E2 effective projections (one row per application+name).
//
// Intended for wipe/test and one-shot upgrades. Production Persist still writes IMD
// meta_model until EDS-2; do not run this on a live DB that will keep writing IMD
// without switching Persist to raw.
//
// Materialized parent copies on source rows are preserved as-is (EDS-2 will stop
// writing them). Effective ids reuse the tip id (newest created_at, then id) when possible.
func MigrateIMDCatalogToDualStore(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := EnsureDualStoreTables(db); err != nil {
		return err
	}

	var rawCount int64
	if err := db.Model(&RawModel{}).Count(&rawCount).Error; err != nil {
		return fmt.Errorf("count meta_raw_model: %w", err)
	}
	if rawCount > 0 {
		return fmt.Errorf("meta_raw_model already has %d rows; refuse migrate (wipe raw first)", rawCount)
	}

	var sources []*Model
	if err := db.
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
		Find(&sources).Error; err != nil {
		return fmt.Errorf("load meta_model for dual-store migrate: %w", err)
	}

	for _, src := range sources {
		if src == nil {
			continue
		}
		if err := copyModelTreeToRaw(db, src); err != nil {
			return err
		}
	}

	return RecomputeAllEffectiveFromRaw(db)
}

// RecomputeAllEffectiveFromRaw replaces effective meta_model* content from all live raw rows.
func RecomputeAllEffectiveFromRaw(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	var raws []*RawModel
	if err := db.
		Preload("Fields").
		Preload("Services").
		Preload("Services.Parameters").
		Preload("Services.TypeParameters").
		Find(&raws).Error; err != nil {
		return fmt.Errorf("load meta_raw_model: %w", err)
	}

	groups := map[string][]*RawModel{}
	tipIDs := map[string]string{}
	for _, raw := range raws {
		if raw == nil {
			continue
		}
		key := logicalModelKey(raw.Application, raw.Name)
		groups[key] = append(groups[key], raw)
		prev := tipIDs[key]
		if prev == "" || rawIsNewerTip(raw, findRawByID(groups[key], prev)) {
			tipIDs[key] = raw.Id.String
		}
	}

	// Soft-delete / clear existing effective shape trees before rewrite.
	if err := clearEffectiveShapeTrees(db); err != nil {
		return err
	}

	for key, group := range groups {
		merged, err := MergeEffectiveModel(group)
		if err != nil {
			return fmt.Errorf("E2 merge %s: %w", key, err)
		}
		if merged == nil {
			continue
		}
		effID := tipIDs[key]
		if effID == "" {
			effID = xid.New().String()
		}
		if err := persistEffectiveProjection(db, merged, effID); err != nil {
			return fmt.Errorf("persist effective %s: %w", key, err)
		}
	}

	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		return err
	}
	return nil
}

func logicalModelKey(application, name string) string {
	return strings.TrimSpace(application) + "\x00" + strings.TrimSpace(name)
}

func findRawByID(raws []*RawModel, id string) *RawModel {
	for _, r := range raws {
		if r != nil && r.Id.String == id {
			return r
		}
	}
	return nil
}

func rawIsNewerTip(candidate, previous *RawModel) bool {
	if previous == nil {
		return true
	}
	if candidate.CreatedAt.After(previous.CreatedAt) {
		return true
	}
	if candidate.CreatedAt.Equal(previous.CreatedAt) && candidate.Id.String > previous.Id.String {
		return true
	}
	return false
}

func copyModelTreeToRaw(db *gorm.DB, src *Model) error {
	raw := &RawModel{
		BaseModel:    BaseModel{Id: src.Id, CreatedAt: src.CreatedAt, UpdatedAt: src.UpdatedAt},
		Name:         src.Name,
		Path:         src.Path,
		Application:  src.Application,
		ClassName:    src.ClassName,
		ModelTable:   src.ModelTable,
		Abstract:     src.Abstract,
		AutoMigrate:  src.AutoMigrate,
		Readonly:     src.Readonly,
		RawExtends:   src.RawExtends,
		Extends:      src.Extends,
		CompanyField: src.CompanyField,
		ModuleId:     src.ModuleId,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		return fmt.Errorf("create meta_raw_model %s: %w", src.Id.String, err)
	}

	for _, f := range src.Fields {
		if f == nil {
			continue
		}
		rf := rawFieldFromField(f, raw.Id)
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rf).Error; err != nil {
			return fmt.Errorf("create meta_raw_field %s: %w", f.Id.String, err)
		}
		for _, d := range f.Decorators {
			if err := copyDecoratorToRaw(db, d, sql.NullString{}, sql.NullString{String: rf.Id.String, Valid: true}, sql.NullString{}); err != nil {
				return err
			}
		}
	}

	for _, s := range src.Services {
		if s == nil {
			continue
		}
		rs := rawServiceFromService(s, raw.Id)
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rs).Error; err != nil {
			return fmt.Errorf("create meta_raw_service %s: %w", s.Id.String, err)
		}
		for _, p := range s.Parameters {
			if p == nil {
				continue
			}
			rp := &RawParameter{
				BaseModel:        BaseModel{Id: p.Id, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt},
				Name:             p.Name,
				TsTypeAnnotation: p.TsTypeAnnotation,
				ProtobufType:     p.ProtobufType,
				ServiceId:        sql.NullString{String: rs.Id.String, Valid: true},
			}
			if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rp).Error; err != nil {
				return err
			}
		}
		for _, tp := range s.TypeParameters {
			if tp == nil {
				continue
			}
			rtp := &RawTypeParameter{
				BaseModel:      BaseModel{Id: tp.Id, CreatedAt: tp.CreatedAt, UpdatedAt: tp.UpdatedAt},
				Name:           tp.Name,
				ModuleSpecPath: tp.ModuleSpecPath,
				ReferenceIdent: tp.ReferenceIdent,
				ServiceId:      sql.NullString{String: rs.Id.String, Valid: true},
			}
			if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rtp).Error; err != nil {
				return err
			}
		}
		for _, d := range s.Decorators {
			if err := copyDecoratorToRaw(db, d, sql.NullString{}, sql.NullString{}, sql.NullString{String: rs.Id.String, Valid: true}); err != nil {
				return err
			}
		}
	}

	for _, d := range src.Decorators {
		if err := copyDecoratorToRaw(db, d, sql.NullString{String: raw.Id.String, Valid: true}, sql.NullString{}, sql.NullString{}); err != nil {
			return err
		}
	}
	return nil
}

func rawFieldFromField(f *Field, modelID sql.NullString) *RawField {
	return &RawField{
		BaseModel:                BaseModel{Id: f.Id, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt},
		Name:                     f.Name,
		TsTypeAnnotation:         f.TsTypeAnnotation,
		TsTypeReference:          f.TsTypeReference,
		OriginModelPath:          f.OriginModelPath,
		FieldType:                f.FieldType,
		Relation:                 f.Relation,
		RelationModel:            f.RelationModel,
		RelationFilter:           f.RelationFilter,
		RelationInverseField:     f.RelationInverseField,
		RelationJoinModel:        f.RelationJoinModel,
		RelationJoinField:        f.RelationJoinField,
		RelationInverseJoinField: f.RelationInverseJoinField,
		RelationModelParentField: f.RelationModelParentField,
		Selection:                f.Selection,
		SelectionKind:            f.SelectionKind,
		SelectionMethod:          f.SelectionMethod,
		FieldString:              f.FieldString,
		StringText:               f.StringText,
		FieldHelp:                f.FieldHelp,
		HelpText:                 f.HelpText,
		ReferenceIdent:           f.ReferenceIdent,
		ModuleSpecPath:           f.ModuleSpecPath,
		AccessibilityModifier:    f.AccessibilityModifier,
		IsStatic:                 f.IsStatic,
		IsReadonly:               f.IsReadonly,
		MaxUploadBytes:           f.MaxUploadBytes,
		MaxWidth:                 f.MaxWidth,
		MaxHeight:                f.MaxHeight,
		Indexed:                  f.Indexed,
		NotNull:                  f.NotNull,
		Size:                     f.Size,
		Precision:                f.Precision,
		Scale:                    f.Scale,
		ScaleField:               f.ScaleField,
		CurrencyField:            f.CurrencyField,
		Round:                    f.Round,
		ResolvedSpec:             f.ResolvedSpec,
		ModelId:                  modelID,
	}
}

func rawServiceFromService(s *Service, modelID sql.NullString) *RawService {
	return &RawService{
		BaseModel:             BaseModel{Id: s.Id, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt},
		Name:                  s.Name,
		OriginModelPath:       s.OriginModelPath,
		AccessibilityModifier: s.AccessibilityModifier,
		TsTypeAnnotation:      s.TsTypeAnnotation,
		ProtobufType:          s.ProtobufType,
		IsStatic:              s.IsStatic,
		ModelId:               modelID,
	}
}

func copyDecoratorToRaw(db *gorm.DB, d *Decorator, modelID, fieldID, serviceID sql.NullString) error {
	if d == nil {
		return nil
	}
	rd := &RawDecorator{
		BaseModel:      BaseModel{Id: d.Id, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt},
		Name:           d.Name,
		ModuleSpecPath: d.ModuleSpecPath,
		ReferenceIdent: d.ReferenceIdent,
		ModelId:        modelID,
		FieldId:        fieldID,
		ServiceId:      serviceID,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rd).Error; err != nil {
		return fmt.Errorf("create meta_raw_decorator %s: %w", d.Id.String, err)
	}
	for _, a := range d.Arguments {
		if a == nil {
			continue
		}
		ra := &RawArgument{
			BaseModel:      BaseModel{Id: a.Id, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt},
			Type:           a.Type,
			Value:          a.Value,
			ReferenceIdent: a.ReferenceIdent,
			ModuleSpecPath: a.ModuleSpecPath,
			DecoratorId:    sql.NullString{String: rd.Id.String, Valid: true},
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(ra).Error; err != nil {
			return fmt.Errorf("create meta_raw_argument %s: %w", a.Id.String, err)
		}
	}
	return nil
}

func clearEffectiveShapeTrees(db *gorm.DB) error {
	// Order: arguments → decorators → parameters/type_parameters → fields/services → models.
	for _, step := range []struct {
		name string
		fn   func() error
	}{
		{"meta_argument", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Argument{}).Error
		}},
		{"meta_decorator", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Decorator{}).Error
		}},
		{"meta_parameter", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Parameter{}).Error
		}},
		{"meta_type_parameter", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&TypeParameter{}).Error
		}},
		{"meta_field", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Field{}).Error
		}},
		{"meta_service", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Service{}).Error
		}},
		{"meta_model", func() error {
			return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Where("1 = 1").Delete(&Model{}).Error
		}},
	} {
		if err := step.fn(); err != nil {
			return fmt.Errorf("clear %s: %w", step.name, err)
		}
	}
	return nil
}

func persistEffectiveProjection(db *gorm.DB, merged *Model, effectiveID string) error {
	eff := &Model{
		BaseModel: BaseModel{
			Id:        sql.NullString{String: effectiveID, Valid: true},
			CreatedAt: merged.CreatedAt,
			UpdatedAt: merged.UpdatedAt,
		},
		Name:         merged.Name,
		Path:         merged.Path,
		Application:  merged.Application,
		ClassName:    merged.ClassName,
		ModelTable:   merged.ModelTable,
		Abstract:     merged.Abstract,
		AutoMigrate:  merged.AutoMigrate,
		Readonly:     merged.Readonly,
		RawExtends:   merged.RawExtends,
		Extends:      merged.Extends,
		CompanyField: merged.CompanyField,
		// No single ModuleId on effective projection.
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(eff).Error; err != nil {
		return err
	}

	for _, f := range merged.Fields {
		if f == nil {
			continue
		}
		nf := *f
		nf.Id = sql.NullString{String: xid.New().String(), Valid: true}
		nf.ModelId = eff.Id
		nf.Model = nil
		nf.Decorators = nil
		if nf.OriginModelPath == "" {
			nf.OriginModelPath = merged.Path
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&nf).Error; err != nil {
			return err
		}
	}
	for _, s := range merged.Services {
		if s == nil {
			continue
		}
		ns := *s
		ns.Id = sql.NullString{String: xid.New().String(), Valid: true}
		ns.ModelId = eff.Id
		ns.Model = nil
		ns.Decorators = nil
		params := ns.Parameters
		tps := ns.TypeParameters
		ns.Parameters = nil
		ns.TypeParameters = nil
		if ns.OriginModelPath == "" {
			ns.OriginModelPath = merged.Path
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&ns).Error; err != nil {
			return err
		}
		for _, p := range params {
			if p == nil {
				continue
			}
			np := *p
			np.Id = sql.NullString{String: xid.New().String(), Valid: true}
			np.ServiceId = ns.Id
			np.Service = nil
			if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&np).Error; err != nil {
				return err
			}
		}
		for _, tp := range tps {
			if tp == nil {
				continue
			}
			ntp := *tp
			ntp.Id = sql.NullString{String: xid.New().String(), Valid: true}
			ntp.ServiceId = ns.Id
			ntp.Service = nil
			if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&ntp).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureEffectiveAppNameUniqueIndex(db *gorm.DB) error {
	// Soft-deleted rows still occupy the unique key on SQLite; wipe/reinstall is supported.
	sql := fmt.Sprintf(
		"CREATE UNIQUE INDEX IF NOT EXISTS %s ON meta_model (application, name)",
		effectiveAppNameUniqueIndex,
	)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("create unique index %s: %w", effectiveAppNameUniqueIndex, err)
	}
	return nil
}
