// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

type expandsShape struct {
	fields   []*pkgmeta.Field
	services []*pkgmeta.Service
}

// expandsShapeKey scopes cache / cycle / local lookups by application + path so
// same-path declarations from different applications cannot collide.
func expandsShapeKey(application, path string) string {
	return strings.TrimSpace(application) + "\x00" + strings.TrimSpace(path)
}

// ExpandModelsAlongExtends merges parent Fields and Services into each model in memory
// by walking Extends paths against meta_raw_* (and the provided local set).
//
// Used by:
//   - schema migrator (DDL columns for thin subclasses)
//   - recomputeEffective (effective shape must include pkgmeta.BaseModel methods/fields;
//     E2 same-name union alone does not pull differently-named parents)
//
// Does not write to the database.
func ExpandModelsAlongExtends(db *gorm.DB, models []*pkgmeta.Model) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	localByAppPath := make(map[string]*pkgmeta.Model, len(models))
	localByPath := make(map[string]*pkgmeta.Model, len(models))
	for _, m := range models {
		if m == nil || strings.TrimSpace(m.Path) == "" {
			continue
		}
		localByAppPath[expandsShapeKey(m.Application, m.Path)] = m
		localByPath[strings.TrimSpace(m.Path)] = m
	}
	cache := make(map[string]*expandsShape)
	visiting := make(map[string]bool)
	for _, m := range models {
		if m == nil {
			continue
		}
		shape, err := expandShapeAlongExtends(db, m, localByAppPath, localByPath, cache, visiting)
		if err != nil {
			return err
		}
		m.Fields = shape.fields
		m.Services = shape.services
	}
	return nil
}

func expandShapeAlongExtends(
	db *gorm.DB,
	model *pkgmeta.Model,
	localByAppPath map[string]*pkgmeta.Model,
	localByPath map[string]*pkgmeta.Model,
	cache map[string]*expandsShape,
	visiting map[string]bool,
) (*expandsShape, error) {
	if model == nil {
		return &expandsShape{}, nil
	}
	path := strings.TrimSpace(model.Path)
	key := ""
	if path != "" {
		key = expandsShapeKey(model.Application, path)
		if cached, ok := cache[key]; ok {
			return cached, nil
		}
		if visiting[key] {
			return nil, fmt.Errorf("circular extends while expanding model shape: %s", path)
		}
		visiting[key] = true
		defer delete(visiting, key)
	}

	var parentShape *expandsShape
	parentPath := ""
	if extends := strings.TrimSpace(model.Extends); extends != "" {
		parent, err := resolveExtendsModel(db, extends, localByAppPath, localByPath, strings.TrimSpace(model.Application))
		if err != nil {
			return nil, err
		}
		if parent != nil {
			parentPath = parent.Path
			parentShape, err = expandShapeAlongExtends(db, parent, localByAppPath, localByPath, cache, visiting)
			if err != nil {
				return nil, err
			}
		}
	}

	var parentFields []*pkgmeta.Field
	var parentServices []*pkgmeta.Service
	if parentShape != nil {
		parentFields = parentShape.fields
		parentServices = parentShape.services
	}

	fields, err := mergeFieldsForSchema(parentFields, model.Fields, parentPath, path)
	if err != nil {
		return nil, err
	}
	services := mergeServicesForSchema(parentServices, model.Services, parentPath, path)
	out := &expandsShape{fields: fields, services: services}
	if key != "" {
		cache[key] = out
	}
	return out, nil
}

func resolveExtendsModel(
	db *gorm.DB,
	extendsPath string,
	localByAppPath map[string]*pkgmeta.Model,
	localByPath map[string]*pkgmeta.Model,
	preferredApp string,
) (*pkgmeta.Model, error) {
	if preferredApp != "" {
		if local, ok := localByAppPath[expandsShapeKey(preferredApp, extendsPath)]; ok {
			return local, nil
		}
	}
	if local, ok := localByPath[extendsPath]; ok {
		return local, nil
	}
	raws, err := loadRawParentsByPath(db, extendsPath, preferredApp)
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 && preferredApp != "" {
		// Cross-application Extends are rare; fall back to path-only after same-app miss.
		raws, err = loadRawParentsByPath(db, extendsPath, "")
		if err != nil {
			return nil, err
		}
	}
	if len(raws) == 0 {
		return nil, nil
	}
	converted := rawModelsAsModels([]*rawModel{&raws[0]})
	return converted[0], nil
}

