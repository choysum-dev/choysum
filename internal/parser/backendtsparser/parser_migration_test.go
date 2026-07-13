// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
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
		cfg: &config.Config{ModulesPath: "/virtual/modules"},
	}
}

func TestTsParser_ParseModelExtendsProperty_IgnoresNonDefaultExportClass(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/core", ApplicationStr: "core"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/core/service/database/driver/compiler.ts"
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
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/user.ts"
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
	if r.ModelExtendsProperty.ModuleSpecPath != "/virtual/modules/test/service/base" {
		t.Fatalf("unexpected extends module spec path: %s", r.ModelExtendsProperty.ModuleSpecPath)
	}
	if r.Model == nil || r.Model.RawExtends != "/virtual/modules/test/service/base.ts" {
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
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/department.ts"
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
	if r.Model.RawExtends != "/virtual/modules/test/service/base.ts" {
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
	if parentPath.Decorators[0].ModuleSpecPath != filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, "core", "service", "orm", "decorator", "field") {
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
	module := &meta.IrModule{Path: "/virtual/modules/core", ApplicationStr: "core"}
	p := NewTsParser(runtimeScope, module)

	path := filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, "core", "service", "orm", "metadata", "field.ts")
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

func TestTsParser_ParseModelBuildsResolvedFieldSpecFromNewDecorators(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/demo.ts"
	content := `import { Model, Field, Compute, Search } from '../../core/service';
import BaseModel from './base';

@Model('Demo')
export default class Demo extends BaseModel {
	@Field({ type: 'varchar', size: 64, required: true, indexed: true })
  public Name: string

  @Field({ type: 'varchar', related: { path: 'PartnerId.Name', store: true, deps: ['PartnerId', 'PartnerId.Name'] } })
  public PartnerName: string

  @Compute<Demo>('PartnerName', { deps: ['Name'], store: false, searchable: true, runAs: 'sudo' })
  computePartnerName() {
    return this.Name
  }

  @Search<Demo>('PartnerName')
  searchPartnerName() {
    return ['Name', '=', 'A']
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil {
		t.Fatal("expected parsed model")
	}

	fieldByName := map[string]*meta.IrField{}
	for _, field := range r.Model.Fields {
		fieldByName[field.Name] = field
	}

	nameSpec, err := fieldByName["Name"].GetResolvedSpec()
	if err != nil {
		t.Fatalf("parse Name resolved spec failed: %v", err)
	}
	if nameSpec == nil {
		t.Fatal("expected Name resolved spec")
	}
	if nameSpec.Structural.FieldType != "varchar" || nameSpec.Structural.StorageHints == nil || nameSpec.Structural.StorageHints.Size == nil || *nameSpec.Structural.StorageHints.Size != 64 {
		t.Fatalf("unexpected Name resolved structural spec: %+v", nameSpec.Structural)
	}
	if nameSpec.Migration.ShouldCreateColumn != true || nameSpec.Migration.ReasonCode != "FIELD_DEFAULT" {
		t.Fatalf("unexpected Name migration decision: %+v", nameSpec.Migration)
	}

	partnerSpec, err := fieldByName["PartnerName"].GetResolvedSpec()
	if err != nil {
		t.Fatalf("parse PartnerName resolved spec failed: %v", err)
	}
	if partnerSpec == nil {
		t.Fatal("expected PartnerName resolved spec")
	}
	if partnerSpec.Behavior.Compute == nil || partnerSpec.Behavior.Compute.Method != "computePartnerName" || partnerSpec.Behavior.Compute.Store != false {
		t.Fatalf("unexpected PartnerName compute behavior: %+v", partnerSpec.Behavior)
	}
	if partnerSpec.Behavior.Search == nil || partnerSpec.Behavior.Search.Method != "searchPartnerName" {
		t.Fatalf("unexpected PartnerName search behavior: %+v", partnerSpec.Behavior)
	}
	if partnerSpec.Migration.ShouldCreateColumn != false || partnerSpec.Migration.ReasonCode != "COMPUTE_STORE_FALSE" {
		t.Fatalf("unexpected PartnerName migration decision: %+v", partnerSpec.Migration)
	}
	if partnerSpec.Resolved.RunAs.Value == nil || *partnerSpec.Resolved.RunAs.Value != "sudo" {
		t.Fatalf("unexpected PartnerName runAs resolution: %+v", partnerSpec.Resolved.RunAs)
	}
}

func TestTsParser_ParseModelRejectsLegacyFieldSyntax(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/legacy.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('Legacy')
export default class Legacy extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  public Name: string
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil {
		t.Fatal("expected parser to reject legacy field syntax")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "FIELD_LEGACY_SYNTAX_FORBIDDEN") {
		t.Fatalf("unexpected parser error: %v", err)
	}
}

func TestTsParser_ParseModelEmitsConflictDiagnosticsInResolvedSpec(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/conflict.ts"
	content := `import { Model, Field, Compute, SqlCompute } from '../../core/service';
import BaseModel from './base';

@Model('ConflictModel')
export default class ConflictModel extends BaseModel {
	@Field({ type: 'varchar', size: 64 })
  public DisplayName: string

  @Compute<ConflictModel>('DisplayName', { deps: ['DisplayName'], store: true })
  computeDisplayName() {
    return this.DisplayName
  }

  @SqlCompute<ConflictModel>('DisplayName')
  sqlDisplayName() {
    return this.$sql.field(ConflictModel, 'DisplayName')
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fieldByName := map[string]*meta.IrField{}
	for _, field := range r.Model.Fields {
		fieldByName[field.Name] = field
	}
	spec, err := fieldByName["DisplayName"].GetResolvedSpec()
	if err != nil {
		t.Fatalf("parse DisplayName resolved spec failed: %v", err)
	}
	if spec == nil {
		t.Fatal("expected resolved spec for DisplayName")
	}
	seen := false
	for _, d := range spec.Diagnostics {
		if d.Code == "CONFLICT_COMPUTE_SQLCOMPUTE" && d.Severity == "error" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("expected CONFLICT_COMPUTE_SQLCOMPUTE diagnostic, got %+v", spec.Diagnostics)
	}
}
