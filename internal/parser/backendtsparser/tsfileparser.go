// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type tsFileParser struct {
	*parser.TsParser
	runtimeScope scope.Scope
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
	var companyScopedOpt *bool
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
			if companyScopedOpt == nil {
				if cs, ok := opts["companyScoped"].(bool); ok {
					v := cs
					companyScopedOpt = &v
				}
			}
			if pf, ok := opts["parentField"].(string); ok && pf != "" {
				parentFieldName = pf
				// don't break here; continue scanning to also capture companyScoped if present
			}
		}
		if parentFieldName != "" && companyScopedOpt != nil {
			break
		}
	}
	if companyScopedOpt != nil {
		model.CompanyScoped = companyScopedOpt
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
			// Equivalent to @Field({ type:'varchar', column:{ size:1000, index:true } }).
			argObj := map[string]any{
				"type": "varchar",
				"column": map[string]any{
					"size":  1000,
					"index": true,
				},
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

	// services
	if len(class.MemberMethods) > 0 {
		model.Services = make([]*meta.IrService, 0)
		for _, memberMethod := range class.MemberMethods {
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
	addons_path := runtimeOptionsFromScope(p.runtimeScope).addonsPath
	// Keep historical skip behavior for the metadata and onchange type entrypoints.
	// addons/core/service/orm/metadata/field.ts
	// addons/core/service/runtime/onchange/types.ts
	if p.Path == filepath.Join(addons_path, "core", "service", "orm", "metadata", "field.ts") || p.Path == filepath.Join(addons_path, "core", "service", "runtime", "onchange", "types.ts") {
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
