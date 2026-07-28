// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"encoding/json"
	"fmt"
	"regexp"
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
				if method.IsAsync && !store {
					addDiag(fieldName, "ASYNC_VIRTUAL_COMPUTE", "virtual compute handler (store: false) cannot be async")
				}
				var searchable *bool
				if v, ok := opts["searchable"].(bool); ok {
					searchable = toBoolPtr(v)
				}
				binding.compute = &meta.IrFieldBehaviorComputeSpec{
					Method:     method.Name,
					Deps:       deps,
					Store:      store,
					Searchable: searchable,
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
	case "monetary":
		// Physical storage matches decimal (DECIMAL(38,18)); logical FieldType stays monetary.
		return "decimal"
	default:
		return fieldType
	}
}

var (
	termReferenceCallPattern = regexp.MustCompile(`(?s)^([A-Za-z_$][\w$]*)\s*\(\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)(?:\s*,\s*\{(.*?)\})?\s*\)$`)
)

func parseTextCallLiteral(value string) (string, bool) {
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") {
		return strings.TrimSuffix(strings.TrimPrefix(value, "`"), "`"), true
	}
	parsed, err := parser.ParseJSStringLiteral(value)
	return parsed, err == nil
}

func parseTermReferenceCall(raw string, ownerModule string, defaultScope string, bindings map[string]parser.TranslateBinding) (*meta.TermReference, bool) {
	match := termReferenceCallPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 4 || strings.TrimSpace(ownerModule) == "" {
		return nil, false
	}
	callee := strings.TrimSpace(match[1])
	binding, known := bindings[callee]
	referenceOutput := (known && binding.ReferenceOutput) || callee == "_lt"
	if !known && callee != "_lt" {
		return nil, false
	}
	if !referenceOutput {
		return nil, false
	}
	src, srcOK := parseTextCallLiteral(match[2])
	scope := strings.TrimSpace(defaultScope)
	if known && strings.TrimSpace(binding.DefaultScope) != "" && scope == "" {
		scope = strings.TrimSpace(binding.DefaultScope)
	}
	scopeOK := scope != ""
	if strings.TrimSpace(match[3]) != "" {
		scope = strings.TrimSpace(parseFactoryStringOption(match[3], referenceScopePattern))
		if scope == "" {
			pathValue := strings.TrimSpace(parseFactoryStringOption(match[3], referencePathPattern))
			locationValue := strings.TrimSpace(parseFactoryStringOption(match[3], referenceLocationPattern))
			scope = pathValue
			if scope != "" && locationValue != "" {
				scope += "@" + locationValue
			}
		}
		scopeOK = scope != ""
	}
	if !srcOK || !scopeOK || strings.TrimSpace(src) == "" || strings.TrimSpace(scope) == "" {
		return nil, false
	}
	module := ownerModule
	if known && strings.TrimSpace(binding.Module) != "" {
		module = strings.TrimSpace(binding.Module)
	}
	reference := meta.NewTermReference(module, scope, src, "literal")
	return &reference, true
}

