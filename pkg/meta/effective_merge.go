// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"fmt"
	"sort"
)

// MergeSameNameModelsByExtensionChain merges same logical-name model rows (E2).
// Ranking: extends-depth ascending, then UpdatedAt ascending, then Id ascending.
// Fields merge by name via ResolveSelectionFieldConflict; Services last-write wins.
// Canonical model scalars and model-level Decorators come from the last ranked row
// (hangers follow the winning Field/Service/Model — EDS E2 §4.6). Cycles in Extends → error.
//
// Callers typically pass all live rows for one (application, name) — including sibling
// IMD branches (EDS2). Codegen may additionally pre-filter to a primary extension chain
// before calling this when reading legacy multi-row IMD catalogs.
func MergeSameNameModelsByExtensionChain(models []*Model) (*Model, error) {
	if len(models) == 0 {
		return nil, nil
	}

	byPath := make(map[string]*Model, len(models))
	for _, m := range models {
		if m != nil && m.Path != "" {
			byPath[m.Path] = m
		}
	}

	type rankedModel struct {
		model *Model
		depth int
	}

	depthMemo := make(map[string]int)
	visiting := make(map[string]bool)
	var depthErr error
	var depthOf func(*Model) int
	depthOf = func(m *Model) int {
		if m == nil || depthErr != nil {
			return 0
		}
		if m.Path == "" {
			return 0
		}
		if d, ok := depthMemo[m.Path]; ok {
			return d
		}
		if visiting[m.Path] {
			depthErr = fmt.Errorf("extends cycle detected at path %q", m.Path)
			return 0
		}
		visiting[m.Path] = true
		defer delete(visiting, m.Path)

		parent := byPath[m.Extends]
		depth := 0
		if parent != nil {
			depth = depthOf(parent) + 1
		}
		depthMemo[m.Path] = depth
		return depth
	}

	ranked := make([]rankedModel, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		ranked = append(ranked, rankedModel{model: m, depth: depthOf(m)})
		if depthErr != nil {
			return nil, depthErr
		}
	}
	if len(ranked) == 0 {
		return nil, nil
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		li := ranked[i]
		lj := ranked[j]
		if li.depth != lj.depth {
			return li.depth < lj.depth
		}
		if !li.model.UpdatedAt.Equal(lj.model.UpdatedAt) {
			return li.model.UpdatedAt.Before(lj.model.UpdatedAt)
		}
		return li.model.Id.String < lj.model.Id.String
	})

	fieldIndex := make(map[string]int)
	mergedFields := make([]*Field, 0)
	serviceIndex := make(map[string]int)
	mergedServices := make([]*Service, 0)

	for _, item := range ranked {
		m := item.model
		for _, f := range m.Fields {
			if f == nil || f.Name == "" {
				continue
			}
			if idx, ok := fieldIndex[f.Name]; ok {
				resolved, err := ResolveSelectionFieldConflict(mergedFields[idx], f)
				if err != nil {
					return nil, err
				}
				mergedFields[idx] = resolved
			} else {
				if FieldHasSelectionAdd(f) {
					return nil, fmt.Errorf("field %s selectionAdd requires an inherited static selection", f.Name)
				}
				fieldIndex[f.Name] = len(mergedFields)
				mergedFields = append(mergedFields, f)
			}
		}

		for _, s := range m.Services {
			if s == nil || s.Name == "" {
				continue
			}
			if idx, ok := serviceIndex[s.Name]; ok {
				mergedServices[idx] = s
			} else {
				serviceIndex[s.Name] = len(mergedServices)
				mergedServices = append(mergedServices, s)
			}
		}
	}

	canonical := ranked[len(ranked)-1].model
	merged := *canonical
	merged.Fields = mergedFields
	merged.Services = mergedServices
	return &merged, nil
}

// MergeEffectiveModel runs E2 over declaration-layer RawModel rows.
func MergeEffectiveModel(raws []*RawModel) (*Model, error) {
	return MergeSameNameModelsByExtensionChain(RawModelsAsModels(raws))
}

