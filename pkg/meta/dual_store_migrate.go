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

const (
	effectiveAppNameUniqueIndex     = "uidx_meta_model_app_name_alive"
	effectiveAppNameUniqueIndexTemp = effectiveAppNameUniqueIndex + "_new"
)

// execDDL runs DDL; overridden in tests for failure injection.
var execDDL = func(db *gorm.DB, sql string) error {
	return db.Exec(sql).Error
}

// Test hooks (production defaults). Override in *_test.go to force error paths.
var (
	countMetaRawModels = func(db *gorm.DB) (int64, error) {
		var n int64
		err := db.Unscoped().Model(&RawModel{}).Count(&n).Error
		return n, err
	}
	loadIMDModelsForMigrate = func(tx *gorm.DB) ([]*Model, error) {
		var sources []*Model
		err := tx.
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
			Find(&sources).Error
		return sources, err
	}
	loadRawModelsForRecompute = func(tx *gorm.DB) ([]*RawModel, error) {
		var raws []*RawModel
		err := tx.
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
			Find(&raws).Error
		return raws, err
	}
	clearEffectiveShapeTreesFn   = clearEffectiveShapeTrees
	persistEffectiveProjectionFn = func(db *gorm.DB, merged *Model, effectiveID string) error {
		return persistEffectiveProjection(db, merged, effectiveID, nil)
	}
)

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

// MigrateIMDCatalogToDualStore copies today's live meta_model* IMD rows into meta_raw_*,
// then rebuilds meta_model* as E2 effective projections (one row per application+name).
//
// Intended for wipe/test and one-shot upgrades from legacy IMD catalogs.
// After EDS-2, Persist writes meta_raw_* and recomputes effective projections.
//
// Soft-deleted meta_model rows are skipped (default GORM scope). Materialized parent
// copies on source rows are preserved as-is (EDS-2 will stop writing them). Effective
// ids reuse the tip id (newest created_at, then id) when possible. An empty live
// catalog is a no-op (does not hard-delete existing effective rows).
//
// DDL (EnsureDualStoreTables + effective unique index) runs outside the transaction;
// copy + recompute run inside one transaction so failures roll back partial raw/effective
// writes. MySQL DDL would implicitly commit if run inside the transaction.
func MigrateIMDCatalogToDualStore(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := EnsureDualStoreTables(db); err != nil {
		return err
	}

	rawCount, err := countMetaRawModels(db)
	if err != nil {
		return fmt.Errorf("count meta_raw_model: %w", err)
	}
	if rawCount > 0 {
		return fmt.Errorf("meta_raw_model already has %d rows (incl. soft-deleted); refuse migrate (wipe raw first)", rawCount)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		sources, err := loadIMDModelsForMigrate(tx)
		if err != nil {
			return fmt.Errorf("load meta_model for dual-store migrate: %w", err)
		}

		live := 0
		for _, src := range sources {
			if src != nil {
				live++
			}
		}
		// Empty live catalog: do not call recompute (it hard-deletes effective trees).
		if live == 0 {
			return nil
		}
		return migrateIMDSources(tx, sources)
	}); err != nil {
		return err
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		return err
	}
	return RemapACLToEffectiveModelIDs(db)
}

// migrateIMDSources copies live IMD sources into meta_raw_* then rebuilds effective rows.
func migrateIMDSources(tx *gorm.DB, sources []*Model) error {
	seenModulePath := map[string]string{}
	for _, src := range sources {
		if src == nil {
			continue
		}
		if !src.ModuleId.Valid || strings.TrimSpace(src.ModuleId.String) == "" {
			return fmt.Errorf("meta_model %s (%s/%s) missing module_id; required for meta_raw_model uniqueness", src.Id.String, src.Application, src.Name)
		}
		key := src.ModuleId.String + "\x00" + src.Path
		if prev, ok := seenModulePath[key]; ok {
			return fmt.Errorf("duplicate live (module_id, path) %q / %q (models %s and %s)", src.ModuleId.String, src.Path, prev, src.Id.String)
		}
		seenModulePath[key] = src.Id.String
	}

	for _, src := range sources {
		if src == nil {
			continue
		}
		if err := copyModelTreeToRaw(tx, src); err != nil {
			return err
		}
	}

	return recomputeAllEffectiveFromRawTx(tx)
}

// RecomputeAllEffectiveFromRaw hard-deletes all effective meta_model* shape rows, then
// rebuilds them from live meta_raw_* via E2. DML runs in a transaction; the effective
// unique-index DDL runs after commit. Callers with only a partial raw catalog will lose
// effective rows that have no raw counterpart.
func RecomputeAllEffectiveFromRaw(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return recomputeAllEffectiveFromRawTx(tx)
	}); err != nil {
		return err
	}
	if err := EnsureEffectiveAppNameUniqueIndex(db); err != nil {
		return err
	}
	return RemapACLToEffectiveModelIDs(db)
}