func buildFieldResolvedSpec(field *meta.IrField, binding *resolvedFieldBehaviorBinding, inherited []meta.IrFieldDiagnostic, ownerModule string, referenceScope string, translateBindings map[string]parser.TranslateBinding) (*meta.IrFieldResolvedSpec, error) {
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
		if binding != nil {
			return nil, fmt.Errorf("field %s has behavior decorators but is missing @Field decorator", field.Name)
		}
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

	if stringRaw, hasString := options["string"]; hasString && stringRaw != nil {
		raw := strings.TrimSpace(fmt.Sprintf("%v", stringRaw))
		if raw != "" {
			if reference, ok := parseTermReferenceCall(raw, ownerModule, referenceScope, translateBindings); ok {
				spec.Structural.String = reference.Src
				spec.Structural.StringText = reference
			} else if match := termReferenceCallPattern.FindStringSubmatch(raw); len(match) == 4 {
				if _, known := translateBindings[strings.TrimSpace(match[1])]; known {
					if fallback, ok := parseTextCallLiteral(match[2]); ok {
						spec.Structural.String = fallback
					}
				}
			} else if literal, err := parser.ParseJSStringLiteral(raw); err == nil && strings.TrimSpace(literal) != "" {
				spec.Structural.String = strings.TrimSpace(literal)
			} else if !strings.ContainsAny(raw, "(){}[]") {
				// Plain identifier-free string already unquoted by ObjectLiteral encoding.
				spec.Structural.String = raw
			}
		}
	}

	if selectionRaw, hasSelection := options["selection"]; hasSelection && selectionRaw != nil {
		switch selection := selectionRaw.(type) {
		case []any:
			for _, item := range selection {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if entry["value"] == nil || entry["label"] == nil {
					continue
				}
				value := strings.TrimSpace(fmt.Sprintf("%v", entry["value"]))
				labelRaw := strings.TrimSpace(fmt.Sprintf("%v", entry["label"]))
				if reference, ok := parseTermReferenceCall(labelRaw, ownerModule, referenceScope, translateBindings); ok {
					src := strings.TrimSpace(reference.Src)
					if value == "" || src == "" {
						continue
					}
					spec.Structural.Selection = append(spec.Structural.Selection, meta.IrFieldSelectionItem{
						Value:     value,
						Label:     src,
						LabelText: reference,
					})
					continue
				}
				if match := termReferenceCallPattern.FindStringSubmatch(labelRaw); len(match) == 4 {
					callee := strings.TrimSpace(match[1])
					binding, known := translateBindings[callee]
					isLt := callee == "_lt" || (known && binding.ReferenceOutput)
					if isLt {
						// Empty/invalid _lt(...) — skip option rather than treat as text _t.
						continue
					}
					return nil, fmt.Errorf("FIELD_SELECTION_LABELTEXT_FORBIDDEN: @Field(%s) selection label must not use text _t(...); use _lt(...) or a bare string", field.Name)
				}
				label := labelRaw
				if literal, err := parser.ParseJSStringLiteral(labelRaw); err == nil {
					label = strings.TrimSpace(literal)
				}
				if value == "" || label == "" {
					continue
				}
				spec.Structural.Selection = append(spec.Structural.Selection, meta.IrFieldSelectionItem{
					Value: value,
					Label: label,
				})
			}
			if len(spec.Structural.Selection) > 0 {
				spec.Structural.SelectionKind = "static"
			}
		case string:
			trimmed := strings.TrimSpace(selection)
			if trimmed == "" {
				return nil, fmt.Errorf("@Field(%s) selection method name must be a non-empty string", field.Name)
			}
			methodName := trimmed
			if literal, err := parser.ParseJSStringLiteral(trimmed); err == nil {
				methodName = strings.TrimSpace(literal)
			}
			if methodName == "" {
				return nil, fmt.Errorf("@Field(%s) selection method name must be a non-empty string", field.Name)
			}
			// Arrow / function source text from ObjectLiteral encoding → dynamic without method name.
			if strings.Contains(methodName, "=>") || strings.HasPrefix(methodName, "function") {
				spec.Structural.SelectionKind = "dynamic"
			} else {
				spec.Structural.SelectionKind = "dynamic"
				spec.Structural.SelectionMethod = methodName
			}
		default:
			return nil, fmt.Errorf("@Field(%s) selection must be an array, method name string, or callable", field.Name)
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
	switch raw := options["index"].(type) {
	case bool:
		if hints.Indexed == nil {
			hints.Indexed = toBoolPtr(raw)
		}
	case string:
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			hints.Index = toStringPtr(trimmed)
			if hints.Indexed == nil {
				hints.Indexed = toBoolPtr(true)
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
	translate := false
	if v, ok := options["translate"].(bool); ok && v {
		translate = true
		spec.Structural.Translate = toBoolPtr(true)
	}
	companyDependent := false
	if v, ok := options["companyDependent"].(bool); ok && v {
		companyDependent = true
		spec.Structural.CompanyDependent = toBoolPtr(true)
	}
	if v, ok := options["copy"].(bool); ok && !v {
		spec.Structural.Copy = toBoolPtr(false)
	} else if companyDependent {
		// Default copy:false for companyDependent when omitted (matches TS decorator).
		if _, ok := options["copy"]; !ok {
			spec.Structural.Copy = toBoolPtr(false)
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

	columnType := resolveColumnType(fieldType)
	if translate || companyDependent {
		// Logical type stays; physical storage is JSON/JSONB map.
		columnType = "jsonobject"
	}
	spec.Structural.ColumnType = columnType

	spec.Migration = meta.IrFieldMigrationDecision{
		StorageKind:        "physical",
		ShouldCreateColumn: true,
		ResolvedColumnType: columnType,
		ReasonCode:         "FIELD_DEFAULT",
	}
	if translate {
		spec.Migration.ReasonCode = "TRANSLATE_LANG_MAP"
	}
	if companyDependent {
		spec.Migration.ReasonCode = "COMPANY_DEPENDENT_MAP"
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
	if translate {
		if fieldType != "char" && fieldType != "varchar" && fieldType != "text" {
			spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
				Code:     "CONFLICT_TRANSLATE_FIELD_TYPE",
				Severity: "error",
				Message:  "translate is only supported on char/varchar/text fields",
			})
		}
		uniqueOn := hints.Unique != nil && *hints.Unique
		uniqueIndexOn := (hints.UniqueIndexEnabled != nil && *hints.UniqueIndexEnabled) ||
			(hints.UniqueIndex != nil && strings.TrimSpace(*hints.UniqueIndex) != "")
		if uniqueOn || uniqueIndexOn {
			spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
				Code:     "CONFLICT_TRANSLATE_UNIQUE",
				Severity: "error",
				Message:  "translate cannot be combined with unique/uniqueIndex",
			})
		}
		indexKind := ""
		if hints.Index != nil {
			indexKind = strings.TrimSpace(*hints.Index)
		}
		btreeIndexed := hints.Indexed != nil && *hints.Indexed && !strings.EqualFold(indexKind, "trigram")
		namedNonTrigram := indexKind != "" && !strings.EqualFold(indexKind, "trigram")
		if btreeIndexed || namedNonTrigram {
			spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
				Code:     "CONFLICT_TRANSLATE_INDEX",
				Severity: "error",
				Message:  "translate only supports index: 'trigram' (or omit index)",
			})
		}
	}
	if translate && companyDependent {
		spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
			Code:     "CONFLICT_TRANSLATE_COMPANY_DEPENDENT",
			Severity: "error",
			Message:  "cannot combine translate and companyDependent",
		})
	}
	if companyDependent {
		allowed := map[string]struct{}{
			"char": {}, "varchar": {}, "text": {}, "boolean": {}, "integer": {}, "float": {},
			"decimal": {}, "monetary": {}, "date": {}, "datetime": {}, "selection": {},
			"ManyToOne": {}, "ManyToOneRef": {},
		}
		if _, ok := allowed[fieldType]; !ok {
			spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
				Code:     "CONFLICT_COMPANY_DEPENDENT_FIELD_TYPE",
				Severity: "error",
				Message:  "companyDependent is not supported on this field type",
			})
		}
		uniqueOn := hints.Unique != nil && *hints.Unique
		uniqueIndexOn := (hints.UniqueIndexEnabled != nil && *hints.UniqueIndexEnabled) ||
			(hints.UniqueIndex != nil && strings.TrimSpace(*hints.UniqueIndex) != "")
		if uniqueOn || uniqueIndexOn {
			spec.Diagnostics = append(spec.Diagnostics, meta.IrFieldDiagnostic{
				Code:     "CONFLICT_COMPANY_DEPENDENT_UNIQUE",
				Severity: "error",
				Message:  "companyDependent cannot be combined with unique/uniqueIndex",
			})
		}
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
