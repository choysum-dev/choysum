// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestMergeSameNameModelsByExtensionChain_PartnerStyleUnion(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	bankPath := "@/partner_bank/service/models/partner.ts"
	commercialPath := "@/partner_commercial/service/models/partner.ts"

	base := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields:    []*pkgmeta.Field{{Name: "Name"}, {Name: "Contacts"}},
	}
	bank := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
		Name:      "Partner",
		Path:      bankPath,
		Extends:   basePath,
		Fields:    []*pkgmeta.Field{{Name: "BankAccounts"}},
	}
	commercial := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "commercial", Valid: true}},
		Name:      "Partner",
		Path:      commercialPath,
		Extends:   basePath,
		Fields:    []*pkgmeta.Field{{Name: "PartnerIdentifiers"}},
	}

	merged, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{base, bank, commercial})
	if err != nil {
		t.Fatalf("MergeSameNameModelsByExtensionChain: %v", err)
	}
	if merged == nil {
		t.Fatal("expected merged model")
	}
	fieldNames := map[string]bool{}
	for _, f := range merged.Fields {
		if f != nil && f.Name != "" {
			fieldNames[f.Name] = true
		}
	}
	for _, expected := range []string{"Name", "Contacts", "BankAccounts", "PartnerIdentifiers"} {
		if !fieldNames[expected] {
			t.Fatalf("expected field %q in %#v", expected, fieldNames)
		}
	}
}

func TestMergeEffectiveModel_FromRaw(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	bankPath := "@/partner_bank/service/models/partner.ts"
	raws := []*rawModel{
		{
			BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
			Name:      "Partner",
			Path:      basePath,
			Fields:    []*rawField{{Name: "Name"}},
		},
		{
			BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
			Name:      "Partner",
			Path:      bankPath,
			Extends:   basePath,
			Fields:    []*rawField{{Name: "BankAccounts"}},
		},
	}
	merged, err := mergeEffectiveModel(raws)
	if err != nil {
		t.Fatalf("mergeEffectiveModel: %v", err)
	}
	names := map[string]bool{}
	for _, f := range merged.Fields {
		if f != nil {
			names[f.Name] = true
		}
	}
	if !names["Name"] || !names["BankAccounts"] {
		t.Fatalf("unexpected fields %#v", names)
	}
}

func TestMergeEffectiveModel_OmitsThisParameter(t *testing.T) {
	raws := []*rawModel{{
		BaseModel:   pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:        "Partner",
		Application: "partner",
		Path:        "@/partner/service/models/partner.ts",
		Services: []*rawService{{
			Name: "Create",
			Parameters: []*rawParameter{
				{Name: "this"},
				{Name: "vals"},
			},
		}},
	}}
	merged, err := mergeEffectiveModel(raws)
	if err != nil {
		t.Fatalf("mergeEffectiveModel: %v", err)
	}
	if merged == nil || len(merged.Services) != 1 {
		t.Fatalf("expected one service, got %#v", merged)
	}
	params := merged.Services[0].Parameters
	if len(params) != 1 || params[0] == nil || params[0].Name != "vals" {
		t.Fatalf("expected only vals parameter (this omitted), got %#v", params)
	}
}

func TestMergeSameNameModelsByExtensionChain_CycleError(t *testing.T) {
	aPath := "@/a/partner.ts"
	bPath := "@/b/partner.ts"
	a := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a", Valid: true}},
		Name:      "Partner",
		Path:      aPath,
		Extends:   bPath,
	}
	b := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b", Valid: true}},
		Name:      "Partner",
		Path:      bPath,
		Extends:   aPath,
	}
	_, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{a, b})
	if err == nil || !strings.Contains(err.Error(), "extends cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SelectionAddWithoutBaseRejected(t *testing.T) {
	extField := &pkgmeta.Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	}); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	ext := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "Partner",
		Path:      "@/partner_vip/service/models/partner.ts",
		Fields:    []*pkgmeta.Field{extField},
	}
	base := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      "@/partner/service/models/partner.ts",
		Fields:    []*pkgmeta.Field{{Name: "Name"}},
	}
	ext.Extends = base.Path
	_, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{base, ext})
	if err == nil || !strings.Contains(err.Error(), "selectionAdd requires an inherited static selection") {
		t.Fatalf("expected selectionAdd-without-base rejection, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SoloSelectionAddRejected(t *testing.T) {
	extField := &pkgmeta.Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	}); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	solo := &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "solo", Valid: true}},
		Name:      "Partner",
		Path:      "@/partner/service/models/partner.ts",
		Fields:    []*pkgmeta.Field{extField},
	}
	_, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{solo})
	if err == nil || !strings.Contains(err.Error(), "selectionAdd requires an inherited static selection") {
		t.Fatalf("expected solo selectionAdd rejection, got %v", err)
	}
}

func TestRawFieldAndFieldExportedColumnsMatch(t *testing.T) {
	skip := map[string]bool{
		"ModelId": true, "Model": true, "Decorators": true,
	}
	fieldType := reflect.TypeOf(pkgmeta.Field{})
	rawType := reflect.TypeOf(rawField{})
	fieldCols := map[string]reflect.Type{}
	for i := 0; i < fieldType.NumField(); i++ {
		f := fieldType.Field(i)
		if !f.IsExported() || skip[f.Name] {
			continue
		}
		fieldCols[f.Name] = f.Type
	}
	rawCols := map[string]reflect.Type{}
	for i := 0; i < rawType.NumField(); i++ {
		f := rawType.Field(i)
		if !f.IsExported() || skip[f.Name] {
			continue
		}
		rawCols[f.Name] = f.Type
	}
	if len(fieldCols) != len(rawCols) {
		t.Fatalf("pkgmeta.Field cols=%d rawField cols=%d", len(fieldCols), len(rawCols))
	}
	for name, typ := range fieldCols {
		got, ok := rawCols[name]
		if !ok {
			t.Fatalf("rawField missing column %s", name)
		}
		if got != typ {
			t.Fatalf("rawField.%s type %v, pkgmeta.Field.%s type %v", name, got, name, typ)
		}
	}
}

func TestMergeSameNameModelsByExtensionChain_SameDepthTieBreak(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	branchAPath := "@/partner_bank/service/models/partner.ts"
	branchBPath := "@/partner_commercial/service/models/partner.ts"

	base := &pkgmeta.Model{BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}}, Name: "Partner", Path: basePath, Fields: []*pkgmeta.Field{{Name: "Name", TsTypeAnnotation: "base-name"}}}
	branchA := &pkgmeta.Model{BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-older", Valid: true}}, Name: "Partner", Path: branchAPath, Extends: basePath, Fields: []*pkgmeta.Field{{Name: "Name", TsTypeAnnotation: "branch-a"}}}
	branchBNewer := &pkgmeta.Model{BaseModel: pkgmeta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-newer", Valid: true}}, Name: "Partner", Path: branchBPath, Extends: basePath, Fields: []*pkgmeta.Field{{Name: "Name", TsTypeAnnotation: "branch-b-newer"}}}

	merged, err := MergeSameNameModelsByExtensionChain([]*pkgmeta.Model{base, branchA, branchBNewer})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	var name *pkgmeta.Field
	for _, f := range merged.Fields {
		if f != nil && f.Name == "Name" {
			name = f
			break
		}
	}
	if name == nil || name.TsTypeAnnotation != "branch-b-newer" {
		t.Fatalf("unexpected UpdatedAt tie-break: %#v", name)
	}
}
