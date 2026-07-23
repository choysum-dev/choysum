// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func TestSeedSelfLanguageCodeAndStringify(t *testing.T) {
	if got := seedSelfLanguageCode(nil); got != "" {
		t.Fatalf("nil values: %q", got)
	}
	if got := seedSelfLanguageCode(map[string]any{}); got != "" {
		t.Fatalf("missing Code: %q", got)
	}
	if got := seedSelfLanguageCode(map[string]any{"Code": " fr_FR "}); got != "fr_FR" {
		t.Fatalf("string Code: %q", got)
	}
	if got := seedSelfLanguageCode(map[string]any{"Code": 42}); got != "42" {
		t.Fatalf("numeric Code: %q", got)
	}
	if got := stringifySeedScalar("keep"); got != "keep" {
		t.Fatalf("string scalar: %q", got)
	}
	if got := stringifySeedScalar(true); got != "true" {
		t.Fatalf("bool scalar: %q", got)
	}
	if got := stringifySeedScalar(3.5); got != "3.5" {
		t.Fatalf("float scalar: %q", got)
	}
}

func TestIsTranslateFieldAndLookupIrField(t *testing.T) {
	if isTranslateField(nil) {
		t.Fatal("nil field")
	}
	l, db := newTestLoader(t)
	field, err := l.lookupIrField(db, nil, "Name")
	if err != nil || field != nil {
		t.Fatalf("nil model: %#v %v", field, err)
	}
	model := &meta.IrModel{}
	field, err = l.lookupIrField(db, model, "Name")
	if err != nil || field != nil {
		t.Fatalf("empty model id: %#v %v", field, err)
	}
	model.Id.String = xid.New().String()
	model.Id.Valid = true
	field, err = l.lookupIrField(db, model, "  ")
	if err != nil || field != nil {
		t.Fatalf("empty field name: %#v %v", field, err)
	}
	field, err = l.lookupIrField(db, model, "Missing")
	if err != nil || field != nil {
		t.Fatalf("missing field: %#v %v", field, err)
	}
}

func TestNormalizeTranslatedSeedValue_AdditionalBranches(t *testing.T) {
	l, db := newTestLoader(t)

	modelID := xid.New().String()
	model := &meta.IrModel{}
	model.Id.String = modelID
	model.Id.Valid = true
	model.Application = "demo"
	model.Name = "Item"
	model.ModelTable = "demo_item"
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	falseVal := false
	nonTranslate := &meta.IrField{Name: "Code", FieldType: "varchar"}
	nonTranslate.ModelId.String = modelID
	nonTranslate.ModelId.Valid = true
	nonSpec := &meta.IrFieldResolvedSpec{
		FieldName: "Code",
		Structural: meta.IrFieldStructuralSpec{
			Name:      "Code",
			FieldType: "varchar",
			Translate: &falseVal,
		},
	}
	if err := nonTranslate.SetResolvedSpec(nonSpec); err != nil {
		t.Fatalf("SetResolvedSpec non-translate: %v", err)
	}
	if err := db.Create(nonTranslate).Error; err != nil {
		t.Fatalf("create Code field: %v", err)
	}

	trueVal := true
	field := &meta.IrField{Name: "Name", FieldType: "varchar"}
	field.ModelId.String = modelID
	field.ModelId.Valid = true
	spec := &meta.IrFieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.IrFieldStructuralSpec{
			Name:      "Name",
			FieldType: "varchar",
			Translate: &trueVal,
		},
		Migration: meta.IrFieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "jsonobject",
			ReasonCode:         "TRANSLATE_LANG_MAP",
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	if err := db.Create(field).Error; err != nil {
		t.Fatalf("create Name field: %v", err)
	}

	rec := record{Module: "demo", ExternalID: "item_1", Model: "demo.Item"}

	got, err := l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Code", "KEEP", nil)
	if err != nil || got != "KEEP" {
		t.Fatalf("non-translate passthrough: %#v %v", got, err)
	}

	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", nil, nil)
	if err != nil || got != nil {
		t.Fatalf("nil value: %#v %v", got, err)
	}

	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", 42, nil)
	if err != nil {
		t.Fatalf("numeric scalar: %v", err)
	}
	if m, ok := got.(map[string]any); !ok || m["en_US"] != "42" {
		t.Fatalf("numeric scalar wrap: %#v", got)
	}

	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", true, nil)
	if err != nil {
		t.Fatalf("bool scalar: %v", err)
	}
	if m, ok := got.(map[string]any); !ok || m["en_US"] != "true" {
		t.Fatalf("bool scalar wrap: %#v", got)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", []any{"x"}, nil)
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("invalid scalar type: %#v", err)
	}

	refValue := map[string]any{"ref": "base.company_main"}
	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", refValue, nil)
	if err != nil {
		t.Fatalf("ref passthrough: %v", err)
	}
	if got == nil {
		t.Fatal("expected ref map passthrough")
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("empty map: %#v", err)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{"": "x"}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("empty lang key: %#v", err)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{"zh-CN": "x"}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("hyphen locale key: %#v", err)
	}

	langModelID := xid.New().String()
	langModel := &meta.IrModel{}
	langModel.Id.String = langModelID
	langModel.Id.Valid = true
	langModel.Application = "base"
	langModel.Name = "Language"
	langModel.ModelTable = "base_language"
	if err := db.Create(langModel).Error; err != nil {
		t.Fatalf("create Language model: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, code text)`).Error; err != nil {
		t.Fatalf("create base_language: %v", err)
	}
	if err := db.Exec(`INSERT INTO base_language (id, code) VALUES ('1', 'zh_CN')`).Error; err != nil {
		t.Fatalf("insert language: %v", err)
	}

	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{
		"en_US": "Hello",
		"zh_CN": 123,
	}, nil)
	if err != nil {
		t.Fatalf("numeric map value: %v", err)
	}
	if m, ok := got.(map[string]any); !ok || m["zh_CN"] != "123" {
		t.Fatalf("coerced map value: %#v", got)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{
		"en_US": "Hello",
		"zh_CN": nil,
	}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("nil map value: %#v", err)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{
		"en_US": "Hello",
		"zh_CN": map[string]any{"nested": true},
	}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("object map value: %#v", err)
	}

	ok, err := l.languageCodeExists(db, "")
	if err != nil || ok {
		t.Fatalf("empty language code: %v %v", ok, err)
	}
	ok, err = l.languageCodeExists(db, "zh_CN")
	if err != nil || !ok {
		t.Fatalf("known language code: %v %v", ok, err)
	}

	emptyTableModel := &meta.IrModel{Name: "Language", Application: "base", Path: "/tmp", ModelTable: ""}
	if err := db.Where("application = ? AND name = ?", "base", "Language").Delete(&meta.IrModel{}).Error; err != nil {
		t.Fatalf("delete language model: %v", err)
	}
	if err := db.Create(emptyTableModel).Error; err != nil {
		t.Fatalf("create empty ModelTable language: %v", err)
	}
	ok, err = l.languageCodeExists(db, "zh_CN")
	if err != nil || ok {
		t.Fatalf("empty ModelTable must skip lookup: %v %v", ok, err)
	}
}
