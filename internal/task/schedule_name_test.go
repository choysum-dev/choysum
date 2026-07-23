// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEncodeDecodeTranslatedScheduleName(t *testing.T) {
	encoded := EncodeTranslatedScheduleName("document.attachment.gc")
	if encoded != `{"en_US":"document.attachment.gc"}` {
		t.Fatalf("unexpected encode: %s", encoded)
	}
	if got := DecodeTranslatedScheduleName(encoded); got != "document.attachment.gc" {
		t.Fatalf("decode lang map: got %q", got)
	}
	if got := DecodeTranslatedScheduleName("plain"); got != "plain" {
		t.Fatalf("decode legacy: got %q", got)
	}
}

func TestWhereScheduleNameEqMatchesLegacyAndTranslated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:schedule_name?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE task_schedule (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`INSERT INTO task_schedule (id, name) VALUES (?, ?)`, "1", "document.attachment.gc").Error; err != nil {
		t.Fatalf("insert legacy: %v", err)
	}
	if err := db.Exec(`INSERT INTO task_schedule (id, name) VALUES (?, ?)`, "2", EncodeTranslatedScheduleName("meta.module_index.daily_sync")).Error; err != nil {
		t.Fatalf("insert translated: %v", err)
	}

	var legacy Schedule
	if err := WhereScheduleNameEq(db, "document.attachment.gc").Take(&legacy).Error; err != nil {
		t.Fatalf("legacy match: %v", err)
	}
	if legacy.Id != "1" {
		t.Fatalf("expected id 1, got %s", legacy.Id)
	}

	var translated Schedule
	if err := WhereScheduleNameEq(db, "meta.module_index.daily_sync").Take(&translated).Error; err != nil {
		t.Fatalf("translated match: %v", err)
	}
	if translated.Id != "2" {
		t.Fatalf("expected id 2, got %s", translated.Id)
	}
}
