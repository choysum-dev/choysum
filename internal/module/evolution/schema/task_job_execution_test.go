// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import "testing"

func TestTaskJobExecutionHelpers(t *testing.T) {
	if name := (taskJobExecution{}).TableName(); name != "task_job_execution" {
		t.Fatalf("unexpected table name: %s", name)
	}

	runtimeScope := newSchemaTestScope(t)
	if err := ensureTaskJobExecutionTable(&schemaTestScope{}); err != nil {
		t.Fatalf("ensureTaskJobExecutionTable(nil session) error = %v", err)
	}
	if err := ensureTaskJobExecutionTable(nil); err != nil {
		t.Fatalf("ensureTaskJobExecutionTable(nil) error = %v", err)
	}
	if err := ensureTaskJobExecutionTable(runtimeScope); err != nil {
		t.Fatalf("ensureTaskJobExecutionTable(env) error = %v", err)
	}
	if !runtimeScope.Session().DB.Migrator().HasTable(&taskJobExecution{}) {
		t.Fatal("expected task_job_execution table to be migrated")
	}

	closedRuntimeScope := newSchemaTestScope(t)
	sqlDB, err := closedRuntimeScope.Session().DB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := ensureTaskJobExecutionTable(closedRuntimeScope); err == nil {
		t.Fatal("expected ensureTaskJobExecutionTable() to fail on closed database")
	}
}
