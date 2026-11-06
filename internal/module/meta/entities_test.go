// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"reflect"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestDualStoreRawAndCatalogEntities(t *testing.T) {
	rawExpected := []reflect.Type{
		reflect.TypeOf(&rawModel{}),
		reflect.TypeOf(&rawField{}),
		reflect.TypeOf(&rawService{}),
		reflect.TypeOf(&rawTypeParameter{}),
		reflect.TypeOf(&rawParameter{}),
		reflect.TypeOf(&rawDecorator{}),
		reflect.TypeOf(&rawArgument{}),
	}
	rawEntities := DualStoreRawEntities()
	if len(rawEntities) != len(rawExpected) {
		t.Fatalf("DualStoreRawEntities() len = %d, want %d", len(rawEntities), len(rawExpected))
	}
	for index, entity := range rawEntities {
		if reflect.TypeOf(entity) != rawExpected[index] {
			t.Fatalf("DualStoreRawEntities()[%d] type = %v, want %v", index, reflect.TypeOf(entity), rawExpected[index])
		}
	}

	entities := pkgmeta.Entities()
	opsEntities := OpsEntities()
	catalog := CatalogEntities()
	wantLen := len(entities) + len(rawEntities) + len(opsEntities)
	if len(catalog) != wantLen {
		t.Fatalf("CatalogEntities() len = %d, want %d", len(catalog), wantLen)
	}

	tableNames := []struct {
		name string
		got  string
		want string
	}{
		{name: "rawModel", got: (&rawModel{}).TableName(), want: "meta_raw_model"},
		{name: "rawField", got: (&rawField{}).TableName(), want: "meta_raw_field"},
		{name: "rawService", got: (&rawService{}).TableName(), want: "meta_raw_service"},
		{name: "rawTypeParameter", got: (&rawTypeParameter{}).TableName(), want: "meta_raw_type_parameter"},
		{name: "rawParameter", got: (&rawParameter{}).TableName(), want: "meta_raw_parameter"},
		{name: "rawDecorator", got: (&rawDecorator{}).TableName(), want: "meta_raw_decorator"},
		{name: "rawArgument", got: (&rawArgument{}).TableName(), want: "meta_raw_argument"},
	}
	for _, check := range tableNames {
		if check.got != check.want {
			t.Fatalf("%s.TableName() = %q, want %q", check.name, check.got, check.want)
		}
	}
}

func TestOpsEntitiesAndTableNames(t *testing.T) {
	entities := OpsEntities()
	if len(entities) != 7 {
		t.Fatalf("OpsEntities() len = %d, want 7", len(entities))
	}

	wantTables := map[string]string{
		"ModuleOrigin":    "meta_module_origin",
		"ModuleCatalog":   "meta_module_catalog",
		"ModuleOperation": "meta_module_operation",
		"ModelData":       "meta_model_data",
		"ModuleMigration": "meta_module_migration",
		"LockLease":       "meta_lock_lease",
		"SchemaSnapshot":  "meta_schema_snapshot",
	}
	gotTables := map[string]string{
		"ModuleOrigin":    (ModuleOrigin{}).TableName(),
		"ModuleCatalog":   (ModuleCatalog{}).TableName(),
		"ModuleOperation": (ModuleOperation{}).TableName(),
		"ModelData":       (&ModelData{}).TableName(),
		"ModuleMigration": (ModuleMigration{}).TableName(),
		"LockLease":       (&LockLease{}).TableName(),
		"SchemaSnapshot":  (SchemaSnapshot{}).TableName(),
	}
	for name, want := range wantTables {
		got := gotTables[name]
		if got != want {
			t.Fatalf("%s.TableName() = %q, want %q", name, got, want)
		}
	}
}
