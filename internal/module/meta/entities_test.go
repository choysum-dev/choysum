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
	if len(entities) != 6 {
		t.Fatalf("OpsEntities() len = %d, want 6", len(entities))
	}

	wantTables := map[string]string{
		"ModuleIndex":            (&ModuleIndex{}).TableName(),
		"ModelData":              (&ModelData{}).TableName(),
		"Setting":                (&Setting{}).TableName(),
		"ModuleMigrationHistory": (ModuleMigrationHistory{}).TableName(),
		"LockLease":              (&LockLease{}).TableName(),
	}
	for name, table := range wantTables {
		if table == "" || table == "meta_ir_"+name {
			t.Fatalf("%s.TableName() = %q, unexpected ir prefix or empty", name, table)
		}
	}
	if got := (&ModuleIndex{}).TableName(); got != "meta_module_index" {
		t.Fatalf("ModuleIndex.TableName() = %q", got)
	}
	if got := (&ModelData{}).TableName(); got != "meta_model_data" {
		t.Fatalf("ModelData.TableName() = %q", got)
	}
	if got := (&Setting{}).TableName(); got != "meta_setting" {
		t.Fatalf("Setting.TableName() = %q", got)
	}
	if got := (ModuleMigrationHistory{}).TableName(); got != "meta_module_migration_history" {
		t.Fatalf("ModuleMigrationHistory.TableName() = %q", got)
	}
	if got := (ModuleManagementLog{}).TableName(); got != "meta_module_management_log" {
		t.Fatalf("ModuleManagementLog.TableName() = %q", got)
	}
	if got := (&LockLease{}).TableName(); got != "meta_lock_lease" {
		t.Fatalf("LockLease.TableName() = %q", got)
	}
}
