// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

type resolvedFieldBehaviorBinding struct {
	compute    *meta.IrFieldBehaviorComputeSpec
	sqlCompute *meta.IrFieldBehaviorSqlComputeSpec
	search     *meta.IrFieldBehaviorMethodRef
	inverse    *meta.IrFieldBehaviorMethodRef
}

func parseDecoratorObjectArg(args []*parser.Argument, index int) (map[string]any, error) {
	if len(args) <= index || args[index] == nil {
		return nil, nil
	}
	if args[index].Type != "ObjectLiteral" || strings.TrimSpace(args[index].Value) == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(args[index].Value), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseDecoratorStringArg(args []*parser.Argument, index int) string {
	if len(args) <= index || args[index] == nil {
		return ""
	}
	raw := strings.TrimSpace(args[index].Value)
	raw = strings.Trim(raw, "`\"'")
	return strings.TrimSpace(raw)
}

func parseStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		s := strings.TrimSpace(fmt.Sprintf("%v", item))
		if s == "" {
			continue
		}
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func toBoolPtr(value bool) *bool {
	v := value
	return &v
}

func toIntPtr(value int) *int {
	v := value
	return &v
}

func toStringPtr(value string) *string {
	v := value
	return &v
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	default:
		return 0, false
	}
}

func collectFieldBehaviorBindings(methods []*parser.MemberMethod) (map[string]*resolvedFieldBehaviorBinding, map[string][]meta.IrFieldDiagnostic, error) {
	bindings := make(map[string]*resolvedFieldBehaviorBinding)
	diagnostics := make(map[string][]meta.IrFieldDiagnostic)

	addDiag := func(field string, code string, message string) {
		diagnostics[field] = append(diagnostics[field], meta.IrFieldDiagnostic{
			Code:     code,
			Severity: "error",
			Message:  message,
		})
	}

	for _, method := range methods {
		if method == nil || len(method.Decorators) == 0 {
			continue
		}

		for _, decorator := range method.Decorators {
			if decorator == nil {
				continue
			}
			if decorator.Name != "Compute" && decorator.Name != "SqlCompute" && decorator.Name != "Search" && decorator.Name != "Inverse" {
				continue
			}

			if len(method.Parameters) > 0 {
				return nil, nil, fmt.Errorf("method %s decorated by @%s must be parameterless", method.Name, decorator.Name)
			}

			fieldName := parseDecoratorStringArg(decorator.Arguments, 0)
			if fieldName == "" {
				return nil, nil, fmt.Errorf("@%s on method %s requires a field name", decorator.Name, method.Name)
			}

			binding := bindings[fieldName]
			if binding == nil {
				binding = &resolvedFieldBehaviorBinding{}
				bindings[fieldName] = binding
			}

			switch decorator.Name {
			case "Compute":
				if binding.compute != nil {
					addDiag(fieldName, "DUPLICATE_COMPUTE", "same field declares multiple @Compute handlers")
					continue
				}
				opts, err := parseDecoratorObjectArg(decorator.Arguments, 1)
				if err != nil {
					return nil, nil, fmt.Errorf("parse @Compute(%s) options: %w", fieldName, err)
				}
				deps := parseStringSlice(opts["deps"])
				if len(deps) == 0 {
					return nil, nil, fmt.Errorf("@Compute(%s) deps must be a non-empty array", fieldName)
				}
				store := true
				if v, ok := opts["store"].(bool); ok {
					store = v
				}
				var searchable *bool
				if v, ok := opts["searchable"].(bool); ok {
					searchable = toBoolPtr(v)
				}
				runAs := ""
				if v, ok := opts["runAs"].(string); ok {
					runAs = strings.TrimSpace(v)
				}
				binding.compute = &meta.IrFieldBehaviorComputeSpec{
					Method:     method.Name,
					Deps:       deps,
					Store:      store,
					Searchable: searchable,
					RunAs:      runAs,
				}
			case "SqlCompute":
				if binding.sqlCompute != nil {
					addDiag(fieldName, "DUPLICATE_SQL_COMPUTE", "same field declares multiple @SqlCompute handlers")
					continue
				}
				binding.sqlCompute = &meta.IrFieldBehaviorSqlComputeSpec{
					Method:     method.Name,
					CtxType:    "SqlComputeCtx",
					ReturnType: "SelectExpressionValue",
				}
			case "Search":
				if binding.search != nil {
					addDiag(fieldName, "DUPLICATE_SEARCH", "same field declares multiple @Search handlers")
					continue
				}
				binding.search = &meta.IrFieldBehaviorMethodRef{Method: method.Name}
			case "Inverse":
				if binding.inverse != nil {
					addDiag(fieldName, "DUPLICATE_INVERSE", "same field declares multiple @Inverse handlers")
					continue
				}
				binding.inverse = &meta.IrFieldBehaviorMethodRef{Method: method.Name}
			}
		}
	}

	return bindings, diagnostics, nil
}

