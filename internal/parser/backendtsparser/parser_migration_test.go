// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type backendParserTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *backendParserTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *backendParserTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *backendParserTestScope) Session() *scope.Session { return nil }
func (e *backendParserTestScope) WithContext(ctx context.Context) scope.Scope {
	return &backendParserTestScope{ctx: ctx, cfg: e.cfg}
}
func (e *backendParserTestScope) Context() context.Context { return e.ctx }
func (e *backendParserTestScope) Logger() *slog.Logger     { return slog.Default() }
func (e *backendParserTestScope) Config() *config.Config   { return e.cfg }
func (e *backendParserTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newBackendParserTestScope() scope.Scope {
	return &backendParserTestScope{
		ctx: context.Background(),
		cfg: &config.Config{AddonsPath: "/virtual/addons"},
	}
}

func TestTsParser_ParseModelExtendsProperty_IgnoresNonDefaultExportClass(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/core", ApplicationStr: "core"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/addons/core/service/database/driver/compiler.ts"
	content := `import { PostgresQueryCompiler } from "kysely";

export class ChoysumPostgresQueryCompiler extends PostgresQueryCompiler {}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.ModelExtendsProperty != nil {
		t.Fatalf("expected no model extends property for non-default export class, got %+v", r.ModelExtendsProperty)
	}
}

func TestTsParser_ParseModelExtendsProperty_DefaultExportAssignmentUsesDefaultReference(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/addons/test/service/user.ts"
	content := `import BaseModel from './base';

class User extends BaseModel {}

export default User;
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.ModelExtendsProperty == nil {
		t.Fatalf("expected model extends property")
	}
	if r.ModelExtendsProperty.ReferenceIdent != "default" {
		t.Fatalf("expected extends reference ident default, got %s", r.ModelExtendsProperty.ReferenceIdent)
	}
	if r.ModelExtendsProperty.ModuleSpecPath != "/virtual/addons/test/service/base" {
		t.Fatalf("unexpected extends module spec path: %s", r.ModelExtendsProperty.ModuleSpecPath)
	}
	if r.Model == nil || r.Model.RawExtends != "/virtual/addons/test/service/base.ts" {
		t.Fatalf("unexpected model raw extends: %+v", r.Model)
	}
}

func TestGetProtoTypeFromTsType(t *testing.T) {
	tests := map[string]string{
		"string":          "string",
		"number":          "double",
		"boolean":         "bool",
		"void":            "google.protobuf.Empty",
		"Promise<number>": "double",
		"CustomType":      "google.protobuf.Value",
	}

	for input, want := range tests {
		if got := getProtoTypeFromTsType(input); got != want {
			t.Fatalf("getProtoTypeFromTsType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTsParser_ParseModelAddsParentPathAndServices(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/addons/test/service/department.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('Department', { companyScoped: true, parentField: 'ParentId' })
export default class Department extends BaseModel {
  @Field({ type: 'varchar' })
  public Name: string

  private static InternalCode: string

	  public static async FindOne(name: string, enabled: boolean): Promise<number> {
    return 1
  }

	  public static async remove(): Promise<void> {}

	  private static async Hidden(): Promise<void> {}

	  public async Remove(): Promise<void> {}
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil {
		t.Fatal("expected parsed model")
	}
	if r.Model.Name != "Department" || r.Model.ClassName != "Department" {
		t.Fatalf("unexpected model identity: %+v", r.Model)
	}
	if r.Model.CompanyScoped == nil || !*r.Model.CompanyScoped {
		t.Fatalf("expected companyScoped=true, got %+v", r.Model.CompanyScoped)
	}
	if r.Model.RawExtends != "/virtual/addons/test/service/base.ts" {
		t.Fatalf("unexpected raw extends: %s", r.Model.RawExtends)
	}
	if r.ModelExtendsProperty == nil || r.ModelExtendsProperty.ReferenceIdent != "default" {
		t.Fatalf("unexpected model extends property: %+v", r.ModelExtendsProperty)
	}

	fieldByName := map[string]*meta.IrField{}
	for _, field := range r.Model.Fields {
		fieldByName[field.Name] = field
	}
	if got := fieldByName["Name"]; got == nil || got.TsTypeAnnotation != "string" || len(got.Decorators) != 1 || got.Decorators[0].Name != "Field" {
		t.Fatalf("unexpected Name field: %+v", got)
	}
	parentPath := fieldByName["ParentPath"]
	if parentPath == nil || len(parentPath.Decorators) != 1 {
		t.Fatalf("expected synthesized ParentPath field, got %+v", parentPath)
	}
	if parentPath.Decorators[0].ModuleSpecPath != filepath.Join(runtimeOptionsFromScope(runtimeScope).addonsPath, "core", "service", "orm", "decorator", "field") {
		t.Fatalf("unexpected ParentPath decorator module: %+v", parentPath.Decorators[0])
	}
	if _, exists := fieldByName["InternalCode"]; exists {
		t.Fatalf("expected private static field to be filtered out, got %+v", r.Model.Fields)
	}

	if len(r.Model.Services) != 1 {
		t.Fatalf("expected one convention service, got %+v", r.Model.Services)
	}
	findOne := r.Model.Services[0]
	if findOne.Name != "FindOne" || findOne.ProtobufType != "double" || len(findOne.Parameters) != 2 {
		t.Fatalf("unexpected FindOne service: %+v", findOne)
	}
	if findOne.Parameters[0].ProtobufType != "string" || findOne.Parameters[1].ProtobufType != "bool" {
		t.Fatalf("unexpected FindOne parameters: %+v", findOne.Parameters)
	}
}

func TestTsParser_ParseSkipsCompatibilityPath(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/core", ApplicationStr: "core"}
	p := NewTsParser(runtimeScope, module)

	path := filepath.Join(runtimeOptionsFromScope(runtimeScope).addonsPath, "core", "service", "orm", "metadata", "field.ts")
	r, err := p.Parse(nil, path, "export default class Ignored {}")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Path != path || r.RawContent == "" {
		t.Fatalf("unexpected skipped parser result: %+v", r)
	}
	if r.Model != nil || r.Imports != nil || r.Exports != nil {
		t.Fatalf("expected compatibility skip to bypass deeper parsing, got %+v", r)
	}
}
