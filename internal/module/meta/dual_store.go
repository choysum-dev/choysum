// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
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

// ensureDualStoreTables creates raw + effective catalog tables (idempotent)
// and ensures the live (application, name) unique index on meta_model (SPL12).
func ensureDualStoreTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	ents := append([]any{
		&pkgmeta.Model{},
		&pkgmeta.Field{},
		&pkgmeta.Service{},
		&pkgmeta.TypeParameter{},
		&pkgmeta.Parameter{},
		&pkgmeta.Decorator{},
		&pkgmeta.Argument{},
	}, DualStoreRawEntities()...)
	if err := db.AutoMigrate(ents...); err != nil {
		return fmt.Errorf("ensure dual-store catalog tables: %w", err)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		return err
	}
	return nil
}

// EnsureEffectiveAppNameUniqueIndex enforces one live effective row per (application, name).
// Safe to call repeatedly after recomputes; call only when live duplicates are gone.
// Catalog AutoMigrate alone does not create this index — call after migrating CatalogEntities.
func EnsureEffectiveAppNameUniqueIndex(db *gorm.DB) error {
	return ensureEffectiveAppNameUniqueIndex(db)
}

func ensureEffectiveAppNameUniqueIndex(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	switch db.Dialector.Name() {
	case "sqlite", "postgres":
		return ensurePartialAliveAppNameUniqueIndex(db)
	default:
		return ensureFullAppNameUniqueIndex(db)
	}
}

func ensureBaseModelID(b *pkgmeta.BaseModel) {
	if b == nil {
		return
	}
	// Invalid NullString still writes NULL even when String is non-empty.
	if !b.Id.Valid || strings.TrimSpace(b.Id.String) == "" {
		b.Id = sql.NullString{String: xid.New().String(), Valid: true}
	}
}

func rawIsNewerTip(candidate, previous *rawModel) bool {
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

func copyModelTreeToRaw(db *gorm.DB, src *pkgmeta.Model) error {
	raw := &rawModel{
		BaseModel:    pkgmeta.BaseModel{Id: src.Id, CreatedAt: src.CreatedAt, UpdatedAt: src.UpdatedAt},
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
			rp := &rawParameter{
				BaseModel:        pkgmeta.BaseModel{Id: p.Id, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt},
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
			rtp := &rawTypeParameter{
				BaseModel:      pkgmeta.BaseModel{Id: tp.Id, CreatedAt: tp.CreatedAt, UpdatedAt: tp.UpdatedAt},
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

func rawFieldFromField(f *pkgmeta.Field, modelID sql.NullString) *rawField {
	return &rawField{
		BaseModel:                pkgmeta.BaseModel{Id: f.Id, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt},
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

func rawServiceFromService(s *pkgmeta.Service, modelID sql.NullString) *rawService {
	return &rawService{
		BaseModel:             pkgmeta.BaseModel{Id: s.Id, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt},
		Name:                  s.Name,
		OriginModelPath:       s.OriginModelPath,
		AccessibilityModifier: s.AccessibilityModifier,
		TsTypeAnnotation:      s.TsTypeAnnotation,
		ProtobufType:          s.ProtobufType,
		IsStatic:              s.IsStatic,
		ModelId:               modelID,
	}
}

func copyDecoratorToRaw(db *gorm.DB, d *pkgmeta.Decorator, modelID, fieldID, serviceID sql.NullString) error {
	if d == nil {
		return nil
	}
	rd := &rawDecorator{
		BaseModel:      pkgmeta.BaseModel{Id: d.Id, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt},
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
		ra := &rawArgument{
			BaseModel:      pkgmeta.BaseModel{Id: a.Id, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt},
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

// persistEffectiveProjection inserts one effective model tree.
// reuseServiceIDs maps service name → prior effective service id. Callers may pass a
// non-nil map only after deleting existing effective rows for this logical key
// (recomputeEffective does this via deleteEffectiveModelTree); otherwise reused ids
// collide with still-present primary keys.
func persistEffectiveProjection(db *gorm.DB, merged *pkgmeta.Model, effectiveID string, reuseServiceIDs map[string]string) error {
	eff := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{
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

func persistDecoratorTree(db *gorm.DB, d *pkgmeta.Decorator, modelID, fieldID, serviceID sql.NullString) error {
	if d == nil {
		return nil
	}
	nd := &pkgmeta.Decorator{
		BaseModel:      pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
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
		na := &pkgmeta.Argument{
			BaseModel:      pkgmeta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
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
