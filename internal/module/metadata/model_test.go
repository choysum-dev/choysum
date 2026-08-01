// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package metadata

import "testing"

func TestEntitiesAndTableNames(t *testing.T) {
	entities := Entities()
	if len(entities) != 5 {
		t.Fatalf("Entities() len = %d, want 5", len(entities))
	}

	wantTables := map[string]string{
		"ModuleIndex":            (&ModuleIndex{}).TableName(),
		"ModelData":              (&ModelData{}).TableName(),
		"Setting":                (&Setting{}).TableName(),
		"ModuleMigrationHistory": (ModuleMigrationHistory{}).TableName(),
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
}
