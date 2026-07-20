// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"

	"github.com/choysum-dev/choysum/internal/parser"
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

func TestTsParser_ParseModelRejectsBehaviorBindingsWithoutFieldDecorator(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/missing_field_decorator.ts"
	content := `import { Model, Compute } from '../../core/service';
import BaseModel from './base';

@Model('MissingFieldDecorator')
export default class MissingFieldDecorator extends BaseModel {
  public DisplayName: string

  @Compute<MissingFieldDecorator>('DisplayName', { deps: ['DisplayName'], store: false })
  computeDisplayName() {
    return this.DisplayName
  }
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil {
		t.Fatal("expected parser to reject behavior binding without @Field decorator")
	}
	if got := err.Error(); !strings.Contains(got, "missing @Field decorator") {
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
	if spec.Resolved.Store.Value != false || spec.Resolved.Store.Source != "@SqlCompute" {
		t.Fatalf("expected SqlCompute store resolution to win conflict, got %+v", spec.Resolved.Store)
	}
}

func TestTsParser_ParseModelResolvedSpecSkipsNilSelectionAndRelatedPath(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/nil_fields.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('NilFieldsModel')
export default class NilFieldsModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: null, label: 'IgnoredNilValue' },
      { value: 'active', label: null },
      { value: 'active', label: 'Active' }
    ]
  })
  public Status: string

  @Field({ type: 'varchar', related: { path: null, store: true, deps: ['PartnerId'] } })
  public DisplayName: string
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

	statusSpec, err := fieldByName["Status"].GetResolvedSpec()
	if err != nil {
		t.Fatalf("parse Status resolved spec failed: %v", err)
	}
	if statusSpec == nil {
		t.Fatal("expected Status resolved spec")
	}
	if len(statusSpec.Structural.Selection) != 1 || statusSpec.Structural.Selection[0].Value != "active" || statusSpec.Structural.Selection[0].Label != "Active" {
		t.Fatalf("unexpected selection entries: %+v", statusSpec.Structural.Selection)
	}

	displaySpec, err := fieldByName["DisplayName"].GetResolvedSpec()
	if err != nil {
		t.Fatalf("parse DisplayName resolved spec failed: %v", err)
	}
	if displaySpec == nil {
		t.Fatal("expected DisplayName resolved spec")
	}
	if displaySpec.Structural.Related != nil {
		t.Fatalf("expected related spec to be skipped when path is nil, got %+v", displaySpec.Structural.Related)
	}
}

func TestTsParser_PreservesFieldStringTermReference(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';
const { _lt } = createTranslate('demo', {
  scope: 'demo.model.Widget.fields',
});

@Model('FieldStringModel')
export default class FieldStringModel extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    string: _lt('Name')
  })
  public Name: string

  @Field({
    type: 'varchar',
    size: 40,
    string: 'Code'
  })
  public Code: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	byName := map[string]*meta.IrField{}
	for _, field := range r.Model.Fields {
		byName[field.Name] = field
	}
	nameField := byName["Name"]
	if nameField == nil {
		t.Fatal("expected Name field")
	}
	spec, err := nameField.GetResolvedSpec()
	if err != nil || spec == nil {
		t.Fatalf("get resolved spec: %v, %#v", err, spec)
	}
	if spec.Structural.String != "Name" || spec.Structural.StringText == nil {
		t.Fatalf("unexpected string metadata: %+v", spec.Structural)
	}
	if spec.Structural.StringText.Module != "demo" || spec.Structural.StringText.Scope != "demo.model.Widget.fields" || spec.Structural.StringText.Src != "Name" {
		t.Fatalf("unexpected stringText: %+v", spec.Structural.StringText)
	}
	if nameField.FieldString != "Name" || !strings.Contains(nameField.StringText, `"src":"Name"`) {
		t.Fatalf("IrField did not persist string/stringText: string=%q stringText=%s", nameField.FieldString, nameField.StringText)
	}
	codeField := byName["Code"]
	if codeField == nil || codeField.FieldString != "Code" || strings.TrimSpace(codeField.StringText) != "" {
		t.Fatalf("unexpected Code field string metadata: %#v", codeField)
	}
}

func TestTsParser_RejectsSelectionTextTranslateLabels(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';
const { _t, _lt } = createTranslate('demo');

@Model('SelectionReferenceModel')
export default class SelectionReferenceModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: _lt('Active', { scope: 'demo.model.status.active' }) },
      { value: 'archived', label: _t('Archived', { scope: 'demo.model.status.archived' }) }
    ]
  })
  public Status: string
}
`
	_, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err == nil || !strings.Contains(err.Error(), "FIELD_SELECTION_LABELTEXT_FORBIDDEN") {
		t.Fatalf("expected selection text _t label rejection, got: %v", err)
	}
}

