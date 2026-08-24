// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	recordwriter "github.com/choysum-dev/choysum/internal/import/writer/record"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestAssertExternalIDProtected_InitdataNamespace(t *testing.T) {
	runtimeScope := newExternalIDTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Create(&meta.Module{Name: "base", ApplicationStr: "base", Path: "/tmp"}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module:      "base",
		Name:        "company_main",
		Application: "base",
		ModelName:   "Company",
		ModelId:     "model-1",
		ResID:       "res-1",
		NoUpdate:    true,
	}).Error; err != nil {
		t.Fatalf("seed model data: %v", err)
	}

	err := recordwriter.AssertExternalIDWritable(db, recordwriter.MetaModelDataKey{
		Module: "base",
		Name:   "company_main",
	}, 3)
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeExternalIDProtected {
		t.Fatalf("error = %v, want external_id_protected", err)
	}
}

func TestAssertExternalIDProtected_InstalledNamespaceWithoutMapping(t *testing.T) {
	runtimeScope := newExternalIDTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Create(&meta.Module{Name: "base", ApplicationStr: "base", Path: "/tmp"}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}

	err := recordwriter.AssertExternalIDWritable(db, recordwriter.MetaModelDataKey{
		Module: "base",
		Name:   "new_id",
	}, 1)
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeExternalIDProtected {
		t.Fatalf("error = %v, want external_id_protected for first import under installed module", err)
	}
}

func TestParseMetaModelDataKey_DefaultImportNamespace(t *testing.T) {
	key, err := recordwriter.ParseMetaModelDataKey("country_demo")
	if err != nil {
		t.Fatalf("ParseMetaModelDataKey: %v", err)
	}
	if key.Module != "import" || key.Name != "country_demo" {
		t.Fatalf("key = %#v", key)
	}
}

func TestParseMetaModelDataKey_Errors(t *testing.T) {
	cases := []string{"", "  ", ".", "mod.", ".name", " . "}
	for _, raw := range cases {
		if _, err := recordwriter.ParseMetaModelDataKey(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
	key, err := recordwriter.ParseMetaModelDataKey("custom.item")
	if err != nil || key.Module != "custom" || key.Name != "item" {
		t.Fatalf("key=%#v err=%v", key, err)
	}
}

func TestAssertExternalIDWritable_WritableMappingAndNilDB(t *testing.T) {
	if err := recordwriter.AssertExternalIDWritable(nil, recordwriter.MetaModelDataKey{Module: "import", Name: "x"}, 1); err == nil {
		t.Fatal("expected nil db error")
	}
	runtimeScope := newExternalIDTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.Create(&modmeta.ModelData{
		Module:      "import",
		Name:        "writable",
		Application: "base",
		ModelName:   "Country",
		ModelId:     "m1",
		ResID:       "r1",
		NoUpdate:    false,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := recordwriter.AssertExternalIDWritable(db, recordwriter.MetaModelDataKey{Module: "import", Name: "writable"}, 2); err != nil {
		t.Fatalf("writable mapping: %v", err)
	}
	if err := recordwriter.AssertExternalIDWritable(db, recordwriter.MetaModelDataKey{Module: "import", Name: "missing"}, 2); err != nil {
		t.Fatalf("missing mapping under import: %v", err)
	}
}

func newExternalIDTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{Db: &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "ext.db")}}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)
	if err := runtimeScope.Session().AutoMigrate(&meta.Module{}, &modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}
