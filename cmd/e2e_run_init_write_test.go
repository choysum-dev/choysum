// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIRunDoesNotWriteInitArtifacts(t *testing.T) {
	configPath, _, dbPath := writeTempInitializedRunConfigWithDB(t, false)

	output, _ := runCLIUntilLine(t, func(line string) bool {
		return strings.Contains(line, "http server listening")
	}, "run", "--config", configPath)
	if !strings.Contains(output, "http server listening") {
		t.Fatalf("expected run info output, got %s", output)
	}
	if strings.Contains(output, "server starting; NEXT: open") {
		t.Fatalf("did not expect CLI startup hint, got %s", output)
	}

	if sqliteTableExists(t, dbPath, "meta_ir_lock_lease") {
		t.Fatalf("unexpected init lease table created by run")
	}
	if sqliteTableExists(t, dbPath, "meta_ir_model") {
		t.Fatalf("unexpected meta_ir_model table created by run")
	}
	if sqliteTableExists(t, dbPath, "meta_ir_model_data") {
		t.Fatalf("unexpected meta_ir_model_data table created by run")
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var settings []metadata.IrSetting
	if err := db.Find(&settings).Error; err != nil {
		t.Fatalf("query settings: %v", err)
	}
	if len(settings) != 1 {
		t.Fatalf("expected 1 init setting, got %d", len(settings))
	}
	if settings[0].Key != "system.init.done" || settings[0].Value != "true" {
		t.Fatalf("unexpected init setting: %s=%s", settings[0].Key, settings[0].Value)
	}
}