func TestTsParser_AcceptsAliasedSelectionReferenceLabel(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';
const { _lt: translate } = createTranslate('demo', { scope: 'demo.model.status',
});

@Model('AliasedReferenceModel')
export default class AliasedReferenceModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: translate('Active') }
    ]
  })
  public Status: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(r.Model.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(r.Model.Fields))
	}
	var items []meta.IrFieldSelectionItem
	if err := json.Unmarshal([]byte(r.Model.Fields[0].Selection), &items); err != nil {
		t.Fatalf("unmarshal selection: %v", err)
	}
	if len(items) != 1 || items[0].Label != "Active" || items[0].LabelText == nil || items[0].LabelText.Src != "Active" {
		t.Fatalf("unexpected selection items: %#v", items)
	}
}

func TestTsParser_AcceptsSelectionLtCallLabels(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "base", Path: "/virtual/modules/base", ApplicationStr: "base"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';
const { _lt } = createTranslate('base');

@Model('Language')
export default class Language extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      {
        value: 'ltr',
        label: _lt('Left to right', {
          scope: 'base.Language.Direction.ltr'
        })
      }
    ]
  })
  public Direction: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/base/service/language.ts", content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var items []meta.IrFieldSelectionItem
	if err := json.Unmarshal([]byte(r.Model.Fields[0].Selection), &items); err != nil {
		t.Fatalf("unmarshal selection: %v", err)
	}
	if len(items) != 1 || items[0].Label != "Left to right" {
		t.Fatalf("unexpected label: %#v", items)
	}
	if items[0].LabelText == nil || items[0].LabelText.Src != "Left to right" {
		t.Fatalf("expected LabelText TermReference, got %#v", items[0].LabelText)
	}
	if items[0].LabelText.Scope != "base.Language.Direction.ltr" {
		t.Fatalf("unexpected LabelText scope: %#v", items[0].LabelText)
	}
}

func TestTsParser_AcceptsPlainStringSelectionLabels(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('PlainSelectionModel')
export default class PlainSelectionModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: 'Active' },
      { value: 'archived', label: 'Archived' }
    ]
  })
  public Status: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	spec, err := r.Model.Fields[0].GetResolvedSpec()
	if err != nil || spec == nil || len(spec.Structural.Selection) != 2 {
		t.Fatalf("get resolved spec: %v, %#v", err, spec)
	}
	if spec.Structural.Selection[0].Label != "Active" || spec.Structural.Selection[0].LabelText != nil {
		t.Fatalf("unexpected selection item: %+v", spec.Structural.Selection[0])
	}
	if spec.Structural.SelectionKind != "static" {
		t.Fatalf("expected static selectionKind, got %q", spec.Structural.SelectionKind)
	}
	if strings.Contains(r.Model.Fields[0].Selection, "labelText") {
		t.Fatalf("selection JSON must not contain labelText: %s", r.Model.Fields[0].Selection)
	}
}

func TestTsParser_DynamicSelectionMethodNameAndCallable(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('DynamicSelectionModel')
export default class DynamicSelectionModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: 'StatusOptions'
  })
  public Status: string

  @Field({
    type: 'selection',
    selection: () => [{ value: 'a', label: 'A' }]
  })
  public Mode: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	byName := map[string]*meta.IrField{}
	for _, field := range r.Model.Fields {
		byName[field.Name] = field
	}
	status := byName["Status"]
	if status == nil {
		t.Fatal("expected Status field")
	}
	statusSpec, err := status.GetResolvedSpec()
	if err != nil || statusSpec == nil {
		t.Fatalf("status spec: %v %#v", err, statusSpec)
	}
	if statusSpec.Structural.SelectionKind != "dynamic" || statusSpec.Structural.SelectionMethod != "StatusOptions" {
		t.Fatalf("unexpected Status dynamic meta: %+v", statusSpec.Structural)
	}
	if len(statusSpec.Structural.Selection) != 0 || strings.TrimSpace(status.Selection) != "" {
		t.Fatalf("dynamic Status must not inline selection: spec=%+v field=%q", statusSpec.Structural.Selection, status.Selection)
	}
	if status.SelectionKind != "dynamic" || status.SelectionMethod != "StatusOptions" {
		t.Fatalf("legacy field columns: kind=%q method=%q", status.SelectionKind, status.SelectionMethod)
	}

	mode := byName["Mode"]
	if mode == nil {
		t.Fatal("expected Mode field")
	}
	modeSpec, err := mode.GetResolvedSpec()
	if err != nil || modeSpec == nil {
		t.Fatalf("mode spec: %v %#v", err, modeSpec)
	}
	if modeSpec.Structural.SelectionKind != "dynamic" || modeSpec.Structural.SelectionMethod != "" {
		t.Fatalf("unexpected Mode dynamic meta: %+v", modeSpec.Structural)
	}
	if len(modeSpec.Structural.Selection) != 0 {
		t.Fatalf("callable Mode must not inline selection: %+v", modeSpec.Structural.Selection)
	}
}