// EnsureEffectiveAppNameUniqueIndex enforces one live effective row per (application, name).
// Safe to call repeatedly after recomputes; call only when live duplicates are gone.
func EnsureEffectiveAppNameUniqueIndex(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return ensureEffectiveAppNameUniqueIndex(db)
}

func recomputeAllEffectiveFromRawTx(tx *gorm.DB) error {
	raws, err := loadRawModelsForRecompute(tx)
	if err != nil {
		return fmt.Errorf("load meta_raw_model: %w", err)
	}

	groups := map[string][]*RawModel{}
	tipIDs := map[string]string{}
	for _, raw := range raws {
		key := logicalModelKey(raw.Application, raw.Name)
		groups[key] = append(groups[key], raw)
		prev := tipIDs[key]
		if prev == "" || rawIsNewerTip(raw, findRawByID(groups[key], prev)) {
			tipIDs[key] = raw.Id.String
		}
	}

	// Hard-delete (truncate) existing effective shape trees before rewrite.
	if err := clearEffectiveShapeTreesFn(tx); err != nil {
		return err
	}

	for key, group := range groups {
		merged, err := MergeEffectiveModel(group)
		if err != nil {
			return fmt.Errorf("E2 merge %s: %w", key, err)
		}
		effID := tipIDs[key]
		if effID == "" {
			effID = xid.New().String()
		}
		if err := persistEffectiveProjectionFn(tx, merged, effID); err != nil {
			return fmt.Errorf("persist effective %s: %w", key, err)
		}
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

func ensureBaseModelID(b *BaseModel) {
	if b == nil {
		return
	}
	if strings.TrimSpace(b.Id.String) == "" {
		b.Id = sql.NullString{String: xid.New().String(), Valid: true}
	}
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
	ensureBaseModelID(&raw.BaseModel)
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(raw).Error; err != nil {
		return fmt.Errorf("create meta_raw_model %s: %w", raw.Id.String, err)
	}

	for _, f := range src.Fields {
		if f == nil {
			continue
		}
		rf := rawFieldFromField(f, raw.Id)
		ensureBaseModelID(&rf.BaseModel)
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rf).Error; err != nil {
			return fmt.Errorf("create meta_raw_field %s: %w", rf.Id.String, err)
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
		ensureBaseModelID(&rs.BaseModel)
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rs).Error; err != nil {
			return fmt.Errorf("create meta_raw_service %s: %w", rs.Id.String, err)
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
			ensureBaseModelID(&rp.BaseModel)
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
			ensureBaseModelID(&rtp.BaseModel)
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
	ensureBaseModelID(&rd.BaseModel)
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(rd).Error; err != nil {
		return fmt.Errorf("create meta_raw_decorator %s: %w", rd.Id.String, err)
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
		ensureBaseModelID(&ra.BaseModel)
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(ra).Error; err != nil {
			return fmt.Errorf("create meta_raw_argument %s: %w", ra.Id.String, err)
		}
	}
	return nil
}

func clearEffectiveShapeTrees(db *gorm.DB) error {
	// Hard-delete order: arguments → decorators → parameters/type_parameters → fields/services → models.
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

// persistEffectiveProjection inserts one effective model tree.
// reuseServiceIDs maps service name → prior effective service id. Callers may pass a
// non-nil map only after deleting existing effective rows for this logical key
// (RecomputeEffective does this via DeleteEffectiveModelTree); otherwise reused ids
// collide with still-present primary keys.
func persistEffectiveProjection(db *gorm.DB, merged *Model, effectiveID string, reuseServiceIDs map[string]string) error {
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

	for _, d := range merged.Decorators {
		if err := persistDecoratorTree(db, d, eff.Id, sql.NullString{}, sql.NullString{}); err != nil {
			return err
		}
	}

	for _, f := range merged.Fields {
		if f == nil {
			continue
		}
		decs := f.Decorators
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
		for _, d := range decs {
			if err := persistDecoratorTree(db, d, sql.NullString{}, nf.Id, sql.NullString{}); err != nil {
				return err
			}
		}
	}
	usedServiceIDs := map[string]bool{}
	for _, s := range merged.Services {
		if s == nil {
			continue
		}
		decs := s.Decorators
		ns := *s
		svcID := xid.New().String()
		if reuseServiceIDs != nil {
			if prev := strings.TrimSpace(reuseServiceIDs[strings.TrimSpace(s.Name)]); prev != "" && !usedServiceIDs[prev] {
				svcID = prev
			}
		}
		usedServiceIDs[svcID] = true
		ns.Id = sql.NullString{String: svcID, Valid: true}
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
			if p == nil || p.Name == "this" {
				// Synthetic receiver; codegen / Auth omit it from effective service shape.
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
		for _, d := range decs {
			if err := persistDecoratorTree(db, d, sql.NullString{}, sql.NullString{}, ns.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func persistDecoratorTree(db *gorm.DB, d *Decorator, modelID, fieldID, serviceID sql.NullString) error {
	if d == nil {
		return nil
	}
	nd := &Decorator{
		BaseModel:      BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:           d.Name,
		ModuleSpecPath: d.ModuleSpecPath,
		ReferenceIdent: d.ReferenceIdent,
		ModelId:        modelID,
		FieldId:        fieldID,
		ServiceId:      serviceID,
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(nd).Error; err != nil {
		return fmt.Errorf("create meta_decorator: %w", err)
	}
	for _, a := range d.Arguments {
		if a == nil {
			continue
		}
		na := &Argument{
			BaseModel:      BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Type:           a.Type,
			Value:          a.Value,
			ReferenceIdent: a.ReferenceIdent,
			ModuleSpecPath: a.ModuleSpecPath,
			DecoratorId:    nd.Id,
		}
		if err := db.Session(&gorm.Session{SkipHooks: true}).Create(na).Error; err != nil {
			return fmt.Errorf("create meta_argument: %w", err)
		}
	}
	return nil
}

func ensureEffectiveAppNameUniqueIndex(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "sqlite", "postgres":
		return ensurePartialAliveAppNameUniqueIndex(db)
	default:
		return ensureFullAppNameUniqueIndex(db)
	}
}

// ensurePartialAliveAppNameUniqueIndex creates the live-only unique index without a
// drop-before-create window: temp index first, then replace the final name.
func ensurePartialAliveAppNameUniqueIndex(db *gorm.DB) error {
	createPartial := func(name string) string {
		return fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON meta_model (application, name) WHERE deleted_at IS NULL",
			name,
		)
	}
	if err := execDDL(db, createPartial(effectiveAppNameUniqueIndexTemp)); err != nil {
		return fmt.Errorf("create unique index %s: %w", effectiveAppNameUniqueIndexTemp, err)
	}
	if err := execDDL(db, fmt.Sprintf("DROP INDEX IF EXISTS %s", effectiveAppNameUniqueIndex)); err != nil {
		return fmt.Errorf("drop unique index %s: %w", effectiveAppNameUniqueIndex, err)
	}
	if err := execDDL(db, createPartial(effectiveAppNameUniqueIndex)); err != nil {
		// Temp index still enforces live uniqueness.
		return fmt.Errorf("create unique index %s: %w", effectiveAppNameUniqueIndex, err)
	}
	if err := execDDL(db, fmt.Sprintf("DROP INDEX IF EXISTS %s", effectiveAppNameUniqueIndexTemp)); err != nil {
		return fmt.Errorf("drop unique index %s: %w", effectiveAppNameUniqueIndexTemp, err)
	}
	return nil
}

// ensureFullAppNameUniqueIndex is the MySQL path (no partial indexes).
func ensureFullAppNameUniqueIndex(db *gorm.DB) error {
	createFull := func(name string) string {
		return fmt.Sprintf("CREATE UNIQUE INDEX %s ON meta_model (application, name)", name)
	}
	if !db.Migrator().HasIndex("meta_model", effectiveAppNameUniqueIndexTemp) {
		if err := execDDL(db, createFull(effectiveAppNameUniqueIndexTemp)); err != nil {
			return fmt.Errorf("create unique index %s: %w", effectiveAppNameUniqueIndexTemp, err)
		}
	}
	if db.Migrator().HasIndex("meta_model", effectiveAppNameUniqueIndex) {
		if err := db.Migrator().DropIndex("meta_model", effectiveAppNameUniqueIndex); err != nil {
			return fmt.Errorf("drop unique index %s: %w", effectiveAppNameUniqueIndex, err)
		}
	}
	if !db.Migrator().HasIndex("meta_model", effectiveAppNameUniqueIndex) {
		if err := execDDL(db, createFull(effectiveAppNameUniqueIndex)); err != nil {
			// Temp index still enforces uniqueness.
			return fmt.Errorf("create unique index %s: %w", effectiveAppNameUniqueIndex, err)
		}
	}
	if db.Migrator().HasIndex("meta_model", effectiveAppNameUniqueIndexTemp) {
		if err := db.Migrator().DropIndex("meta_model", effectiveAppNameUniqueIndexTemp); err != nil {
			return fmt.Errorf("drop unique index %s: %w", effectiveAppNameUniqueIndexTemp, err)
		}
	}
	return nil
}
