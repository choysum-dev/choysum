// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIErrorBlockLastOutput_InitInteractive(t *testing.T) {
	output, code := runCLI(t, "init")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(output, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", output)
	}
}

func TestCLIErrorBlockLastOutput_RunUninitialized(t *testing.T) {
	t.Skip("run no longer blocks on initialization state")
}

func TestCLIErrorBlockLastOutput_RunConfigMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	output, code := runCLI(t, "run", "--config", missing)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunInvalidSqlitePath(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", "relative.db", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
	reason := extractReason(output)
	assertReasonInSet(t, reason, []string{"path is not absolute"})
}

func TestCLIErrorBlockLastOutput_RunAddonsPathMissing(t *testing.T) {
	t.Skip("run no longer performs interactive bootstrap when addons_path is omitted")
}

func TestCLIErrorBlockLastOutput_RunAddonsPathUnreadable(t *testing.T) {
	addonsDir := filepath.Join(t.TempDir(), "addons")
	if err := os.MkdirAll(addonsDir, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	if err := os.Chmod(addonsDir, 0o000); err != nil {
		t.Skipf("chmod addons dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(addonsDir, 0o755)
	})
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), addonsDir)
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunAddonsPathSymlink(t *testing.T) {
	addonsDir := filepath.Join(t.TempDir(), "addons")
	if err := os.MkdirAll(addonsDir, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	linkPath := filepath.Join(t.TempDir(), "addons-link")
	if err := os.Symlink(addonsDir, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), linkPath)
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunAddonsPathWhitespace(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	if err := os.WriteFile(configPath, []byte("addons_path: \"   \"\n"+readConfigDbBlock(t, "sqlite", writeTempSqliteDB(t))), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunAddonsPathControlChar(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	if err := os.WriteFile(configPath, []byte("addons_path: \"bad\\npath\"\n"+readConfigDbBlock(t, "sqlite", writeTempSqliteDB(t))), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunInvalidDatabaseDsnControl(t *testing.T) {
	dsn := "postgres://user:pass@localhost/db\n"
	configPath := writeTempConfigWithDSN(t, "postgres", dsn, "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunDbDialectConflict(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "mysql", "postgres://user:pass@localhost/db", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunConfigSymlink(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	linkPath := filepath.Join(t.TempDir(), "config-link.yaml")
	if err := os.Symlink(configPath, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	output, code := runCLI(t, "run", "--config", linkPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunConfigMissingFields(t *testing.T) {
	addons := filepath.Join(t.TempDir(), "addons")
	if err := os.MkdirAll(addons, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	configPath := writeRawConfig(t, "addons_path: \""+addons+"\"\ndb:\n  dialect: postgres\n")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunAddonsPathListSeparator(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "a:b")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunConfigUnreadable(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	if err := os.Chmod(configPath, 0o000); err != nil {
		t.Fatalf("chmod config: %v", err)
	}
	if file, err := os.Open(configPath); err == nil {
		_ = file.Close()
		t.Skip("config file is readable; skipping")
	}
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunConfigInvalidYAML(t *testing.T) {
	configPath := writeRawConfig(t, "addons_path: [\n")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastLinesEqual(t, output, []string{
		"ERROR: invalid config",
		"REASON: invalid config format (YAML parse failed)",
		"NEXT: fix the config format and rerun 'choysum run'",
	})
}

func TestCLIErrorBlockLastOutput_InitCommandRemoved(t *testing.T) {
	output, code := runCLI(t, "init", "--non-interactive")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(output, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", output)
	}
}

func TestCLIErrorBlockRedactsDsn_Run(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "postgres", "postgres://user:secretpass@127.0.0.1:1/db?connect_timeout=1", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
	assertLastErrorBlock(t, output)
	if strings.Contains(output, "secretpass") {
		t.Fatalf("expected dsn to be redacted")
	}
}

func TestCLIErrorBlockRedactsKeyValueDsn_Run(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "postgres", "host=127.0.0.1 user=choysum password=secretpass dbname=choysum connect_timeout=1", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
	assertLastErrorBlock(t, output)
	if strings.Contains(output, "secretpass") {
		t.Fatalf("expected dsn to be redacted")
	}
}

func TestCLIErrorBlockRedactsKeyValueDsnAliases_Run(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "postgres", "host=127.0.0.1 user=choysum Pass=secretpass pwd=secretpass", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
	assertLastErrorBlock(t, output)
	if strings.Contains(output, "secretpass") {
		t.Fatalf("expected dsn to be redacted")
	}
}

func TestCLIErrorBlockRedactsKeyValueDsnAliasesRepeated_Run(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "postgres", "host=127.0.0.1 user=choysum PASSWORD=secretpass pass=secretpass pwd=secretpass", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
	assertLastErrorBlock(t, output)
	if strings.Contains(output, "secretpass") {
		t.Fatalf("expected dsn to be redacted")
	}
}

func TestCLIErrorBlockRedactsUrlUserinfoSpecialChars_Run(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "postgres", "postgres://user:sec%40ret%3Apass@127.0.0.1:1/db?connect_timeout=1", "")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
	assertLastErrorBlock(t, output)
	if strings.Contains(output, "sec@ret:pass") {
		t.Fatalf("expected dsn to be redacted")
	}
}

func writeTempRunConfig(t *testing.T) string {
	return writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
}

func isTerminal(t *testing.T) bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		t.Fatalf("stat stdin: %v", err)
	}
	return info.Mode()&os.ModeCharDevice != 0
}