func TestTsParser_FactoryModeIsNotRecognized(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';
const { _t } = createTranslate('demo', { mode: 'reference' });

@Model('LegacyModeSelectionModel')
export default class LegacyModeSelectionModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: _t('Active', { scope: 'demo.model.status.active' }) }
    ]
  })
  public Status: string
}
`
	_, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err == nil || !strings.Contains(err.Error(), "FIELD_SELECTION_LABELTEXT_FORBIDDEN") {
		t.Fatalf("expected translate-call selection label rejection, got: %v", err)
	}
}

func TestParseTermReferenceCall_BindingDefaultScopeWhenParserScopeEmpty(t *testing.T) {
	ref, ok := parseTermReferenceCall(`translate('Active')`, "demo", "", map[string]parser.TranslateBinding{
		"translate": {
			Module:          "other",
			DefaultScope:    "demo.model.from_binding",
			ReferenceOutput: true,
		},
	})
	if !ok || ref == nil {
		t.Fatalf("expected binding default scope: ok=%v ref=%#v", ok, ref)
	}
	if ref.Scope != "demo.model.from_binding" || ref.Module != "other" || ref.Src != "Active" {
		t.Fatalf("unexpected reference: %+v", ref)
	}

	// Known `_t` binding must not produce a term reference.
	if _, ok := parseTermReferenceCall(`_t('Archived', { scope: 'demo.x' })`, "demo", "demo.x", map[string]parser.TranslateBinding{
		"_t": {Module: "demo", ReferenceOutput: false},
	}); ok {
		t.Fatal("text _t should not parse as term reference")
	}
}

func TestTsParser_BareLtAndPathLocationSelectionLabels(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Name: "demo", Path: "/virtual/modules/demo", ApplicationStr: "demo"}
	p := NewTsParser(runtimeScope, module)
	content := `
import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('BareLtSelectionModel')
export default class BareLtSelectionModel extends BaseModel {
  @Field({
    type: 'selection',
    selection: [
      { value: 'a', label: _lt('Alpha', { scope: 'demo.model.alpha' }) },
      { value: 'b', label: _lt('Beta', { path: 'demo/model', location: 'beta' }) },
      { value: 'c', label: 'Gamma' }
    ]
  })
  public Status: string
}
`
	r, err := p.Parse(map[string]string{}, "/virtual/modules/demo/service/model.ts", content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var items []meta.IrFieldSelectionItem
	if err := json.Unmarshal([]byte(r.Model.Fields[0].Selection), &items); err != nil {
		t.Fatalf("unmarshal selection: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %#v", items)
	}
	if items[0].LabelText == nil || items[0].Label != "Alpha" {
		t.Fatalf("item0: %#v", items[0])
	}
	if items[1].LabelText == nil || items[1].Label != "Beta" {
		t.Fatalf("item1: %#v", items[1])
	}
	if items[2].LabelText != nil || items[2].Label != "Gamma" {
		t.Fatalf("bare string must not get LabelText: %#v", items[2])
	}
}

func TestTsParser_ParseModelResolvedSpecCoversRelationTypesAndMigrationDecisions(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/migration.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('MigrationModel')
export default class MigrationModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => BaseModel, inverseField: 'ParentId' } })
  public PartnerId: string

  @Field({ type: 'OneToMany', relation: { targetModel: () => BaseModel, inverseField: 'ParentId' } })
  public Lines: any

  @Field({ type: 'ManyToMany', relation: { targetModel: () => BaseModel, inverseField: 'Tags' } })
  public Tags: any

  @Field({ type: 'ManyToManyRef', relation: { targetModel: () => BaseModel } })
  public TagIds: any
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fieldByName := map[string]*meta.IrField{}
	for _, f := range r.Model.Fields {
		fieldByName[f.Name] = f
	}

	partnerSpec, err := fieldByName["PartnerId"].GetResolvedSpec()
	if err != nil || partnerSpec == nil {
		t.Fatalf("PartnerId resolved spec: err=%v spec=%v", err, partnerSpec)
	}
	if partnerSpec.Migration.ShouldCreateColumn != true || partnerSpec.Migration.ReasonCode != "FIELD_DEFAULT" || partnerSpec.Structural.ColumnType != "char" {
		t.Fatalf("unexpected PartnerId migration: %+v", partnerSpec.Migration)
	}

	linesSpec, err := fieldByName["Lines"].GetResolvedSpec()
	if err != nil || linesSpec == nil {
		t.Fatalf("Lines resolved spec: err=%v spec=%v", err, linesSpec)
	}
	if linesSpec.Migration.ShouldCreateColumn != false || linesSpec.Migration.ReasonCode != "RELATION_NON_COLUMN" {
		t.Fatalf("expected OneToMany non-column, got %+v", linesSpec.Migration)
	}

	tagsSpec, err := fieldByName["Tags"].GetResolvedSpec()
	if err != nil || tagsSpec == nil {
		t.Fatalf("Tags resolved spec: err=%v spec=%v", err, tagsSpec)
	}
	if tagsSpec.Migration.ShouldCreateColumn != false || tagsSpec.Migration.ReasonCode != "RELATION_NON_COLUMN" {
		t.Fatalf("expected ManyToMany non-column, got %+v", tagsSpec.Migration)
	}

	tagIdsSpec, err := fieldByName["TagIds"].GetResolvedSpec()
	if err != nil || tagIdsSpec == nil {
		t.Fatalf("TagIds resolved spec: err=%v spec=%v", err, tagIdsSpec)
	}
	if tagIdsSpec.Structural.ColumnType != "jsonobject" {
		t.Fatalf("expected ManyToManyRef columnType jsonobject, got %s", tagIdsSpec.Structural.ColumnType)
	}
}

func TestTsParser_ParseModelResolvedSpecCoversDiagnosticBranches(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/diag.ts"
	content := `import { Model, Field, Compute, Search, SqlCompute, Inverse } from '../../core/service';
