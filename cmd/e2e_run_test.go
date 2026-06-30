package

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later
cmd

import (
	"fmt"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIRunRejectsLegacyBootstrapFlags(t *testing.T) {
	output, code := runCLI(t, "run", "--non-interactive")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --non-interactive") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}
func TestCLIRunInfoShowsActualAddress(t *testing.T) {
	configPath, addr, _ := writeTempInitializedRunConfig(t, false)
	expected := fmt.Sprintf("http://%s", addr)

	output, _ := runCLIUntilLine(t, func(line string) bool {
		return strings.Contains(line, "http server listening")
	}, "run", "--config", configPath)
	if strings.Contains(output, "server starting; NEXT: open") {
		t.Fatalf("did not expect CLI startup hint, got %s", output)
	}
	if !strings.Contains(output, expected) {
		t.Fatalf("expected output to contain %q, got %s", expected, output)
	}
}
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
func TestCLIRunRejectsLegacyAdminUsernameFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-username", "admin")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-username") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}

func TestCLIRunRejectsLegacyAdminPasswordFileFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-password-file", "pw.txt")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-password-file") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}

func TestCLIRunRejectsLegacyAdminPasswordStdinFlag(t *testing.T) {
	output, code := runCLI(t, "run", "--admin-password-stdin")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown flag: --admin-password-stdin") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}
