// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func taskJobExecutionColumns() []ColumnSpec {
	size64 := 64
	size32 := 32
	size128 := 128
	size20 := 20
	size255 := 255
	return []ColumnSpec{
		{Name: "job_id", FieldName: "JobId", PhysicalType: "varchar", Size: &size64, NotNull: true, UniqueIndex: true},
		{Name: "status", FieldName: "Status", PhysicalType: "varchar", Size: &size32, Indexed: true},
		{Name: "lease_owner", FieldName: "LeaseOwner", PhysicalType: "varchar", Size: &size128, Indexed: true},
		{Name: "lease_until", FieldName: "LeaseUntil", PhysicalType: "datetime", Indexed: true},
		{Name: "attempt", FieldName: "Attempt", PhysicalType: "int"},
		{Name: "scheduler_user_id", FieldName: "SchedulerUserId", PhysicalType: "varchar", Size: &size20, Indexed: true},
		{Name: "triggered_by_user_id", FieldName: "TriggeredByUserId", PhysicalType: "varchar", Size: &size20, Indexed: true},
		{Name: "full_method", FieldName: "FullMethod", PhysicalType: "varchar", Size: &size255, Indexed: true},
		{Name: "payload_json", FieldName: "PayloadJson", PhysicalType: "jsonobject"},
		{Name: "result_json", FieldName: "ResultJson", PhysicalType: "jsonobject"},
		{Name: "result_hash", FieldName: "ResultHash", PhysicalType: "varchar", Size: &size128},
		{Name: "result_truncated", FieldName: "ResultTruncated", PhysicalType: "boolean"},
		{Name: "error_json", FieldName: "ErrorJson", PhysicalType: "jsonobject"},
		{Name: "error_hash", FieldName: "ErrorHash", PhysicalType: "varchar", Size: &size128},
		{Name: "error_truncated", FieldName: "ErrorTruncated", PhysicalType: "boolean"},
		{Name: "cancelled_at", FieldName: "CancelledAt", PhysicalType: "datetime", Indexed: true},
		{Name: "started_at", FieldName: "StartedAt", PhysicalType: "datetime", Indexed: true},
		{Name: "finished_at", FieldName: "FinishedAt", PhysicalType: "datetime", Indexed: true},
		{Name: "created_at", FieldName: "CreatedAt", PhysicalType: "datetime", Indexed: true},
		{Name: "updated_at", FieldName: "UpdatedAt", PhysicalType: "datetime", Indexed: true},
	}
}

var (
	structForCreateTableFn    = structForCreateTable
	structForAddColumnFn      = structForAddColumn
	ensureTaskJobIndexesFn    = ensureIndexesForColumn
	taskJobExecutionColumnsFn = taskJobExecutionColumns
)

func schemaDialectName(dialector gorm.Dialector) string {
	if dialector == nil {
		return "sqlite"
	}
	switch strings.ToLower(dialector.Name()) {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlserver":
		return "sqlserver"
	default:
		return "sqlite"
	}
}

func ensureTaskJobExecutionTable(runtimeScope scope.Scope) error {
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	db := runtimeScope.Session()
	dialect := schemaDialectName(db.Dialector)
	const table = "task_job_execution"
	cols := taskJobExecutionColumnsFn()
	if !db.Migrator().HasTable(table) {
		inst, err := structForCreateTableFn(table, cols, dialect)
		if err != nil {
			return fmt.Errorf("build task_job_execution create struct: %w", err)
		}
		if err := db.Table(table).Migrator().CreateTable(inst); err != nil {
			return fmt.Errorf("create task_job_execution: %w", err)
		}
		for _, col := range cols {
			if !col.Indexed && !col.Unique && !col.UniqueIndex && len(col.UniqueIndexNames) == 0 {
				continue
			}
			if err := ensureTaskJobIndexesFn(db.DB, table, col, dialect); err != nil {
				return fmt.Errorf("ensure task_job_execution index %s: %w", col.Name, err)
			}
		}
		return nil
	}
	for _, col := range cols {
		if db.Migrator().HasColumn(table, col.Name) {
			continue
		}
		inst, err := structForAddColumnFn(table, col, dialect)
		if err != nil {
			return fmt.Errorf("build task_job_execution add column %s: %w", col.Name, err)
		}
		fieldName := exportIdent(col.FieldName)
		if err := db.Table(table).Migrator().AddColumn(inst, fieldName); err != nil {
			return fmt.Errorf("add task_job_execution column %s: %w", col.Name, err)
		}
		if !col.Indexed && !col.Unique && !col.UniqueIndex && len(col.UniqueIndexNames) == 0 {
			continue
		}
		if err := ensureTaskJobIndexesFn(db.DB, table, col, dialect); err != nil {
			return fmt.Errorf("ensure task_job_execution index %s: %w", col.Name, err)
		}
	}
	return nil
}
