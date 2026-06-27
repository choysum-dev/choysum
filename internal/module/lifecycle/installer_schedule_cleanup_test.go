// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"
	"time"

	internaltask "github.com/choysum-dev/choysum/internal/task"
)

func TestDisableLegacyModuleIndexDailyScheduleDeletesLegacyEntry(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&internaltask.Schedule{}); err != nil {
		t.Fatalf("auto migrate task schedule: %v", err)
	}

	now := time.Now().UTC()
	legacy := internaltask.Schedule{
		Id:                "sch_legacy",
		Active:            true,
		Name:              "meta.module_index.daily_sync",
		TargetApp:         "meta",
		FullMethod:        "meta.IrModuleIndex/Sync",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "0 0 * * *",
		Timezone:          "UTC",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	other := internaltask.Schedule{
		Id:                "sch_other",
		Active:            true,
		Name:              "document.attachment.gc",
		TargetApp:         "document",
		FullMethod:        "document.AttachmentContent/RunGarbageCollection",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "*/5 * * * *",
		Timezone:          "UTC",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy schedule: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other schedule: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	if err := disableLegacyModuleIndexDailySchedule(runtimeScope); err != nil {
		t.Fatalf("disableLegacyModuleIndexDailySchedule() error = %v", err)
	}

	if err := db.Where("name = ?", "meta.module_index.daily_sync").Take(&internaltask.Schedule{}).Error; err == nil {
		t.Fatal("expected legacy schedule to be deleted")
	}
	if err := db.Where("name = ?", "document.attachment.gc").Take(&internaltask.Schedule{}).Error; err != nil {
		t.Fatalf("expected non-legacy schedule to remain, got %v", err)
	}
}

func TestDisableLegacyModuleIndexDailyScheduleNoTableNoop(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	if err := disableLegacyModuleIndexDailySchedule(runtimeScope); err != nil {
		t.Fatalf("disableLegacyModuleIndexDailySchedule() with missing table error = %v", err)
	}
}
