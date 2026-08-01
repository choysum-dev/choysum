// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MergeSelectionByValue merges static selection options by value (PR-P2-F4 / D4).
// Same value → later label (and labelText) wins; order = base order then new values.
// A plain-string override replaces the whole option and clears inherited LabelText.
func MergeSelectionByValue(base, addends []IrFieldSelectionItem) []IrFieldSelectionItem {
	byValue := make(map[string]IrFieldSelectionItem)
	order := make([]string, 0, len(base))

	add := func(item IrFieldSelectionItem, overwrite bool) {
		value := strings.TrimSpace(item.Value)
		if value == "" {
			return
		}
		item.Value = value
		if _, exists := byValue[value]; !exists {
			order = append(order, value)
			byValue[value] = item
			return
		}
		if !overwrite {
			return
		}
		byValue[value] = item
	}

	for _, item := range base {
		add(item, true)
	}
	for _, item := range addends {
		add(item, true)
	}

	out := make([]IrFieldSelectionItem, 0, len(order))
	for _, value := range order {
		out = append(out, byValue[value])
	}
	return out
}

func selectionItemsFromField(field *IrField, spec *IrFieldResolvedSpec) []IrFieldSelectionItem {
	if spec != nil && len(spec.Structural.Selection) > 0 {
		return append([]IrFieldSelectionItem(nil), spec.Structural.Selection...)
	}
	if field == nil {
		return nil
	}
	raw := strings.TrimSpace(field.Selection)
	if raw == "" {
		return nil
	}
	var items []IrFieldSelectionItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func isDynamicSelectionField(field *IrField, spec *IrFieldResolvedSpec) bool {
	if spec != nil {
		if strings.TrimSpace(spec.Structural.SelectionKind) == "dynamic" {
			return true
		}
		if strings.TrimSpace(spec.Structural.SelectionMethod) != "" && len(spec.Structural.Selection) == 0 {
			return true
		}
	}
	if field != nil {
		if strings.TrimSpace(field.SelectionKind) == "dynamic" {
			return true
		}
		if strings.TrimSpace(field.SelectionMethod) != "" && strings.TrimSpace(field.Selection) == "" {
			return true
		}
	}
	return false
}

func childDeclaresFullSelection(spec *IrFieldResolvedSpec) bool {
	if spec == nil {
		return false
	}
	if len(spec.Structural.Selection) > 0 {
		return true
	}
	kind := strings.TrimSpace(spec.Structural.SelectionKind)
	if kind == "dynamic" {
		return true
	}
	if strings.TrimSpace(spec.Structural.SelectionMethod) != "" {
		return true
	}
	return false
}

// FieldHasSelectionAdd reports whether the field was authored with selectionAdd.
func FieldHasSelectionAdd(field *IrField) bool {
	if field == nil {
		return false
	}
	spec, err := field.GetResolvedSpec()
	if err != nil || spec == nil {
		return false
	}
	return spec.Structural.HasSelectionAdd
}

// ResolveSelectionFieldConflict applies PR-P2-F4 merge rules when a child field
// overrides a parent field of the same name.
//
// - selectionAdd only on a static base → merge into a field derived from base
// - full selection on child → replace with child (explicit fork)
// - otherwise → replace with child (legacy behavior)
func ResolveSelectionFieldConflict(base, child *IrField) (*IrField, error) {
	if child == nil {
		return base, nil
	}
	if base == nil {
		return child, nil
	}

	childSpec, err := child.GetResolvedSpec()
	if err != nil {
		return nil, err
	}
	if childSpec == nil || !childSpec.Structural.HasSelectionAdd {
		return child, nil
	}
	if childDeclaresFullSelection(childSpec) {
		return nil, fmt.Errorf(
			"field %s cannot combine selection and selectionAdd; use selectionAdd alone to append, or selection alone to replace",
			child.Name,
		)
	}

	baseSpec, err := base.GetResolvedSpec()
	if err != nil {
		return nil, err
	}
	if isDynamicSelectionField(base, baseSpec) {
		return nil, fmt.Errorf("field %s selectionAdd requires an inherited static selection", child.Name)
	}

	baseItems := selectionItemsFromField(base, baseSpec)
	baseKind := ""
	if baseSpec != nil {
		baseKind = strings.TrimSpace(baseSpec.Structural.SelectionKind)
	}
	if baseKind == "" && base != nil {
		baseKind = strings.TrimSpace(base.SelectionKind)
	}
	if len(baseItems) == 0 && baseKind != "static" {
		return nil, fmt.Errorf("field %s selectionAdd requires an inherited static selection", child.Name)
	}

	mergedItems := MergeSelectionByValue(baseItems, childSpec.Structural.SelectionAdd)

	// Preserve base field metadata, then overlay child structural options and merged selection.
	out := *base
	out.BaseModel = BaseModel{}
	out.ModelId = child.ModelId
	out.Model = nil
	out.Decorators = child.Decorators
	out.OriginModelPath = child.OriginModelPath
	if strings.TrimSpace(child.FieldType) != "" {
		out.FieldType = child.FieldType
	}
	if strings.TrimSpace(child.TsTypeAnnotation) != "" {
		out.TsTypeAnnotation = child.TsTypeAnnotation
	}
	if strings.TrimSpace(child.TsTypeReference) != "" {
		out.TsTypeReference = child.TsTypeReference
	}

	outSpec := &IrFieldResolvedSpec{}
	if baseSpec != nil {
		*outSpec = *baseSpec
	} else {
		outSpec.FieldName = child.Name
		outSpec.Structural.Name = child.Name
		outSpec.Structural.FieldType = "selection"
	}
	// Overlay authored child structural/behavior bits when present.
	overlayStructuralSelectionAdd(&outSpec.Structural, &childSpec.Structural)
	overlayBehaviorSelectionAdd(&outSpec.Behavior, &childSpec.Behavior)
	outSpec.Structural.Selection = mergedItems
	outSpec.Structural.SelectionKind = "static"
	outSpec.Structural.SelectionMethod = ""
	outSpec.Structural.SelectionAdd = nil
	outSpec.Structural.HasSelectionAdd = false
	outSpec.FieldName = child.Name
	outSpec.Structural.Name = child.Name

	if err := out.SetResolvedSpec(outSpec); err != nil {
		return nil, err
	}
	applySelectionLegacyColumns(&out, outSpec)
	return &out, nil
}

func overlayStructuralSelectionAdd(dst, src *IrFieldStructuralSpec) {
	if dst == nil || src == nil {
		return
	}
	if strings.TrimSpace(src.FieldType) != "" {
		dst.FieldType = src.FieldType
	}
	if strings.TrimSpace(src.String) != "" {
		dst.String = src.String
	}
	if src.StringText != nil {
		dst.StringText = src.StringText
	}
	if strings.TrimSpace(src.Help) != "" {
		dst.Help = src.Help
	}
	if src.HelpText != nil {
		dst.HelpText = src.HelpText
	}
	if src.Related != nil {
		dst.Related = src.Related
	}
	if src.StorageHints != nil {
		dst.StorageHints = src.StorageHints
	}
	if strings.TrimSpace(src.ColumnType) != "" {
		dst.ColumnType = src.ColumnType
	}
	if strings.TrimSpace(src.CheckConstraint) != "" {
		dst.CheckConstraint = src.CheckConstraint
	}
	if src.Translate != nil {
		dst.Translate = src.Translate
	}
	if src.CompanyDependent != nil {
		dst.CompanyDependent = src.CompanyDependent
	}
	if src.Copy != nil {
		dst.Copy = src.Copy
	}
	if src.Readonly != nil {
		dst.Readonly = src.Readonly
	}
	if src.MaxUploadBytes != nil {
		dst.MaxUploadBytes = src.MaxUploadBytes
	}
	if src.MaxWidth != nil {
		dst.MaxWidth = src.MaxWidth
	}
	if src.MaxHeight != nil {
		dst.MaxHeight = src.MaxHeight
	}
}

func overlayBehaviorSelectionAdd(dst, src *IrFieldBehaviorSpec) {
	if dst == nil || src == nil {
		return
	}
	if src.Compute != nil {
		dst.Compute = src.Compute
	}
	if src.SqlCompute != nil {
		dst.SqlCompute = src.SqlCompute
	}
	if src.Inverse != nil {
		dst.Inverse = src.Inverse
	}
	if src.Search != nil {
		dst.Search = src.Search
	}
}

func applySelectionLegacyColumns(field *IrField, spec *IrFieldResolvedSpec) {
	if field == nil || spec == nil {
		return
	}
	if len(spec.Structural.Selection) > 0 {
		if b, err := json.Marshal(spec.Structural.Selection); err == nil {
			field.Selection = string(b)
		}
	} else {
		field.Selection = ""
	}
	field.SelectionKind = strings.TrimSpace(spec.Structural.SelectionKind)
	field.SelectionMethod = strings.TrimSpace(spec.Structural.SelectionMethod)
	if strings.TrimSpace(spec.Structural.String) != "" {
		field.FieldString = strings.TrimSpace(spec.Structural.String)
	}
	if spec.Structural.StringText != nil {
		if b, err := json.Marshal(spec.Structural.StringText); err == nil {
			field.StringText = string(b)
		}
	}
	field.FieldHelp = ""
	field.HelpText = ""
	if strings.TrimSpace(spec.Structural.Help) != "" {
		field.FieldHelp = strings.TrimSpace(spec.Structural.Help)
	}
	if spec.Structural.HelpText != nil {
		if b, err := json.Marshal(spec.Structural.HelpText); err == nil {
			field.HelpText = string(b)
		}
	}
	if spec.Behavior.Compute != nil || spec.Behavior.SqlCompute != nil {
		field.IsReadonly = true
	}
	if spec.Structural.Readonly != nil && *spec.Structural.Readonly {
		field.IsReadonly = true
	}
	field.MaxUploadBytes = 0
	field.MaxWidth = 0
	field.MaxHeight = 0
	if spec.Structural.MaxUploadBytes != nil && *spec.Structural.MaxUploadBytes > 0 {
		field.MaxUploadBytes = *spec.Structural.MaxUploadBytes
	}
	if spec.Structural.MaxWidth != nil && *spec.Structural.MaxWidth > 0 {
		field.MaxWidth = *spec.Structural.MaxWidth
	}
	if spec.Structural.MaxHeight != nil && *spec.Structural.MaxHeight > 0 {
		field.MaxHeight = *spec.Structural.MaxHeight
	}
	if spec.Structural.StorageHints != nil {
		hints := spec.Structural.StorageHints
		if hints.Required != nil {
			field.NotNull = *hints.Required
		}
		if hints.Indexed != nil {
			field.Indexed = *hints.Indexed
		}
		if hints.Size != nil {
			field.Size = *hints.Size
		}
		if hints.Precision != nil {
			field.Precision = *hints.Precision
		}
		if hints.Scale != nil {
			field.Scale = *hints.Scale
		}
	}
}
