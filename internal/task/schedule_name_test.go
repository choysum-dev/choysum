// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dialectorWithName struct {
	gorm.Dialector
	name string
}

func (d dialectorWithName) Name() string { return d.name }

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
	if got := EncodeTranslatedScheduleName("  spaced  "); got != `{"en_US":"spaced"}` {
		t.Fatalf("encode trim: %s", got)
	}
	if got := DecodeTranslatedScheduleName(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := DecodeTranslatedScheduleName(`{"zh_CN":"中文"}`); got != "中文" {
		t.Fatalf("zh fallback: %q", got)
	}
	if got := DecodeTranslatedScheduleName(`{"en_US":"","fr_FR":"Fr"}`); got != "Fr" {
		t.Fatalf("first non-empty: %q", got)
	}
	if got := DecodeTranslatedScheduleName(`{"en_US":"","zh_CN":""}`); got != `{"en_US":"","zh_CN":""}` {
		t.Fatalf("all empty returns raw: %q", got)
	}
	if got := DecodeTranslatedScheduleName("{not-json"); got != "{not-json" {
		t.Fatalf("invalid json: %q", got)
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

func TestWhereScheduleNameEqNilAndPostgresSQL(t *testing.T) {
	if got := WhereScheduleNameEq(nil, "x"); got != nil {
		t.Fatal("nil db must return nil")
	}

	db, err := gorm.Open(sqlite.Open("file:schedule_name_pg?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Dialector = dialectorWithName{Dialector: db.Dialector, name: "postgres"}
	if got := db.Dialector.Name(); got != "postgres" {
		t.Fatalf("expected postgres dialector name, got %s", got)
	}
	tx := WhereScheduleNameEq(db.Session(&gorm.Session{DryRun: true}).Table("task_schedule"), "document.attachment.gc")
	stmt := tx.Find(&struct{}{}).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "name->>'en_US'") {
		t.Fatalf("expected postgres unwrap clause, got %s", sql)
	}
	if !strings.Contains(sql, "to_jsonb") {
		t.Fatalf("expected to_jsonb legacy clause, got %s", sql)
	}
}