func resolveColumnType(fieldType string) string {
	switch fieldType {
	case "selection":
		return "varchar"
	case "ManyToOne", "ManyToOneRef":
		return "char"
	case "ManyToManyRef":
		return "jsonobject"
	case "binary", "image":
		return "blob"
	default:
		return fieldType
	}
}

func buildFieldResolvedSpec(field *meta.IrField, binding *resolvedFieldBehaviorBinding, inherited []meta.IrFieldDiagnostic) (*meta.IrFieldResolvedSpec, error) {
	if field == nil {
		return nil, nil
	}

	var options map[string]any
	for _, decorator := range field.Decorators {
		if decorator == nil || decorator.Name != "Field" || len(decorator.Arguments) == 0 {
			continue
		}
		arg := decorator.Arguments[0]
		if arg == nil || arg.Type != "ObjectLiteral" || strings.TrimSpace(arg.Value) == "" {
			continue
		}
		if err := json.Unmarshal([]byte(arg.Value), &options); err != nil {
			return nil, fmt.Errorf("parse @Field(%s) options: %w", field.Name, err)
		}
		break
	}

	if len(options) == 0 {
		return nil, nil
	}
	_, hasLegacyColumnOption := options["column"]
	_, hasLegacySelectOption := options["select"]
	if hasLegacyColumnOption {
		return nil, fmt.Errorf("FIELD_LEGACY_SYNTAX_FORBIDDEN: @Field(%s) uses legacy option column", field.Name)
	}
	if hasLegacySelectOption {
		return nil, fmt.Errorf("FIELD_LEGACY_SYNTAX_FORBIDDEN: @Field(%s) uses legacy option select", field.Name)
	}

	fieldType, _ := options["type"].(string)
	fieldType = strings.TrimSpace(fieldType)
	if fieldType == "" {
		return nil, fmt.Errorf("@Field(%s) missing required type", field.Name)
	}

	spec := &meta.IrFieldResolvedSpec{
		FieldName: field.Name,
		Structural: meta.IrFieldStructuralSpec{
			Name:      field.Name,
			FieldType: fieldType,
		},
		Diagnostics: append([]meta.IrFieldDiagnostic{}, inherited...),
	}

	if relation, ok := options["relation"].(map[string]any); ok {
		spec.Structural.Relation = relation
	}

	if selection, ok := options["selection"].([]any); ok {
		for _, item := range selection {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if entry["value"] == nil || entry["label"] == nil {
				continue
			}
			value := strings.TrimSpace(fmt.Sprintf("%v", entry["value"]))
			label := strings.TrimSpace(fmt.Sprintf("%v", entry["label"]))
			if value == "" || label == "" {
				continue
			}
			spec.Structural.Selection = append(spec.Structural.Selection, map[string]string{"value": value, "label": label})
		}
	}

	if related, ok := options["related"].(map[string]any); ok {
		path := ""
		if related["path"] != nil {
			path = strings.TrimSpace(fmt.Sprintf("%v", related["path"]))
		}
		relatedSpec := &meta.IrFieldRelatedSpec{
			Path: path,
		}
		if v, ok := related["store"].(bool); ok {
			relatedSpec.Store = v
		}
		relatedSpec.Deps = parseStringSlice(related["deps"])
		if relatedSpec.Path != "" {
			spec.Structural.Related = relatedSpec
		}
	}

	hints := &meta.IrFieldStructuralStorageHints{}
	if v, ok := options["required"].(bool); ok {
		hints.Required = toBoolPtr(v)
	}
	if hints.Required == nil {
		if v, ok := options["notNull"].(bool); ok {
			hints.Required = toBoolPtr(v)
		}
	}
	if v, ok := options["indexed"].(bool); ok {
		hints.Indexed = toBoolPtr(v)
	}
	if hints.Indexed == nil {
		switch raw := options["index"].(type) {
		case bool:
			hints.Indexed = toBoolPtr(raw)
		case string:
			trimmed := strings.TrimSpace(raw)
			if trimmed != "" {
				hints.Indexed = toBoolPtr(true)
				hints.Index = toStringPtr(trimmed)
			}
		}
	}
	if v, ok := asInt(options["size"]); ok {
		hints.Size = toIntPtr(v)
	}
	if v, ok := asInt(options["precision"]); ok {
		hints.Precision = toIntPtr(v)
	}
	if v, ok := asInt(options["scale"]); ok {
		hints.Scale = toIntPtr(v)
	}
	if v, ok := options["primaryKey"].(bool); ok {
		hints.PrimaryKey = toBoolPtr(v)
	}
	if v, ok := options["unique"].(bool); ok {
		hints.Unique = toBoolPtr(v)
	}
	switch raw := options["uniqueIndex"].(type) {
	case bool:
		hints.UniqueIndexEnabled = toBoolPtr(raw)
	case string:
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			hints.UniqueIndex = toStringPtr(trimmed)
		}
	}
	switch raw := options["default"].(type) {
	case string:
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			hints.Default = toStringPtr(trimmed)
		}
	case bool, float64, float32, int, int32, int64, uint, uint32, uint64:
		hints.Default = toStringPtr(strings.TrimSpace(fmt.Sprintf("%v", raw)))
	}
	if hints.Required != nil || hints.Indexed != nil || hints.Index != nil || hints.Size != nil || hints.Precision != nil || hints.Scale != nil || hints.PrimaryKey != nil || hints.Unique != nil || hints.UniqueIndex != nil || hints.UniqueIndexEnabled != nil || hints.Default != nil {
		spec.Structural.StorageHints = hints
	}

	if binding != nil {
		spec.Behavior.Compute = binding.compute
		spec.Behavior.SqlCompute = binding.sqlCompute
		spec.Behavior.Search = binding.search
		spec.Behavior.Inverse = binding.inverse
	}

	if v, ok := options["checkConstraint"].(string); ok {
		spec.Structural.CheckConstraint = strings.TrimSpace(v)
	}

	storeValue := true
	storeSource := "default"
	if spec.Behavior.SqlCompute != nil {
		storeValue = false
		storeSource = "@SqlCompute"
	} else if spec.Behavior.Compute != nil {
		storeValue = spec.Behavior.Compute.Store
		storeSource = "@Compute"
	} else if spec.Structural.Related != nil {
		storeValue = spec.Structural.Related.Store
		storeSource = "@Field.related.store"
	}
	spec.Resolved.Store = meta.IrResolvedValue[bool]{Value: storeValue, Source: storeSource}

	if spec.Behavior.Search != nil {
		v := true
		spec.Resolved.Searchable = meta.IrResolvedValue[*bool]{Value: &v, Source: "@Search"}
	} else if spec.Behavior.Compute != nil && spec.Behavior.Compute.Searchable != nil {
		spec.Resolved.Searchable = meta.IrResolvedValue[*bool]{Value: spec.Behavior.Compute.Searchable, Source: "@Compute.searchable"}
	}

	if spec.Behavior.Compute != nil && strings.TrimSpace(spec.Behavior.Compute.RunAs) != "" {
		runAs := strings.TrimSpace(spec.Behavior.Compute.RunAs)
		spec.Resolved.RunAs = meta.IrResolvedValue[*string]{Value: &runAs, Source: "@Compute.runAs"}
	}

	columnType := resolveColumnType(fieldType)
	spec.Structural.ColumnType = columnType

	spec.Migration = meta.IrFieldMigrationDecision{
		StorageKind:        "physical",
		ShouldCreateColumn: true,
		ResolvedColumnType: columnType,
		ReasonCode:         "FIELD_DEFAULT",
	}

	switch {
	case spec.Behavior.SqlCompute != nil:
		spec.Migration.StorageKind = "virtualSql"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ReasonCode = "SQL_COMPUTE"
		spec.Migration.ResolvedColumnType = ""
	case fieldType == "OneToMany" || fieldType == "ManyToMany":
		spec.Migration.StorageKind = "relationOnly"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ReasonCode = "RELATION_NON_COLUMN"
		spec.Migration.ResolvedColumnType = ""
	case spec.Structural.Related != nil && !spec.Structural.Related.Store:
		spec.Migration.StorageKind = "virtualRuntime"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ReasonCode = "RELATED_STORE_FALSE"
		spec.Migration.ResolvedColumnType = ""
	case spec.Behavior.Compute != nil && !spec.Behavior.Compute.Store:
		spec.Migration.StorageKind = "virtualRuntime"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ReasonCode = "COMPUTE_STORE_FALSE"
		spec.Migration.ResolvedColumnType = ""
	case spec.Behavior.Compute != nil && spec.Behavior.Compute.Store:
		spec.Migration.StorageKind = "physical"
		spec.Migration.ShouldCreateColumn = true
		spec.Migration.ReasonCode = "COMPUTE_STORE_TRUE"
	}

	if spec.Behavior.Compute != nil && spec.Behavior.SqlCompute != nil {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_COMPUTE_SQLCOMPUTE",
			Severity: "error",
			Message:  "same field cannot declare both @Compute and @SqlCompute",
		})
	}
	if spec.Behavior.SqlCompute != nil && spec.Structural.Related != nil && spec.Structural.Related.Store {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_SQLCOMPUTE_RELATED_STORE",
			Severity: "error",
			Message:  "@SqlCompute cannot be combined with related.store=true",
		})
	}
	if spec.Behavior.Inverse != nil && spec.Behavior.Compute == nil && spec.Structural.Related == nil {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_INVERSE_WITHOUT_SOURCE",
			Severity: "error",
			Message:  "inverse handler requires compute or related field source",
		})
	}
	if spec.Behavior.Search != nil && spec.Migration.StorageKind == "physical" {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "WARN_SEARCH_ON_PHYSICAL_FIELD",
			Severity: "warning",
			Message:  "search handler is usually unnecessary on physical fields unless rewrite is required",
		})
	}
	if (fieldType == "OneToMany" || fieldType == "ManyToMany") && spec.Migration.ShouldCreateColumn {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_RELATION_TO_MANY_COLUMN",
			Severity: "error",
			Message:  "OneToMany/ManyToMany fields cannot create physical columns",
		})
	}
	if spec.Behavior.Compute != nil && !spec.Behavior.Compute.Store && spec.Structural.StorageHints != nil && spec.Structural.StorageHints.Required != nil && *spec.Structural.StorageHints.Required {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_REQUIRED_VIRTUAL_COMPUTE",
			Severity: "error",
			Message:  "compute.store=false field cannot be required",
		})
	}
	if spec.Structural.Related != nil && !spec.Structural.Related.Store && spec.Behavior.Inverse != nil {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_INVERSE_ON_NON_STORED_RELATED",
			Severity: "error",
			Message:  "related.store=false field cannot declare inverse handler",
		})
	}

	return spec, nil
}

func applyResolvedSpecToLegacyField(field *meta.IrField, spec *meta.IrFieldResolvedSpec) {
	if field == nil || spec == nil {
		return
	}
	field.FieldType = spec.Structural.FieldType
	field.Relation = spec.Structural.FieldType

	if spec.Behavior.Compute != nil || spec.Behavior.SqlCompute != nil {
		field.IsReadonly = true
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

	if len(spec.Structural.Selection) > 0 {
		if b, err := json.Marshal(spec.Structural.Selection); err == nil {
			field.Selection = string(b)
		}
	}

	if relation := spec.Structural.Relation; relation != nil {
		if v, ok := relation["targetModel"]; ok {
			field.RelationModel = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if v, ok := relation["inverseField"]; ok {
			field.RelationInverseField = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if v, ok := relation["joinField"]; ok {
			field.RelationJoinField = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if v, ok := relation["inverseJoinField"]; ok {
			field.RelationInverseJoinField = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if v, ok := relation["joinModel"]; ok {
			field.RelationJoinModel = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
}