func loadRawParentsByPath(db *gorm.DB, extendsPath, application string) ([]rawModel, error) {
	q := db.
		Preload("Fields", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Fields.Decorators", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Fields.Decorators.Arguments", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Services", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Services.Decorators", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Services.Decorators.Arguments", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Services.Parameters", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Services.TypeParameters", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Where("path = ?", extendsPath)
	if application != "" {
		q = q.Where("application = ?", application)
	}
	var raws []rawModel
	if err := q.Order("id DESC").Find(&raws).Error; err != nil {
		return nil, fmt.Errorf("load raw parent by path %s: %w", extendsPath, err)
	}
	return raws, nil
}

func mergeFieldsForSchema(parentFields, childFields []*pkgmeta.Field, parentPath, childPath string) ([]*pkgmeta.Field, error) {
	result := make([]*pkgmeta.Field, 0, len(parentFields)+len(childFields))
	indexByName := make(map[string]int)

	for _, pf := range parentFields {
		if pf == nil || pf.Name == "" {
			continue
		}
		if _, exists := indexByName[pf.Name]; exists {
			continue
		}
		origin := pf.OriginModelPath
		if origin == "" {
			origin = parentPath
		}
		nf := cloneFieldShallow(pf)
		nf.OriginModelPath = origin
		indexByName[nf.Name] = len(result)
		result = append(result, nf)
	}

	for _, cf := range childFields {
		if cf == nil || cf.Name == "" {
			continue
		}
		nf := cloneFieldShallow(cf)
		nf.OriginModelPath = childPath
		if idx, ok := indexByName[nf.Name]; ok {
			merged, err := resolveSelectionFieldConflict(result[idx], nf)
			if err != nil {
				return nil, err
			}
			if merged != nil {
				merged.OriginModelPath = childPath
			}
			result[idx] = merged
			continue
		}
		if fieldHasSelectionAdd(nf) {
			return nil, fmt.Errorf("field %s selectionAdd requires an inherited static selection", nf.Name)
		}
		indexByName[nf.Name] = len(result)
		result = append(result, nf)
	}
	return result, nil
}

func mergeServicesForSchema(parentServices, childServices []*pkgmeta.Service, parentPath, childPath string) []*pkgmeta.Service {
	result := make([]*pkgmeta.Service, 0, len(parentServices)+len(childServices))
	indexByName := make(map[string]int)

	for _, ps := range parentServices {
		if ps == nil || ps.Name == "" {
			continue
		}
		if _, exists := indexByName[ps.Name]; exists {
			continue
		}
		origin := ps.OriginModelPath
		if origin == "" {
			origin = parentPath
		}
		ns := cloneServiceShallow(ps)
		ns.OriginModelPath = origin
		indexByName[ns.Name] = len(result)
		result = append(result, ns)
	}

	for _, cs := range childServices {
		if cs == nil || cs.Name == "" {
			continue
		}
		ns := cloneServiceShallow(cs)
		ns.OriginModelPath = childPath
		if idx, ok := indexByName[ns.Name]; ok {
			result[idx] = ns
			continue
		}
		indexByName[ns.Name] = len(result)
		result = append(result, ns)
	}
	return result
}

func cloneFieldShallow(src *pkgmeta.Field) *pkgmeta.Field {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = pkgmeta.BaseModel{}
	dst.ModelId = sqlNullStringEmpty()
	dst.Model = nil
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*pkgmeta.Decorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dd := *d
			dd.BaseModel = pkgmeta.BaseModel{}
			dd.ModelId = sqlNullStringEmpty()
			dd.FieldId = sqlNullStringEmpty()
			dd.ServiceId = sqlNullStringEmpty()
			dd.Model = nil
			dd.Field = nil
			dd.Service = nil
			if len(d.Arguments) > 0 {
				dd.Arguments = make([]*pkgmeta.Argument, 0, len(d.Arguments))
				for _, a := range d.Arguments {
					if a == nil {
						continue
					}
					aa := *a
					aa.BaseModel = pkgmeta.BaseModel{}
					aa.DecoratorId = sqlNullStringEmpty()
					aa.Decorator = nil
					dd.Arguments = append(dd.Arguments, &aa)
				}
			} else {
				dd.Arguments = nil
			}
			dst.Decorators = append(dst.Decorators, &dd)
		}
	} else {
		dst.Decorators = nil
	}
	return &dst
}

func cloneServiceShallow(src *pkgmeta.Service) *pkgmeta.Service {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = pkgmeta.BaseModel{}
	dst.ModelId = sqlNullStringEmpty()
	dst.Model = nil
	if len(src.Parameters) > 0 {
		dst.Parameters = make([]*pkgmeta.Parameter, 0, len(src.Parameters))
		for _, p := range src.Parameters {
			if p == nil {
				continue
			}
			pp := *p
			pp.BaseModel = pkgmeta.BaseModel{}
			pp.ServiceId = sqlNullStringEmpty()
			pp.Service = nil
			dst.Parameters = append(dst.Parameters, &pp)
		}
	} else {
		dst.Parameters = nil
	}
	if len(src.TypeParameters) > 0 {
		dst.TypeParameters = make([]*pkgmeta.TypeParameter, 0, len(src.TypeParameters))
		for _, tp := range src.TypeParameters {
			if tp == nil {
				continue
			}
			tt := *tp
			tt.BaseModel = pkgmeta.BaseModel{}
			tt.ServiceId = sqlNullStringEmpty()
			tt.Service = nil
			dst.TypeParameters = append(dst.TypeParameters, &tt)
		}
	} else {
		dst.TypeParameters = nil
	}
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*pkgmeta.Decorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dd := *d
			dd.BaseModel = pkgmeta.BaseModel{}
			dd.ModelId = sqlNullStringEmpty()
			dd.FieldId = sqlNullStringEmpty()
			dd.ServiceId = sqlNullStringEmpty()
			dd.Model = nil
			dd.Field = nil
			dd.Service = nil
			dd.Arguments = nil
			if len(d.Arguments) > 0 {
				dd.Arguments = make([]*pkgmeta.Argument, 0, len(d.Arguments))
				for _, a := range d.Arguments {
					if a == nil {
						continue
					}
					aa := *a
					aa.BaseModel = pkgmeta.BaseModel{}
					aa.DecoratorId = sqlNullStringEmpty()
					aa.Decorator = nil
					dd.Arguments = append(dd.Arguments, &aa)
				}
			}
			dst.Decorators = append(dst.Decorators, &dd)
		}
	} else {
		dst.Decorators = nil
	}
	return &dst
}

func sqlNullStringEmpty() sql.NullString {
	return sql.NullString{}
}
