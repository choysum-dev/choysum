// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type expandsShape struct {
	fields   []*Field
	services []*Service
}

// ExpandModelsAlongExtends merges parent Fields and Services into each model in memory
// by walking Extends paths against meta_raw_* (and the provided local set).
//
// Used by:
//   - schema migrator (DDL columns for thin subclasses)
//   - RecomputeEffective (effective shape must include BaseModel methods/fields;
//     E2 same-name union alone does not pull differently-named parents)
//
// Does not write to the database.
func ExpandModelsAlongExtends(db *gorm.DB, models []*Model) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	localByPath := make(map[string]*Model, len(models))
	for _, m := range models {
		if m == nil || strings.TrimSpace(m.Path) == "" {
			continue
		}
		localByPath[m.Path] = m
	}
	cache := make(map[string]*expandsShape)
	visiting := make(map[string]bool)
	for _, m := range models {
		if m == nil {
			continue
		}
		shape, err := expandShapeAlongExtends(db, m, localByPath, cache, visiting)
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
	model *Model,
	localByPath map[string]*Model,
	cache map[string]*expandsShape,
	visiting map[string]bool,
) (*expandsShape, error) {
	if model == nil {
		return &expandsShape{}, nil
	}
	path := strings.TrimSpace(model.Path)
	if path != "" {
		if cached, ok := cache[path]; ok {
			return cached, nil
		}
		if visiting[path] {
			return nil, fmt.Errorf("circular extends while expanding model shape: %s", path)
		}
		visiting[path] = true
		defer delete(visiting, path)
	}

	var parentShape *expandsShape
	parentPath := ""
	if extends := strings.TrimSpace(model.Extends); extends != "" {
		parent, err := resolveExtendsModel(db, extends, localByPath, strings.TrimSpace(model.Application))
		if err != nil {
			return nil, err
		}
		if parent != nil {
			parentPath = parent.Path
			parentShape, err = expandShapeAlongExtends(db, parent, localByPath, cache, visiting)
			if err != nil {
				return nil, err
			}
		}
	}

	var parentFields []*Field
	var parentServices []*Service
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
	if path != "" {
		cache[path] = out
	}
	return out, nil
}

func resolveExtendsModel(db *gorm.DB, extendsPath string, localByPath map[string]*Model, preferredApp string) (*Model, error) {
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
	converted := RawModelsAsModels([]*RawModel{&raws[0]})
	return converted[0], nil
}

func loadRawParentsByPath(db *gorm.DB, extendsPath, application string) ([]RawModel, error) {
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
	var raws []RawModel
	if err := q.Order("id DESC").Find(&raws).Error; err != nil {
		return nil, fmt.Errorf("load raw parent by path %s: %w", extendsPath, err)
	}
	return raws, nil
}

func mergeFieldsForSchema(parentFields, childFields []*Field, parentPath, childPath string) ([]*Field, error) {
	result := make([]*Field, 0, len(parentFields)+len(childFields))
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
			merged, err := ResolveSelectionFieldConflict(result[idx], nf)
			if err != nil {
				return nil, err
			}
			if merged != nil {
				merged.OriginModelPath = childPath
			}
			result[idx] = merged
			continue
		}
		if FieldHasSelectionAdd(nf) {
			return nil, fmt.Errorf("field %s selectionAdd requires an inherited static selection", nf.Name)
		}
		indexByName[nf.Name] = len(result)
		result = append(result, nf)
	}
	return result, nil
}

func mergeServicesForSchema(parentServices, childServices []*Service, parentPath, childPath string) []*Service {
	result := make([]*Service, 0, len(parentServices)+len(childServices))
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

func cloneFieldShallow(src *Field) *Field {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = BaseModel{}
	dst.ModelId = sqlNullStringEmpty()
	dst.Model = nil
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*Decorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dd := *d
			dd.BaseModel = BaseModel{}
			dd.ModelId = sqlNullStringEmpty()
			dd.FieldId = sqlNullStringEmpty()
			dd.ServiceId = sqlNullStringEmpty()
			dd.Model = nil
			dd.Field = nil
			dd.Service = nil
			if len(d.Arguments) > 0 {
				dd.Arguments = make([]*Argument, 0, len(d.Arguments))
				for _, a := range d.Arguments {
					if a == nil {
						continue
					}
					aa := *a
					aa.BaseModel = BaseModel{}
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

func cloneServiceShallow(src *Service) *Service {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = BaseModel{}
	dst.ModelId = sqlNullStringEmpty()
	dst.Model = nil
	if len(src.Parameters) > 0 {
		dst.Parameters = make([]*Parameter, 0, len(src.Parameters))
		for _, p := range src.Parameters {
			if p == nil {
				continue
			}
			pp := *p
			pp.BaseModel = BaseModel{}
			pp.ServiceId = sqlNullStringEmpty()
			pp.Service = nil
			dst.Parameters = append(dst.Parameters, &pp)
		}
	} else {
		dst.Parameters = nil
	}
	if len(src.TypeParameters) > 0 {
		dst.TypeParameters = make([]*TypeParameter, 0, len(src.TypeParameters))
		for _, tp := range src.TypeParameters {
			if tp == nil {
				continue
			}
			tt := *tp
			tt.BaseModel = BaseModel{}
			tt.ServiceId = sqlNullStringEmpty()
			tt.Service = nil
			dst.TypeParameters = append(dst.TypeParameters, &tt)
		}
	} else {
		dst.TypeParameters = nil
	}
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*Decorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dd := *d
			dd.BaseModel = BaseModel{}
			dd.ModelId = sqlNullStringEmpty()
			dd.FieldId = sqlNullStringEmpty()
			dd.ServiceId = sqlNullStringEmpty()
			dd.Model = nil
			dd.Field = nil
			dd.Service = nil
			dd.Arguments = nil
			if len(d.Arguments) > 0 {
				dd.Arguments = make([]*Argument, 0, len(d.Arguments))
				for _, a := range d.Arguments {
					if a == nil {
						continue
					}
					aa := *a
					aa.BaseModel = BaseModel{}
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
