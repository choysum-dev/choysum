// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	leasemodel "github.com/choysum-dev/choysum/internal/state/lease/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const cliE2EEnv = "CHOYSUM_CLI_E2E"

func cliE2EHelperEnv() []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		switch key {
		case cliE2EEnv, "CHOYSUM_DEFAULT_CHOYSUM_PATH", "CHOYSUM_DB_DIALECT", "CHOYSUM_DB_DSN", "CHOYSUM_AUTH_INTERNAL_KEY", "CHOYSUM_COMPILE_MINIFY", "CHOYSUM_COMPILE_SOURCEMAP", "CHOYSUM_SERVER_HOT_RELOAD":
			continue
		}
		env = append(env, entry)
	}
	env = append(env, cliE2EEnv+"=1")
	return env
}

func TestCLIErrorBlockHelper(t *testing.T) {
	if os.Getenv(cliE2EEnv) != "1" {
		return
	}
	args := helperArgs(os.Args)
	cmd := NewCommander(context.Background())
	cmd.rootCmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func helperArgs(args []string) []string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			if i+1 < len(args) {
				return args[i+1:]
			}
			return []string{}
		}
	}
	return []string{}
}

func runCLI(t *testing.T, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(output), exitErr.ExitCode()
	}
	return string(output), 1
}

func runCLIInDir(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(output), exitErr.ExitCode()
	}
	return string(output), 1
}

func runCLIWithStdin(t *testing.T, stdin string, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()
	cmd.Stdin = strings.NewReader(stdin)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(output), exitErr.ExitCode()
	}
	return string(output), 1
}

func runCLISeparated(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	if err == nil {
		return stdoutBuf.String(), stderrBuf.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode()
	}
	return stdoutBuf.String(), stderrBuf.String(), 1
}

func runCLIUntilLine(t *testing.T, waitFor func(string) bool, args ...string) (string, int) {
	return runCLIUntilLineWithTimeout(t, 10*time.Second, waitFor, args...)
}

func runCLIUntilLineWithTimeout(t *testing.T, timeout time.Duration, waitFor func(string) bool, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	stdoutDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
		close(stdoutDone)
	}()

	foundCh := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		tee := io.TeeReader(stderrPipe, &stderrBuf)
		scanner := bufio.NewScanner(tee)
		for scanner.Scan() {
			line := scanner.Text()
			if waitFor != nil && waitFor(line) {
				select {
				case <-foundCh:
				default:
					close(foundCh)
				}
			}
		}
		close(stderrDone)
	}()

	select {
	case <-foundCh:
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		case <-stderrDone:
		}
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
	}

	err = cmd.Wait()
	<-stdoutDone
	<-stderrDone

	output := stdoutBuf.String() + stderrBuf.String()
	if err == nil {
		return output, 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return output, exitErr.ExitCode()
	}
	return output, 1
}

func runCLIUntilLineSeparated(t *testing.T, waitFor func(string) bool, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	stdoutDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
		close(stdoutDone)
	}()

	foundCh := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		tee := io.TeeReader(stderrPipe, &stderrBuf)
		scanner := bufio.NewScanner(tee)
		for scanner.Scan() {
			line := scanner.Text()
			if waitFor != nil && waitFor(line) {
				select {
				case <-foundCh:
				default:
					close(foundCh)
				}
			}
		}
		close(stderrDone)
	}()

	select {
	case <-foundCh:
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		case <-stderrDone:
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
	}

	err = cmd.Wait()
	<-stdoutDone
	<-stderrDone

	if err == nil {
		return stdoutBuf.String(), stderrBuf.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode()
	}
	return stdoutBuf.String(), stderrBuf.String(), 1
}

func writeTempSqliteDB(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "db.sqlite")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create sqlite db: %v", err)
	}
	_ = file.Close()
	return path
}

func writeInitializedSqliteDB(t *testing.T) string {
	path := writeTempSqliteDB(t)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&metadata.IrSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&metadata.IrSetting{Key: "system.init.done", Value: "true"}).Error; err != nil {
		t.Fatalf("insert init marker: %v", err)
	}
	return path
}

func writeSqliteDBWithInitLease(t *testing.T) string {
	path := writeTempSqliteDB(t)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&leasemodel.IrLockLease{}); err != nil {
		t.Fatalf("migrate lease: %v", err)
	}
	leaseRow := &leasemodel.IrLockLease{
		Resource:  "system:init",
		OwnerId:   "busy-owner",
		ExpiresAt: time.Now().Add(30 * time.Second),
	}
	if err := db.Create(leaseRow).Error; err != nil {
		t.Fatalf("insert lease: %v", err)
	}
	return path
}

func writeSqliteDBWithInitSetting(t *testing.T, value *string) string {
	path := writeTempSqliteDB(t)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&metadata.IrSetting{}); err != nil {
		t.Fatalf("migrate settings: %v", err)
	}
	if value != nil {
		if err := db.Create(&metadata.IrSetting{Key: "system.init.done", Value: *value}).Error; err != nil {
			t.Fatalf("insert init setting: %v", err)
		}
	}
	return path
}