import BaseModel from './base';

@Model('DiagModel')
export default class DiagModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  public PhysicalSearch: string

  @Search<DiagModel>('PhysicalSearch')
  searchPhysical() {
    return ['Name', '=', 'A']
  }

  @Field({ type: 'varchar' })
  public InverseOnly: string

  @Inverse<DiagModel>('InverseOnly')
  inverseOnly() {
    this.$inverse.value()
  }

  @Field({ type: 'varchar', related: { path: 'PartnerId.Name', store: true } })
  public SqlRelated: string

  @SqlCompute<DiagModel>('SqlRelated')
  sqlComputeRelated() {
    return this.$sql.field(DiagModel, 'SqlRelated')
  }

  @Field({ type: 'varchar', related: { path: 'PartnerId.Name', store: false } })
  public NonStoredWithInverse: string

  @Inverse<DiagModel>('NonStoredWithInverse')
  inverseNonStored() {
    this.$inverse.value()
  }

  @Field({ type: 'varchar', size: 64 })
  public StoreCompute: string

  @Compute<DiagModel>('StoreCompute', { deps: ['StoreCompute'], store: true })
  computeStore() {
    return this.StoreCompute
  }

  @Field({ type: 'varchar', size: 64, required: true })
  public RequiredVirtual: string

  @Compute<DiagModel>('RequiredVirtual', { deps: ['RequiredVirtual'], store: false })
  computeRequiredVirtual() {
    return this.RequiredVirtual
  }

  @Field({ type: 'varchar', size: 64, related: { path: 'PartnerId.Name', store: false } })
  public NonStoredRelated: string

  @Compute<DiagModel>('NonStoredRelated', { deps: ['NonStoredRelated'], store: false })
  computeNonStoredRelated() {
    return this.NonStoredRelated
  }

	@Field({ type: 'varchar', size: 64 })
	public AsyncVirtual: string

	@Compute<DiagModel>('AsyncVirtual', { deps: ['AsyncVirtual'], store: false })
	async computeAsyncVirtual() {
		return this.AsyncVirtual
	}
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fieldByName := map[string]*meta.IrField{}
	for _, f := range r.Model.Fields {
		fieldByName[f.Name] = f
	}

	// WARN_SEARCH_ON_PHYSICAL_FIELD
	physicalSearchSpec, _ := fieldByName["PhysicalSearch"].GetResolvedSpec()
	foundSearchWarn := false
	for _, d := range physicalSearchSpec.Diagnostics {
		if d.Code == "WARN_SEARCH_ON_PHYSICAL_FIELD" && d.Severity == "warning" {
			foundSearchWarn = true
		}
	}
	if !foundSearchWarn {
		t.Fatalf("expected WARN_SEARCH_ON_PHYSICAL_FIELD diagnostic, got %+v", physicalSearchSpec.Diagnostics)
	}

	// CONFLICT_INVERSE_WITHOUT_SOURCE
	inverseOnlySpec, _ := fieldByName["InverseOnly"].GetResolvedSpec()
	foundInvSrc := false
	for _, d := range inverseOnlySpec.Diagnostics {
		if d.Code == "CONFLICT_INVERSE_WITHOUT_SOURCE" {
			foundInvSrc = true
		}
	}
	if !foundInvSrc {
		t.Fatalf("expected CONFLICT_INVERSE_WITHOUT_SOURCE diagnostic, got %+v", inverseOnlySpec.Diagnostics)
	}

	// CONFLICT_SQLCOMPUTE_RELATED_STORE
	sqlRelatedSpec, _ := fieldByName["SqlRelated"].GetResolvedSpec()
	foundSqlRelStore := false
	for _, d := range sqlRelatedSpec.Diagnostics {
		if d.Code == "CONFLICT_SQLCOMPUTE_RELATED_STORE" {
			foundSqlRelStore = true
		}
	}
	if !foundSqlRelStore {
		t.Fatalf("expected CONFLICT_SQLCOMPUTE_RELATED_STORE diagnostic, got %+v", sqlRelatedSpec.Diagnostics)
	}

	// CONFLICT_INVERSE_ON_NON_STORED_RELATED
	nonStoredSpec, _ := fieldByName["NonStoredWithInverse"].GetResolvedSpec()
	if nonStoredSpec.Migration.ShouldCreateColumn != false || nonStoredSpec.Migration.ReasonCode != "RELATED_STORE_FALSE" {
		t.Fatalf("expected RELATED_STORE_FALSE migration, got %+v", nonStoredSpec.Migration)
	}
	foundNonStoredInv := false
	for _, d := range nonStoredSpec.Diagnostics {
		if d.Code == "CONFLICT_INVERSE_ON_NON_STORED_RELATED" {
			foundNonStoredInv = true
		}
	}
	if !foundNonStoredInv {
		t.Fatalf("expected CONFLICT_INVERSE_ON_NON_STORED_RELATED diagnostic, got %+v", nonStoredSpec.Diagnostics)
	}

	// COMPUTE_STORE_TRUE
	storeComputeSpec, _ := fieldByName["StoreCompute"].GetResolvedSpec()
	if storeComputeSpec.Migration.ReasonCode != "COMPUTE_STORE_TRUE" || storeComputeSpec.Migration.ShouldCreateColumn != true {
		t.Fatalf("expected COMPUTE_STORE_TRUE migration, got %+v", storeComputeSpec.Migration)
	}

	// CONFLICT_REQUIRED_VIRTUAL_COMPUTE
	requiredVirtualSpec, _ := fieldByName["RequiredVirtual"].GetResolvedSpec()
	foundReqVirtual := false
	for _, d := range requiredVirtualSpec.Diagnostics {
		if d.Code == "CONFLICT_REQUIRED_VIRTUAL_COMPUTE" {
			foundReqVirtual = true
		}
	}
	if !foundReqVirtual {
		t.Fatalf("expected CONFLICT_REQUIRED_VIRTUAL_COMPUTE diagnostic, got %+v", requiredVirtualSpec.Diagnostics)
	}

	// non-stored related with compute → related store propagation
	nonStoredRelatedSpec, _ := fieldByName["NonStoredRelated"].GetResolvedSpec()
	if nonStoredRelatedSpec.Resolved.Store.Value != false {
		t.Fatalf("expected non-stored related store=false, got %+v", nonStoredRelatedSpec.Resolved.Store)
	}
	if nonStoredRelatedSpec.Resolved.Store.Source != "@Compute" {
		t.Fatalf("expected Compute store source, got %s", nonStoredRelatedSpec.Resolved.Store.Source)
	}

	// ASYNC_VIRTUAL_COMPUTE
	asyncVirtualSpec, _ := fieldByName["AsyncVirtual"].GetResolvedSpec()
	foundAsyncVirtual := false
	for _, d := range asyncVirtualSpec.Diagnostics {
		if d.Code == "ASYNC_VIRTUAL_COMPUTE" {
			foundAsyncVirtual = true
		}
	}
	if !foundAsyncVirtual {
		t.Fatalf("expected ASYNC_VIRTUAL_COMPUTE diagnostic, got %+v", asyncVirtualSpec.Diagnostics)
	}
}

