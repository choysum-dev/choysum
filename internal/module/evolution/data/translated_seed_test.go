// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func TestNormalizeTranslatedSeedValue_ScalarAndMap(t *testing.T) {
	l, db := newTestLoader(t)

	modelID := xid.New().String()
	model := &meta.Model{}
	model.Id.String = modelID
	model.Id.Valid = true
	model.Application = "demo"
	model.Name = "Item"
	model.ModelTable = "demo_item"
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	trueVal := true
	field := &meta.Field{Name: "Name", FieldType: "varchar"}
	field.ModelId.String = modelID
	field.ModelId.Valid = true
	spec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Name:      "Name",
			FieldType: "varchar",
			Translate: &trueVal,
		},
		Migration: meta.FieldMigrationDecision{
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
		t.Fatalf("create field: %v", err)
	}

	rec := record{Module: "demo", ExternalID: "item_1", Model: "demo.Item"}

	got, err := l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", "Hello", nil)
	if err != nil {
		t.Fatalf("scalar normalize: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["en_US"] != "Hello" {
		t.Fatalf("expected en_US scalar wrap, got %#v", got)
	}

	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{
		"en_US": "Hello",
		"zh_CN": "你好",
	}, nil)
	if err == nil {
		t.Fatal("expected unknown zh_CN without Language row to fail")
	}
	var le *LoadError
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedLangUnknown {
		t.Fatalf("expected translated_lang_unknown, got %#v", err)
	}

	langModelID := xid.New().String()
	langModel := &meta.Model{}
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
		"zh_CN": "你好",
	}, nil)
	if err != nil {
		t.Fatalf("map normalize: %v", err)
	}
	m, ok = got.(map[string]any)
	if !ok || m["zh_CN"] != "你好" || m["en_US"] != "Hello" {
		t.Fatalf("expected bilingual map, got %#v", got)
	}

	_, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, rec, model, "Name", map[string]any{
		"_t": "Hello",
	}, nil)
	if !errors.As(err, &le) || le.Code != LoadErrorCodeTranslatedSeedInvalid {
		t.Fatalf("expected reject _t seed, got %#v", err)
	}

	langNameField := &meta.Field{Name: "Name", FieldType: "varchar"}
	langNameField.ModelId.String = langModelID
	langNameField.ModelId.Valid = true
	langSpec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Name:      "Name",
			FieldType: "varchar",
			Translate: &trueVal,
		},
		Migration: meta.FieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "jsonobject",
			ReasonCode:         "TRANSLATE_LANG_MAP",
		},
	}
	if err := langNameField.SetResolvedSpec(langSpec); err != nil {
		t.Fatalf("SetResolvedSpec Language.Name: %v", err)
	}
	if err := db.Create(langNameField).Error; err != nil {
		t.Fatalf("create Language.Name field: %v", err)
	}

	langRec := record{Module: "base", ExternalID: "language_fr", Model: "base.Language"}
	got, err = l.normalizeTranslatedSeedValue(db, "/tmp/data.json", 0, langRec, langModel, "Name", map[string]any{
		"fr_FR": "Français",
	}, map[string]any{"Code": "fr_FR", "Name": map[string]any{"fr_FR": "Français"}})
	if err != nil {
		t.Fatalf("self Code should allow fr_FR on Language seed: %v", err)
	}
	m, ok = got.(map[string]any)
	if !ok || m["fr_FR"] != "Français" {
		t.Fatalf("unexpected self-code map: %#v", got)
	}
}
