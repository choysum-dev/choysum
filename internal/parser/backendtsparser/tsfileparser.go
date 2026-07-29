// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type tsFileParser struct {
	*parser.TsParser
	runtimeScope      scope.Scope
	ownerModule       string
	referenceOutput   bool
	referenceScope    string
	translateBindings map[string]parser.TranslateBinding
}

var (
	referenceScopePattern    = regexp.MustCompile(`\bscope\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
	referencePathPattern     = regexp.MustCompile(`\bpath\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
	referenceLocationPattern = regexp.MustCompile(`\blocation\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
)

// ensureSynthesizedParentPathTitle attaches the core BaseModel.fields TermReference used by
// runtime MetadataStorage. Plain "Parent Path" in @Field options does not create stringText
// during resolve (owner module would also be wrong); OSearch reads static web store metadata
// only, so codegen must emit module=core stringText or the picker shows the raw prop name.
func ensureSynthesizedParentPathTitle(spec *meta.IrFieldResolvedSpec) {
	if spec == nil || spec.FieldName != "ParentPath" {
		return
	}
	title := strings.TrimSpace(spec.Structural.String)
	if title == "" {
		title = "Parent Path"
		spec.Structural.String = title
	}
	if spec.Structural.StringText != nil || title != "Parent Path" {
		return
	}
	ref := meta.NewTermReference("core", "core.model.BaseModel.fields", "Parent Path", "literal")
	spec.Structural.StringText = &ref
}

func parseFactoryStringOption(options string, pattern *regexp.Regexp) string {
	match := pattern.FindStringSubmatch(options)
	if len(match) != 2 {
		return ""
	}
	raw := strings.TrimSpace(match[1])
	if strings.HasPrefix(raw, "`") && strings.HasSuffix(raw, "`") {
		return strings.TrimSuffix(strings.TrimPrefix(raw, "`"), "`")
	}
	parsed, err := parser.ParseJSStringLiteral(raw)
	if err != nil {
		return ""
	}
	return parsed
}

func (p *tsFileParser) detectReferenceFactory() {
	// Reuse the shared destructuring parser so aliases and multi-property
	// bindings (e.g. const { _t: t, locale } = createTranslate(...)) work.
	p.translateBindings = parser.ParseTranslateBindings(p.Content)
	for _, binding := range p.translateBindings {
		if !binding.ReferenceOutput {
			continue
		}
		p.referenceOutput = true
		p.referenceScope = binding.DefaultScope
		return
	}
}

func getProtoTypeFromTsType(tsType string) string {
	tsType = strings.TrimSpace(tsType)
	if strings.HasPrefix(tsType, "Promise<") && strings.HasSuffix(tsType, ">") {
		tsType = tsType[8 : len(tsType)-1]
	}
	switch tsType {
	case "string":
		return "string"
	case "number":
		return "double"
	case "boolean":
		return "bool"
	case "void":
		return "google.protobuf.Empty"
	default:
		return "google.protobuf.Value"
	}
}

func (p *tsFileParser) parseModel() (*meta.IrModel, *parser.Class, *parser.PropertyNode, error) {
	class, err := p.ParseClassNode(nil, nil)
	if err != nil {
		return nil, nil, nil, xfmt.Errorf("failed to parse class: %w", err)
	}

	if class == nil {
		return nil, nil, nil, nil
	}

	model := &meta.IrModel{
		ClassName: class.Name,
		Name:      class.Name,
		Abstract:  class.Abstract,
		Path:      p.Path,
	}

	if len(class.Decorators) > 0 {
		model.Decorators = make([]*meta.IrDecorator, 0)
		for _, d := range class.Decorators {
			decorator := &meta.IrDecorator{
				Name:           d.Name,
				ModuleSpecPath: d.ModuleSpecPath,
				ReferenceIdent: d.ReferenceIdent,
			}
			if len(d.Arguments) > 0 {
				decorator.Arguments = make([]*meta.IrArgument, 0)
				for _, arg := range d.Arguments {
					choysumMetaArgument := &meta.IrArgument{
						Type:           arg.Type,
						Value:          arg.Value,
						ReferenceIdent: arg.ReferenceIdent,
						ModuleSpecPath: arg.ModuleSpecPath,
					}
					decorator.Arguments = append(decorator.Arguments, choysumMetaArgument)
				}
			}
			model.Decorators = append(model.Decorators, decorator)
		}
	}

	// extends
	var modelExtendsProperty *parser.PropertyNode
	if class.Extends != nil {
		model.RawExtends = class.Extends.ModuleSpecPath + ".ts"
		modelExtendsProperty = &parser.PropertyNode{
			ReferenceIdent: class.Extends.ReferenceIdent,
			ModuleSpecPath: class.Extends.ModuleSpecPath,
			Text:           class.Extends.Text,
			Start:          class.Extends.Start,
			End:            class.Extends.End,
			Line:           class.Extends.Line,
			Column:         class.Extends.Column,
		}
	}

	// fields
	if len(class.MemberVars) > 0 {
		model.Fields = make([]*meta.IrField, 0)
		for _, memberVar := range class.MemberVars {
			if memberVar.AccessibilityModifier != "public" && memberVar.IsStatic {
				continue
			}
			field := &meta.IrField{
				Name:                  memberVar.Name,
				TsTypeAnnotation:      strings.Replace(memberVar.TypeAnnotation, "'", "\"", -1), // Normalize to double quotes for later JSON parsing.
				TsTypeReference:       memberVar.TsTypeReference,
				ReferenceIdent:        memberVar.ReferenceIdent,
				ModuleSpecPath:        memberVar.ModuleSpecPath,
				AccessibilityModifier: memberVar.AccessibilityModifier,
				IsStatic:              memberVar.IsStatic,
				IsReadonly:            memberVar.IsReadonly,
			}
			if len(memberVar.Decorators) > 0 {
				field.Decorators = make([]*meta.IrDecorator, 0)
				for _, d := range memberVar.Decorators {
					decorator := &meta.IrDecorator{
						Name:           d.Name,
						ModuleSpecPath: d.ModuleSpecPath,
						ReferenceIdent: d.ReferenceIdent,
					}
					if len(d.Arguments) > 0 {
						decorator.Arguments = make([]*meta.IrArgument, 0)
						for _, arg := range d.Arguments {
							choysumMetaArgument := &meta.IrArgument{
								Type:           arg.Type,
								Value:          arg.Value,
								ReferenceIdent: arg.ReferenceIdent,
								ModuleSpecPath: arg.ModuleSpecPath,
							}
							decorator.Arguments = append(decorator.Arguments, choysumMetaArgument)
						}
					}
					field.Decorators = append(field.Decorators, decorator)
				}
			}
			model.Fields = append(model.Fields, field)
		}
	}

	// Parse-stage support for @Model('Name', { ...options }) object-literal arguments.
	parentFieldName := ""
	var companyFieldOpt *string
	for _, d := range class.Decorators {
		if d.Name != "Model" || len(d.Arguments) == 0 {
			continue
		}
		// Find any ObjectLiteral argument, usually the second one.
		for _, arg := range d.Arguments {
			if arg.Type != "ObjectLiteral" || arg.Value == "" {
				continue
			}
			var opts map[string]any
			if err := json.Unmarshal([]byte(arg.Value), &opts); err != nil {
				continue
			}
			if companyFieldOpt == nil {
				if cf, ok := opts["companyField"].(string); ok {
					trimmed := strings.TrimSpace(cf)
					if trimmed != "" {
						v := trimmed
						companyFieldOpt = &v
					}
				}
			}
			if pf, ok := opts["parentField"].(string); ok && pf != "" {
				parentFieldName = pf
				// don't break here; continue scanning to also capture companyField if present
			}
		}
		if parentFieldName != "" && companyFieldOpt != nil {
			break
		}
	}
	if companyFieldOpt != nil {
		model.CompanyField = companyFieldOpt
	}

	// Synthesize ParentPath when parentField is declared and ParentPath is absent.
	if parentFieldName != "" {
		exists := false
		for _, f := range model.Fields {
			if f.Name == "ParentPath" {
				exists = true
				break
			}
		}
		if !exists {
			// Equivalent to runtime MetadataStorage ParentPath injection:
			// @Field({ type:'varchar', size:1000, indexed:true, string: _lt('Parent Path', { scope: 'core.model.BaseModel.fields' }) })
			argObj := map[string]any{
				"type":    "varchar",
				"size":    1000,
				"indexed": true,
				"string":  "Parent Path",
			}
			argBytes, _ := json.Marshal(argObj)

			// Fill decorator module metadata so build plugins do not filter it out.
			fieldDecoratorModuleSpec, fieldDecoratorIdent := meta.FieldDecoratorModuleSpec(p.runtimeScope)

			field := &meta.IrField{
				Name: "ParentPath",
				Decorators: []*meta.IrDecorator{
					{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec, // Key: module path.
						ReferenceIdent: fieldDecoratorIdent,      // Key: exported identifier.
						Arguments: []*meta.IrArgument{
							{
								Type:  "ObjectLiteral",
								Value: string(argBytes),
							},
						},
					},
				},
			}
			model.Fields = append(model.Fields, field)
		}
	}

	behaviorBindings, behaviorDiagnostics, err := collectFieldBehaviorBindings(class.MemberMethods)
	if err != nil {
		return nil, nil, nil, xfmt.Errorf("failed to collect field behavior decorators: %w", err)
	}
	fieldNames := make(map[string]struct{}, len(model.Fields))
	for _, field := range model.Fields {
		if field == nil {
			continue
		}
		fieldNames[field.Name] = struct{}{}
	}
	for bindingField := range behaviorBindings {
		if _, ok := fieldNames[bindingField]; ok {
			continue
		}
		return nil, nil, nil, xfmt.Errorf("orphan behavior decorator binding for unknown field: %s", bindingField)
	}
	for _, field := range model.Fields {
		if field == nil {
			continue
		}
		binding := behaviorBindings[field.Name]
		diagnostics := behaviorDiagnostics[field.Name]
		resolvedSpec, err := buildFieldResolvedSpec(field, binding, diagnostics, p.ownerModule, p.referenceScope, p.translateBindings)
		if err != nil {
			return nil, nil, nil, xfmt.Errorf("failed to resolve field %s: %w", field.Name, err)
		}
		if resolvedSpec == nil {
			continue
		}
		ensureSynthesizedParentPathTitle(resolvedSpec)
		if err := field.SetResolvedSpec(resolvedSpec); err != nil {
			return nil, nil, nil, xfmt.Errorf("failed to persist resolved field %s spec: %w", field.Name, err)
		}
		applyResolvedSpecToLegacyField(field, resolvedSpec)
	}

	// services
	if len(class.MemberMethods) > 0 {
		model.Services = make([]*meta.IrService, 0)
		for _, memberMethod := range class.MemberMethods {
			if !memberMethod.IsAsync {
				continue
			}
			if !meta.IsConventionalModelService(memberMethod.AccessibilityModifier, memberMethod.IsStatic, memberMethod.Name) {
				continue
			}

			service := &meta.IrService{
				Name:                  memberMethod.Name,
				TsTypeAnnotation:      memberMethod.TypeAnnotation,
				ProtobufType:          getProtoTypeFromTsType(memberMethod.TypeAnnotation),
				AccessibilityModifier: memberMethod.AccessibilityModifier,
				IsStatic:              memberMethod.IsStatic,
			}

			if len(memberMethod.Decorators) > 0 {
				service.Decorators = make([]*meta.IrDecorator, 0)
				for _, d := range memberMethod.Decorators {
					decorator := &meta.IrDecorator{
						Name:           d.Name,
						ReferenceIdent: d.ReferenceIdent,
						ModuleSpecPath: d.ModuleSpecPath,
					}
					if len(d.Arguments) > 0 {
						decorator.Arguments = make([]*meta.IrArgument, 0)
						for _, arg := range d.Arguments {
							choysumMetaArgument := &meta.IrArgument{
								Type:           arg.Type,
								Value:          arg.Value,
								ReferenceIdent: arg.ReferenceIdent,
								ModuleSpecPath: arg.ModuleSpecPath,
							}
							decorator.Arguments = append(decorator.Arguments, choysumMetaArgument)
						}
					}
					service.Decorators = append(service.Decorators, decorator)
				}
			}

			if len(memberMethod.TypeParameters) > 0 {
				service.TypeParameters = make([]*meta.IrTypeParameter, 0)
				for _, typeParam := range memberMethod.TypeParameters {
					typeParameter := &meta.IrTypeParameter{
						Name:           typeParam.Name,
						ModuleSpecPath: typeParam.ModuleSpecPath,
						ReferenceIdent: typeParam.ReferenceIdent,
					}
					service.TypeParameters = append(service.TypeParameters, typeParameter)
				}
			}

			if len(memberMethod.Parameters) > 0 {
				service.Parameters = make([]*meta.IrParameter, 0)
				for _, param := range memberMethod.Parameters {
					parameter := &meta.IrParameter{
						Name:             param.Name,
						TsTypeAnnotation: param.TypeAnnotation,
						ProtobufType:     getProtoTypeFromTsType(param.TypeAnnotation),
					}
					service.Parameters = append(service.Parameters, parameter)
				}
			}

			model.Services = append(model.Services, service)
		}
	}

	return model, class, modelExtendsProperty, nil
}

func (p *tsFileParser) parse() (*parser.ParserResult, error) {
	p.detectReferenceFactory()
	modules_path := runtimeOptionsFromScope(p.runtimeScope).modulesPath
	// Keep historical skip behavior for the metadata and onchange type entrypoints.
	// modules/core/service/orm/metadata/field.ts
	// modules/core/service/runtime/onchange/types.ts
	if p.Path == filepath.Join(modules_path, "core", "service", "orm", "metadata", "field.ts") || p.Path == filepath.Join(modules_path, "core", "service", "runtime", "onchange", "types.ts") {
		return &parser.ParserResult{
			Path:       p.Path,
			RawContent: p.Content,
		}, nil
	}

	parserResult := &parser.ParserResult{
		Path:       p.Path,
		RawContent: p.Content,
	}

	err := p.ParseImport(nil)
	if err != nil {
		return nil, xfmt.Errorf("failed to parse import: %w", err)
	}
	parserResult.Imports = p.ImportsMap

	err = p.ParseExport(nil)
	if err != nil {
		return nil, xfmt.Errorf("failed to parse export: %w", err)
	}
	parserResult.Exports = p.ExportsMap

	model, class, modelExtendsProperty, err := p.parseModel()
	if err != nil {
		return nil, xfmt.Errorf("failed to parse model: %w", err)
	}
	parserResult.Model = model
	parserResult.ModelClassNode = class
	parserResult.ModelExtendsProperty = modelExtendsProperty

	return parserResult, nil
}