// RawModelsAsModels converts raw declaration trees into in-memory Model trees for E2 merge.
// IDs and timestamps are preserved for ranking; FK targets are not DB-backed.
func RawModelsAsModels(raws []*RawModel) []*Model {
	out := make([]*Model, 0, len(raws))
	for _, raw := range raws {
		if raw == nil {
			continue
		}
		out = append(out, rawModelAsModel(raw))
	}
	return out
}

func rawModelAsModel(raw *RawModel) *Model {
	m := &Model{
		BaseModel:    raw.BaseModel,
		Name:         raw.Name,
		Path:         raw.Path,
		Application:  raw.Application,
		ClassName:    raw.ClassName,
		ModelTable:   raw.ModelTable,
		Abstract:     raw.Abstract,
		AutoMigrate:  raw.AutoMigrate,
		Readonly:     raw.Readonly,
		RawExtends:   raw.RawExtends,
		Extends:      raw.Extends,
		CompanyField: raw.CompanyField,
		ModuleId:     raw.ModuleId,
	}
	for _, f := range raw.Fields {
		if f == nil {
			continue
		}
		m.Fields = append(m.Fields, rawFieldAsField(f))
	}
	for _, s := range raw.Services {
		if s == nil {
			continue
		}
		m.Services = append(m.Services, rawServiceAsService(s))
	}
	for _, d := range raw.Decorators {
		if d == nil {
			continue
		}
		m.Decorators = append(m.Decorators, rawDecoratorAsDecorator(d))
	}
	return m
}

func rawFieldAsField(f *RawField) *Field {
	out := &Field{
		BaseModel:                f.BaseModel,
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
	}
	for _, d := range f.Decorators {
		if d == nil {
			continue
		}
		out.Decorators = append(out.Decorators, rawDecoratorAsDecorator(d))
	}
	return out
}

func rawServiceAsService(s *RawService) *Service {
	svc := &Service{
		BaseModel:             s.BaseModel,
		Name:                  s.Name,
		OriginModelPath:       s.OriginModelPath,
		AccessibilityModifier: s.AccessibilityModifier,
		TsTypeAnnotation:      s.TsTypeAnnotation,
		ProtobufType:          s.ProtobufType,
		IsStatic:              s.IsStatic,
	}
	for _, p := range s.Parameters {
		if p == nil || p.Name == "this" {
			// Keep "this" on raw declaration rows; omit from effective / merge input
			// (matches codegen getApplication and ModuleBuilder.loadLatestModelByPath).
			continue
		}
		svc.Parameters = append(svc.Parameters, &Parameter{
			BaseModel:        p.BaseModel,
			Name:             p.Name,
			TsTypeAnnotation: p.TsTypeAnnotation,
			ProtobufType:     p.ProtobufType,
		})
	}
	for _, tp := range s.TypeParameters {
		if tp == nil {
			continue
		}
		svc.TypeParameters = append(svc.TypeParameters, &TypeParameter{
			BaseModel:      tp.BaseModel,
			Name:           tp.Name,
			ModuleSpecPath: tp.ModuleSpecPath,
			ReferenceIdent: tp.ReferenceIdent,
		})
	}
	for _, d := range s.Decorators {
		if d == nil {
			continue
		}
		svc.Decorators = append(svc.Decorators, rawDecoratorAsDecorator(d))
	}
	return svc
}

func rawDecoratorAsDecorator(d *RawDecorator) *Decorator {
	out := &Decorator{
		BaseModel:      d.BaseModel,
		Name:           d.Name,
		ModuleSpecPath: d.ModuleSpecPath,
		ReferenceIdent: d.ReferenceIdent,
	}
	for _, a := range d.Arguments {
		if a == nil {
			continue
		}
		out.Arguments = append(out.Arguments, &Argument{
			BaseModel:      a.BaseModel,
			Type:           a.Type,
			Value:          a.Value,
			ReferenceIdent: a.ReferenceIdent,
			ModuleSpecPath: a.ModuleSpecPath,
		})
	}
	return out
}