func TestTsParser_ParseModelResolvedSpecCoversStorageHintVariants(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/hints.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('HintsModel')
export default class HintsModel extends BaseModel {
  @Field({ type: 'varchar', size: 64, required: true, primaryKey: true, unique: true, uniqueIndex: 'uq_hints_name', index: 'idx_hints_name', default: 'default_value' })
  public Name: string

  @Field({ type: 'varchar', indexed: true })
  public IndexedName: string

  @Field({ type: 'decimal', precision: 12, scale: 4, default: 0 })
  public Amount: string

  @Field({ type: 'varchar', notNull: true })
  public NotNullName: string

  @Field({ type: 'varchar', index: 'idx_custom' })
  public CustomIndex: string

  @Field({ type: 'varchar', index: true })
  public BoolIndex: string

  @Field({ type: 'varchar', default: 42 })
  public NumDefault: string

  @Field({ type: 'varchar', checkConstraint: 'status in (draft,done)' })
  public CheckedStatus: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fieldByName := map[string]*meta.IrField{}
	for _, f := range r.Model.Fields {
		fieldByName[f.Name] = f
	}

	nameSpec, err := fieldByName["Name"].GetResolvedSpec()
	if err != nil || nameSpec == nil {
		t.Fatalf("Name resolved spec: err=%v", err)
	}
	h := nameSpec.Structural.StorageHints
	if h == nil {
		t.Fatal("expected storage hints")
	}
	if h.PrimaryKey == nil || !*h.PrimaryKey {
		t.Fatal("expected primaryKey=true")
	}
	if h.Unique == nil || !*h.Unique {
		t.Fatal("expected unique=true")
	}
	if h.UniqueIndex == nil || *h.UniqueIndex != "uq_hints_name" {
		t.Fatalf("expected uniqueIndex string, got %v", h.UniqueIndex)
	}
	if h.Index == nil || *h.Index != "idx_hints_name" {
		t.Fatalf("expected named index, got %v", h.Index)
	}
	if h.Indexed == nil || !*h.Indexed {
		t.Fatal("expected Indexed to be true when index string is set")
	}
	if h.Required == nil || !*h.Required {
		t.Fatal("expected required=true")
	}
	if h.Default == nil || *h.Default != "default_value" {
		t.Fatalf("expected default='default_value', got %v", h.Default)
	}

	amountSpec, _ := fieldByName["Amount"].GetResolvedSpec()
	ah := amountSpec.Structural.StorageHints
	if ah == nil || ah.Precision == nil || *ah.Precision != 12 || ah.Scale == nil || *ah.Scale != 4 {
		t.Fatalf("expected precision=12 scale=4, got %+v", ah)
	}
	if ah.Default == nil || *ah.Default != "0" {
		t.Fatalf("expected numeric default '0', got %v", ah.Default)
	}

	notNullSpec, _ := fieldByName["NotNullName"].GetResolvedSpec()
	nh := notNullSpec.Structural.StorageHints
	if nh == nil || nh.Required == nil || !*nh.Required {
		t.Fatal("expected required=true from notNull")
	}

	customIdxSpec, _ := fieldByName["CustomIndex"].GetResolvedSpec()
	ch := customIdxSpec.Structural.StorageHints
	if ch == nil || ch.Index == nil || *ch.Index != "idx_custom" || ch.Indexed == nil || !*ch.Indexed {
		t.Fatalf("expected named index hint, got %+v", ch)
	}

	boolIdxSpec, _ := fieldByName["BoolIndex"].GetResolvedSpec()
	bh := boolIdxSpec.Structural.StorageHints
	if bh == nil || bh.Indexed == nil || !*bh.Indexed {
		t.Fatalf("expected boolean index hint, got %+v", bh)
	}

	numDefaultSpec, _ := fieldByName["NumDefault"].GetResolvedSpec()
	ndh := numDefaultSpec.Structural.StorageHints
	if ndh == nil || ndh.Default == nil || *ndh.Default != "42" {
		t.Fatalf("expected numeric default '42', got %v", ndh)
	}

	checkedSpec, _ := fieldByName["CheckedStatus"].GetResolvedSpec()
	if checkedSpec.Structural.CheckConstraint != "status in (draft,done)" {
		t.Fatalf("expected checkConstraint, got %q", checkedSpec.Structural.CheckConstraint)
	}
}

func TestTsParser_ParseModelResolvedSpecCoversCollectBehaviorBindingsBranches(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/behaviors.ts"
	content := `import { Model, Field, Inverse, Compute } from '../../core/service';
import BaseModel from './base';

@Model('BehaviorsModel')
export default class BehaviorsModel extends BaseModel {
  @Field({ type: 'varchar', related: { path: 'PartnerId.Name', store: true } })
  public DisplayName: string

  @Compute<BehaviorsModel>('DisplayName', { deps: ['PartnerId'], store: false })
  computeDn() {
    return this.DisplayName
  }

  @Inverse<BehaviorsModel>('DisplayName')
  inverseDn() {
    this.$inverse.value()
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fieldByName := map[string]*meta.IrField{}
	for _, f := range r.Model.Fields {
		fieldByName[f.Name] = f
	}

	spec, err := fieldByName["DisplayName"].GetResolvedSpec()
	if err != nil || spec == nil {
		t.Fatalf("DisplayName resolved spec: err=%v", err)
	}
	if spec.Behavior.Compute == nil || spec.Behavior.Inverse == nil {
		t.Fatalf("expected both Compute and Inverse, got %+v", spec.Behavior)
	}
}

func TestGetProtoTypeFromTsType_EdgeCases(t *testing.T) {
	tests := map[string]string{
		"":                 "google.protobuf.Value",
		"   ":              "google.protobuf.Value",
		"  string  ":       "string",
		"Promise<void>":    "google.protobuf.Empty",
		"Promise<boolean>": "bool",
		"Promise<string>":  "string",
		"Promise<Custom>":  "google.protobuf.Value",
	}
	for input, want := range tests {
		if got := getProtoTypeFromTsType(input); got != want {
			t.Fatalf("getProtoTypeFromTsType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTsParser_ParseSkipsOnchangeTypesCompatibilityPath(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/core", ApplicationStr: "core"}
	p := NewTsParser(runtimeScope, module)

	path := filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, "core", "service", "runtime", "onchange", "types.ts")
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

func TestTsParser_ParseModelRejectsOrphanBehaviorBinding(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/orphan.ts"
	content := `import { Model, Compute } from '../../core/service';
import BaseModel from './base';

@Model('OrphanModel')
export default class OrphanModel extends BaseModel {
  @Compute<OrphanModel>('NonExistentField', { deps: ['Name'], store: false })
  computeOrphan() {
    return ''
  }
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil || !strings.Contains(err.Error(), "orphan behavior decorator binding for unknown field") {
		t.Fatalf("expected orphan binding error, got %v", err)
	}
}

func TestTsParser_ParseModelRejectsOrphanBehaviorBindingWithPrivateStaticMember(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/private_static_orphan.ts"
	content := `import { Model, Compute } from '../../core/service';
import BaseModel from './base';

@Model('PrivateStaticOrphan')
export default class PrivateStaticOrphan extends BaseModel {
  private static InternalCode: string

  @Compute<PrivateStaticOrphan>('InternalCode', { deps: ['Name'], store: false })
  computeInternal() {
    return ''
  }
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil || !strings.Contains(err.Error(), "orphan behavior decorator binding for unknown field") {
		t.Fatalf("expected orphan binding error for private static member, got %v", err)
	}
}

func TestTsParser_ParseAllowsProtectedStaticMemberToPassFilter(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/protected_static.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('ProtectedStaticModel')
export default class ProtectedStaticModel extends BaseModel {
  @Field({ type: 'varchar' })
  public Name: string

  protected static Secret: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	// The filtered-out protected static member should not appear in fields.
	for _, f := range r.Model.Fields {
		if f.Name == "Secret" {
			t.Fatalf("expected protected static field to be filtered out")
		}
	}
}

func TestTsParser_ParseAllowsPrivateInstanceMember(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/private_instance.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('PrivateInstanceModel')
export default class PrivateInstanceModel extends BaseModel {
  private InternalNote: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	found := false
	for _, f := range r.Model.Fields {
		if f.Name == "InternalNote" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected private instance field to be kept")
	}
}

func TestTsParser_ParseModelRejectsBehaviorDecoratorWithParameters(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/behavior_params.ts"
	content := `import { Model, Field, Compute } from '../../core/service';
import BaseModel from './base';

@Model('BehaviorParamsModel')
export default class BehaviorParamsModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  public Name: string

  @Compute<BehaviorParamsModel>('Name', { deps: ['Name'], store: false })
  computeName(extraParam: string) {
    return this.Name
  }
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil || !strings.Contains(err.Error(), "must be parameterless") {
		t.Fatalf("expected parameterless error, got %v", err)
	}
}

func TestTsParser_ParseModelRejectsBehaviorDecoratorWithEmptyFieldName(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/empty_field_name.ts"
	content := `import { Model, Compute } from '../../core/service';
import BaseModel from './base';

@Model('EmptyFieldNameModel')
export default class EmptyFieldNameModel extends BaseModel {
  @Compute<EmptyFieldNameModel>('', { deps: ['Name'], store: false })
  computeEmpty() {
    return ''
  }
}
`

	_, err := p.Parse(map[string]string{}, path, content)
	if err == nil || !strings.Contains(err.Error(), "requires a field name") {
		t.Fatalf("expected empty field name error, got %v", err)
	}
}

func TestTsParser_ParseModelNoExtendsClass(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/no_extends.ts"
	content := `import { Model, Field } from '../../core/service';

@Model('NoExtendsModel')
export default class NoExtendsModel {
  @Field({ type: 'varchar', size: 64 })
  public Name: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil || r.Model.Name != "NoExtendsModel" {
		t.Fatalf("unexpected model: %+v", r.Model)
	}
	// Without extends, RawExtends should be empty and no extends property synthesized.
	if r.Model.RawExtends != "" {
		t.Fatalf("expected empty RawExtends, got %q", r.Model.RawExtends)
	}
	if r.ModelExtendsProperty != nil {
		t.Fatalf("expected nil ModelExtendsProperty, got %+v", r.ModelExtendsProperty)
	}
}

func TestTsParser_ParseModelServiceWithDecorators(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/decorated_svc.ts"
	content := `import { Model, Field, Api, Guard } from '../../core/service';
import BaseModel from './base';

@Model('DecoratedSvcModel')
export default class DecoratedSvcModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  public Name: string

  @Api
  @Guard
  public static async FindByName(name: string): Promise<void> {}
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.Model.Services) != 1 {
		t.Fatalf("expected one service, got %+v", r.Model.Services)
	}
	svc := r.Model.Services[0]
	if len(svc.Decorators) != 2 {
		t.Fatalf("expected two service decorators, got %d", len(svc.Decorators))
	}
}

func TestTsParser_ParseModelServiceWithTypeParameters(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/generic_svc.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('GenericSvcModel')
export default class GenericSvcModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  public Name: string

  public static async FindOne<T>(id: string): Promise<T> {
    return {} as T
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.Model.Services) != 1 {
		t.Fatalf("expected one service, got %+v", r.Model.Services)
	}
	svc := r.Model.Services[0]
	if len(svc.TypeParameters) != 1 || svc.TypeParameters[0].Name != "T" {
		t.Fatalf("expected one type parameter T, got %+v", svc.TypeParameters)
	}
}

func TestTsParser_ParseModelWithExistingParentPathField(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/existing_parent.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('ExistingParentModel', { parentField: 'ParentId' })
export default class ExistingParentModel extends BaseModel {
  @Field({ type: 'varchar' })
  public Name: string

  @Field({ type: 'ManyToOne' })
  public ParentPath: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	// Only one ParentPath field should exist (the explicit one, not synthesized).
	count := 0
	for _, f := range r.Model.Fields {
		if f.Name == "ParentPath" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one ParentPath field, got %d", count)
	}
}

func TestTsParser_ParseSkipsModelWithPublicStaticField(t *testing.T) {
	runtimeScope := newBackendParserTestScope()
	module := &meta.IrModule{Path: "/virtual/modules/test", ApplicationStr: "test"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/test/service/public_static.ts"
	content := `import { Model, Field } from '../../core/service';
import BaseModel from './base';

@Model('PublicStaticModel')
export default class PublicStaticModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  public Name: string

  public static DefaultName: string
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	found := false
	for _, f := range r.Model.Fields {
		if f.Name == "DefaultName" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected public static field to be kept")
	}
}

func TestAsInt_AllTypes(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
		ok    bool
	}{
		{name: "float64", input: float64(42), want: 42, ok: true},
		{name: "float32", input: float32(3.14), want: 3, ok: true},
		{name: "int", input: int(100), want: 100, ok: true},
		{name: "int32", input: int32(200), want: 200, ok: true},
		{name: "int64", input: int64(300), want: 300, ok: true},
		{name: "uint", input: uint(400), want: 400, ok: true},
		{name: "uint32", input: uint32(500), want: 500, ok: true},
		{name: "uint64", input: uint64(600), want: 600, ok: true},
		{name: "string", input: "not int", want: 0, ok: false},
		{name: "bool", input: true, want: 0, ok: false},
		{name: "nil", input: nil, want: 0, ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := asInt(tc.input)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("asInt(%v) = (%d, %v), want (%d, %v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestParseDecoratorObjectArg_InvalidJSON(t *testing.T) {
	args := []*parser.Argument{
		{Type: "ObjectLiteral", Value: `{invalid}`},
	}
	_, err := parseDecoratorObjectArg(args, 0)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
}

func TestParseDecoratorObjectArg_NilAndNonObject(t *testing.T) {
	// nil when index out of range
	got, err := parseDecoratorObjectArg([]*parser.Argument{}, 0)
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", got, err)
	}

	// nil for non-ObjectLiteral type
	got, err = parseDecoratorObjectArg([]*parser.Argument{{Type: "StringLiteral", Value: `"hello"`}}, 0)
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil) for non-object, got (%v, %v)", got, err)
	}

	// nil for empty string value
	got, err = parseDecoratorObjectArg([]*parser.Argument{{Type: "ObjectLiteral", Value: "   "}}, 0)
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil) for empty value, got (%v, %v)", got, err)
	}
}

func TestParseDecoratorStringArg_EdgeCases(t *testing.T) {
	// empty args
	if got := parseDecoratorStringArg([]*parser.Argument{}, 0); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	// nil element
	if got := parseDecoratorStringArg([]*parser.Argument{nil}, 0); got != "" {
		t.Fatalf("expected empty string for nil, got %q", got)
	}
	// quoted value
	if got := parseDecoratorStringArg([]*parser.Argument{{Value: "  `quoted`  "}}, 0); got != "quoted" {
		t.Fatalf("expected 'quoted', got %q", got)
	}
	// double-quoted value
	if got := parseDecoratorStringArg([]*parser.Argument{{Value: `  "double"  `}}, 0); got != "double" {
		t.Fatalf("expected 'double', got %q", got)
	}
	// single-quoted value
	if got := parseDecoratorStringArg([]*parser.Argument{{Value: `  'single'  `}}, 0); got != "single" {
		t.Fatalf("expected 'single', got %q", got)
	}
}