func writeTempConfigWithDSN(t *testing.T, dialect, dsn, addonsPath string) string {
	tmpDir := t.TempDir()
	defaultChoysumPath := filepath.Join(tmpDir, ".choysum")
	distPath := filepath.Join(tmpDir, "dist")
	dsn = normalizeConfigSQLiteDSN(dialect, dsn)
	if addonsPath == "" {
		addonsPath = filepath.Join(tmpDir, "addons")
		if err := os.MkdirAll(addonsPath, 0o755); err != nil {
			t.Fatalf("mkdir addons: %v", err)
		}
	}
	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := fmt.Sprintf("default_choysum_path: %s\naddons_path: %s\ndist_path: %s\ndb:\n  dialect: %s\n  dsn: %s\n", strconv.Quote(defaultChoysumPath), strconv.Quote(addonsPath), strconv.Quote(distPath), dialect, strconv.Quote(dsn))
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeTempInitializedRunConfig(t *testing.T, enabledTLS bool) (string, string, string) {
	tmpDir := t.TempDir()
	addonsPath := filepath.Join(tmpDir, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("mkdir addons: %v", err)
	}
	distPath := filepath.Join(tmpDir, "dist")
	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	defaultChoysumPath := filepath.Join(tmpDir, ".choysum")
	dbPath := writeInitializedSqliteDB(t)
	bindAddress := "127.0.0.1"
	port := findFreePort(t)
	addr := net.JoinHostPort(bindAddress, strconv.Itoa(port))
	configPath := filepath.Join(tmpDir, "config.yaml")
	dbDSN := normalizeConfigSQLiteDSN("sqlite", dbPath)
	content := fmt.Sprintf("default_choysum_path: %s\naddons_path: %s\ndist_path: %s\ndb:\n  dialect: sqlite\n  dsn: %s\nserver:\n  bindAddress: %s\n  port: %d\n  enabledTLS: %t\nauth:\n  enabled: false\n",
		strconv.Quote(defaultChoysumPath),
		strconv.Quote(addonsPath),
		strconv.Quote(distPath),
		strconv.Quote(dbDSN),
		strconv.Quote(bindAddress),
		port,
		enabledTLS,
	)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath, addr, dbPath
}

func writeTempInitializedRunConfigWithDB(t *testing.T, enabledTLS bool) (string, string, string) {
	return writeTempInitializedRunConfig(t, enabledTLS)
}

func sqliteTableExists(t *testing.T, dbPath, table string) bool {
	if strings.TrimSpace(dbPath) == "" {
		t.Fatalf("missing sqlite path")
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var count int
	if err := db.Raw("SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&count).Error; err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	return count > 0
}

func writeRawConfig(t *testing.T, content string) string {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func readConfigDbBlock(t *testing.T, dialect, dsn string) string {
	t.Helper()
	dsn = normalizeConfigSQLiteDSN(dialect, dsn)
	return fmt.Sprintf("db:\n  dialect: %s\n  dsn: %s\n", dialect, strconv.Quote(dsn))
}

func normalizeConfigSQLiteDSN(dialect, dsn string) string {
	if !strings.EqualFold(strings.TrimSpace(dialect), "sqlite") {
		return dsn
	}
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || strings.EqualFold(trimmed, ":memory:") {
		return dsn
	}
	if strings.Contains(trimmed, "?") || strings.HasPrefix(strings.ToLower(trimmed), "file:") || strings.Contains(trimmed, "://") {
		return dsn
	}
	if !filepath.IsAbs(trimmed) {
		return dsn
	}
	return fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", trimmed)
}

func findFreePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	addr, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener addr: %T", lis.Addr())
	}
	return addr.Port
}

func lastNonEmptyLines(output string) []string {
	parts := strings.Split(output, "\n")
	lines := make([]string, 0, len(parts))
	for _, line := range parts {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func assertLastErrorBlock(t *testing.T, output string) {
	lines := lastNonEmptyLines(output)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	last := lines[len(lines)-3:]
	if !strings.HasPrefix(last[0], "ERROR: ") {
		t.Fatalf("missing ERROR line: %q", last[0])
	}
	if !strings.HasPrefix(last[1], "REASON: ") {
		t.Fatalf("missing REASON line: %q", last[1])
	}
	if !strings.HasPrefix(last[2], "NEXT: ") {
		t.Fatalf("missing NEXT line: %q", last[2])
	}
}

func assertLastLinesEqual(t *testing.T, output string, expected []string) {
	lines := lastNonEmptyLines(output)
	if len(lines) < len(expected) {
		t.Fatalf("expected at least %d lines, got %d", len(expected), len(lines))
	}
	last := lines[len(lines)-len(expected):]
	for i := range expected {
		if last[i] != expected[i] {
			t.Fatalf("unexpected line %d: %q", i, last[i])
		}
	}
}

func extractReason(output string) string {
	lines := lastNonEmptyLines(output)
	if len(lines) < 2 {
		return ""
	}
	line := lines[len(lines)-2]
	if !strings.HasPrefix(line, "REASON: ") {
		return ""
	}
	return strings.TrimPrefix(line, "REASON: ")
}

func assertReasonInSet(t *testing.T, reason string, candidates []string) {
	for _, candidate := range candidates {
		if reason == candidate {
			return
		}
	}
	t.Fatalf("unexpected reason: %s", reason)
}
