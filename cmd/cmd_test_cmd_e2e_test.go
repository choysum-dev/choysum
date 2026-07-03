package cmd

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"io"
	"net"

	clicompat "github.com/choysum-dev/choysum/internal/cli/compat"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"

	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"syscall"
	"testing"

	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
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

func TestCLIErrorBlockLastOutput_RunModulesPathMissing(t *testing.T) {
	t.Skip("run no longer performs interactive bootstrap when modules_path is omitted")
}

func TestCLIErrorBlockLastOutput_RunModulesPathUnreadable(t *testing.T) {
	modulesDir := filepath.Join(t.TempDir(), "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	if err := os.Chmod(modulesDir, 0o000); err != nil {
		t.Skipf("chmod modules dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(modulesDir, 0o755)
	})
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), modulesDir)
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunModulesPathSymlink(t *testing.T) {
	modulesDir := filepath.Join(t.TempDir(), "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	linkPath := filepath.Join(t.TempDir(), "modules-link")
	if err := os.Symlink(modulesDir, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), linkPath)
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunModulesPathWhitespace(t *testing.T) {
	t.Skip("whitespace-only modules_path is now normalized to empty by normalizePathRelativeToConfig and falls back to the default; no longer an error case")
}

func TestCLIErrorBlockLastOutput_RunModulesPathControlChar(t *testing.T) {
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	if err := os.WriteFile(configPath, []byte("modules_path: \"bad\\npath\"\n"+readConfigDbBlock(t, "sqlite", writeTempSqliteDB(t))), 0o600); err != nil {
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
	modules := filepath.Join(t.TempDir(), "modules")
	if err := os.MkdirAll(modules, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	configPath := writeRawConfig(t, "modules_path: \""+modules+"\"\ndb:\n  dialect: postgres\n")
	output, code := runCLI(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
	assertLastErrorBlock(t, output)
}

func TestCLIErrorBlockLastOutput_RunModulesPathListSeparator(t *testing.T) {
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
	configPath := writeRawConfig(t, "modules_path: [\n")
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

func TestCLIErrorBlockIsLastOutput(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "run", "--config", " ")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	assertLastErrorBlock(t, stderr)
}

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
	cmd := NewCommander(context.Background(), "test-version")
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

func writeTempConfigWithDSN(t *testing.T, dialect, dsn, modulesPath string) string {
	tmpDir := t.TempDir()
	defaultChoysumPath := filepath.Join(tmpDir, ".choysum")
	distPath := filepath.Join(tmpDir, "dist")
	dsn = normalizeConfigSQLiteDSN(dialect, dsn)
	if modulesPath == "" {
		modulesPath = filepath.Join(tmpDir, "modules")
		if err := os.MkdirAll(modulesPath, 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
	}
	if err := os.MkdirAll(distPath, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := fmt.Sprintf("default_choysum_path: %s\nmodules_path: %s\ndist_path: %s\ndb:\n  dialect: %s\n  dsn: %s\n", strconv.Quote(defaultChoysumPath), strconv.Quote(modulesPath), strconv.Quote(distPath), dialect, strconv.Quote(dsn))
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeTempInitializedRunConfig(t *testing.T, enabledTLS bool) (string, string, string) {
	tmpDir := t.TempDir()
	modulesPath := filepath.Join(tmpDir, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
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
	content := fmt.Sprintf("default_choysum_path: %s\nmodules_path: %s\ndist_path: %s\ndb:\n  dialect: sqlite\n  dsn: %s\nserver:\n  bindAddress: %s\n  port: %d\n  enabledTLS: %t\nauth:\n  enabled: false\n",
		strconv.Quote(defaultChoysumPath),
		strconv.Quote(modulesPath),
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
func TestCLIInitCommandRemoved(t *testing.T) {
	output, code := runCLI(t, "init")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", code, output)
	}
	if !strings.Contains(output, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", output)
	}
}
func TestCLIInitCommandRemovedStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "init", "--non-interactive")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", stderr)
	}
}
func TestCLIWarnBeforeErrorBlock(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real: %v", err)
	}
	linkDir := filepath.Join(base, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	dsn := filepath.Join(linkDir, "missing.db")
	configPath := writeTempConfigWithDSN(t, "sqlite", dsn, "")

	stdout, stderr, code := runCLISeparated(t, "run", "--config", configPath)
	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR:") {
		t.Fatalf("expected error block, got %q", stderr)
	}

	lines := lastNonEmptyLines(stderr)
	warnIndex := -1
	errorIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "WARN: sqlite parent directory is a symlink") {
			warnIndex = i
		}
		if strings.HasPrefix(line, "ERROR:") {
			errorIndex = i
		}
	}
	if warnIndex == -1 {
		t.Fatalf("expected symlink warning in stderr")
	}
	if errorIndex == -1 {
		t.Fatalf("expected error line in stderr")
	}
	if warnIndex >= errorIndex {
		t.Fatalf("expected warning before error block")
	}
}

type remoteCatalogModule struct {
	Name             string            `json:"name"`
	LatestVersion    string            `json:"latestVersion"`
	Description      string            `json:"description,omitempty"`
	Versions         []string          `json:"versions,omitempty"`
	VersionCLIRanges map[string]string `json:"versionCLIRanges,omitempty"`
}

func buildRemoteRegistryTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		content := files[path]
		hdr := &tar.Header{Name: path, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func npmSHA512Integrity(data []byte) string {
	h := sha512.New()
	h.Write(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func startRemoteRegistryCatalogAndNPMServer(t *testing.T, moduleName, latestVersion string) *httptest.Server {
	t.Helper()

	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		t.Fatal("module name must not be empty")
	}

	latestVersion = strings.TrimSpace(latestVersion)
	if latestVersion == "" {
		latestVersion = "v0.2.0"
	}
	if !strings.HasPrefix(latestVersion, "v") {
		latestVersion = "v" + latestVersion
	}

	npmVersion := strings.TrimPrefix(latestVersion, "v")
	packageName := "@acme/choysum-" + moduleName
	metadataPathEscaped := "/" + url.PathEscape(packageName)
	metadataPathPlain := "/" + packageName
	tarballPath := fmt.Sprintf("/tarballs/choysum-%s-%s.tgz", moduleName, npmVersion)

	packageJSON := fmt.Sprintf(`{"name":"%s","version":"%s","description":"%s module","license":"Apache-2.0","author":"test","choysum":{"moduleName":"%s","application":"%s","category":"test","depends":[]}}`, packageName, npmVersion, moduleName, moduleName, moduleName)
	tarballBody := buildRemoteRegistryTarGz(t, map[string]string{"package/package.json": packageJSON})
	integrity := npmSHA512Integrity(tarballBody)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := "http://" + r.Host
		switch {
		case r.URL.Path == "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modules": map[string]any{
					moduleName: map[string]any{
						"moduleId":      moduleName,
						"description":   moduleName + " module",
						"latestVersion": latestVersion,
						"package":       packageName,
						"source": map[string]any{
							"type":      "npm",
							"registry":  baseURL,
							"package":   packageName,
							"version":   latestVersion,
							"tarball":   baseURL + tarballPath,
							"integrity": integrity,
						},
						"versions": map[string]any{
							latestVersion: map[string]any{
								"registry":  baseURL,
								"package":   packageName,
								"tarball":   baseURL + tarballPath,
								"integrity": integrity,
								"choysum": map[string]any{
									"cli": ">=0.0.0-0 <0.0.0",
								},
							},
						},
					},
				},
			})
		case r.URL.EscapedPath() == metadataPathEscaped || r.URL.Path == metadataPathPlain:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist-tags": map[string]any{"latest": npmVersion},
				"versions": map[string]any{
					npmVersion: map[string]any{
						"name":        packageName,
						"version":     npmVersion,
						"description": moduleName + " module",
						"license":     "Apache-2.0",
						"author":      "test",
						"choysum": map[string]any{
							"moduleName":  moduleName,
							"application": moduleName,
							"category":    "test",
							"depends":     []string{},
						},
						"dist": map[string]any{
							"tarball":   baseURL + tarballPath,
							"integrity": integrity,
						},
					},
				},
			})
		case r.URL.Path == tarballPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(tarballBody)
		default:
			http.NotFound(w, r)
		}
	}))
}

func startRemoteRegistryCatalogServer(t *testing.T, modules []remoteCatalogModule) *httptest.Server {
	t.Helper()

	byName := make(map[string]remoteCatalogModule, len(modules))
	for _, item := range modules {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			t.Fatalf("module name must not be empty")
		}
		sort.Strings(item.Versions)
		byName[item.Name] = item
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		modulesPayload := map[string]any{}
		for _, item := range byName {
			versions := item.Versions
			if len(versions) == 0 {
				versions = []string{item.LatestVersion}
			}
			versionPayload := map[string]any{}
			for _, version := range versions {
				version = strings.TrimSpace(version)
				if version == "" {
					continue
				}
				cliRange := strings.TrimSpace(item.VersionCLIRanges[version])
				if cliRange == "" {
					cliRange = ">=0.0.0-0 <0.0.0"
				}
				versionPayload[version] = map[string]any{
					"tarball":   "https://registry.npmjs.org/@acme/choysum-" + item.Name + "/-/choysum-" + item.Name + "-" + strings.TrimPrefix(version, "v") + ".tgz",
					"integrity": "sha512-" + item.Name + "-" + strings.TrimPrefix(version, "v"),
					"package":   "@acme/choysum-" + item.Name,
					"choysum": map[string]any{
						"cli": cliRange,
					},
				}
			}
			modulesPayload[item.Name] = map[string]any{
				"moduleId":      item.Name,
				"description":   item.Description,
				"latestVersion": item.LatestVersion,
				"package":       "@acme/choysum-" + item.Name,
				"versions":      versionPayload,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"modules": modulesPayload})
	})

	return httptest.NewServer(mux)
}

func setModuleCatalogIndexURLForCLIConfig(t *testing.T, configPath, indexURL string) {
	t.Helper()

	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body = append(body, []byte(fmt.Sprintf("module_catalog_index_url: %q\n", indexURL))...)
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func setLegacyRegistryURLConfigForCLI(t *testing.T, configPath string) {
	t.Helper()

	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body = append(body, []byte("registry_index_url: \"https://index.legacy.dev/v1/index.json\"\n")...)
	body = append(body, []byte("registries:\n  official:\n    url: \"https://index.legacy.dev/v1/index.json\"\n")...)
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestCLIModuleRemoteSearchListInfo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{Name: "auth", LatestVersion: "v1.2.3", Description: "Authentication", Versions: []string{"v1.0.0", "v1.2.3"}},
		{Name: "partner", LatestVersion: "v0.9.0", Description: "Partner management", Versions: []string{"v0.9.0"}},
	})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	listOutput, listCode := runCLI(t, "module", "list", "--remote", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if listCode != 0 {
		t.Fatalf("module list --remote failed, code=%d output=%s", listCode, listOutput)
	}
	if !strings.Contains(listOutput, "auth") || !strings.Contains(listOutput, "partner") {
		t.Fatalf("expected remote module names in list output, got %q", listOutput)
	}
	if !strings.Contains(listOutput, "v1.2.3") {
		t.Fatalf("expected latest version in list output, got %q", listOutput)
	}

	searchOutput, searchCode := runCLI(t, "module", "search", "au", "--remote", "--config", configPath)
	if searchCode != 0 {
		t.Fatalf("module search --remote failed, code=%d output=%s", searchCode, searchOutput)
	}
	if !strings.Contains(searchOutput, "auth") {
		t.Fatalf("expected auth in search output, got %q", searchOutput)
	}
	if strings.Contains(searchOutput, "partner") {
		t.Fatalf("did not expect partner in filtered search output, got %q", searchOutput)
	}

	infoOutput, infoCode := runCLI(t, "module", "info", "auth", "--remote", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if infoCode != 0 {
		t.Fatalf("module info --remote failed, code=%d output=%s", infoCode, infoOutput)
	}
	if !strings.Contains(infoOutput, `"name": "auth"`) || !strings.Contains(infoOutput, `"latestVersion": "v1.2.3"`) {
		t.Fatalf("unexpected info output: %q", infoOutput)
	}
}

func TestCLIModuleRemoteInfoNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0"}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "info", "missing", "--remote", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for missing module, output=%s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Fatalf("expected not found message, got %q", output)
	}
}

func TestCLIModuleRemoteListRequiresCompatVersionInDev(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0", Versions: []string{"v1.0.0"}}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "list", "--remote", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when cli compat version is unresolved, output=%s", output)
	}
	if !strings.Contains(output, "ERR_CLI_COMPAT_VERSION_UNRESOLVED") {
		t.Fatalf("expected unresolved cli compat error, got %q", output)
	}
}

func TestCLIModuleRemoteListAllAllowsUnresolvedCompatVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0", Versions: []string{"v1.0.0"}}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "list", "--remote", "--all", "--config", configPath)
	if code != 0 {
		t.Fatalf("expected list --remote --all to succeed without compat version, code=%d output=%s", code, output)
	}
	if !strings.Contains(output, "auth") {
		t.Fatalf("expected module name in list output, got %q", output)
	}
	if !strings.Contains(output, "WARN_CLI_COMPAT_FILTER_SKIPPED") {
		t.Fatalf("expected compat filter skipped warning, got %q", output)
	}
}

func TestCLIModuleRemoteInfoRequiresCompatVersionInDev(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0", Versions: []string{"v1.0.0"}}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "info", "auth", "--remote", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when cli compat version is unresolved, output=%s", output)
	}
	if !strings.Contains(output, "ERR_CLI_COMPAT_VERSION_UNRESOLVED") {
		t.Fatalf("expected unresolved cli compat error, got %q", output)
	}
}

func TestCLIModuleRemoteInfoAllAllowsUnresolvedCompatVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0", Versions: []string{"v1.0.0"}}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "info", "auth", "--remote", "--all", "--config", configPath)
	if code != 0 {
		t.Fatalf("expected module info --remote --all to succeed without compat version, code=%d output=%s", code, output)
	}
	if !strings.Contains(output, `"name": "auth"`) {
		t.Fatalf("expected module info payload for auth, got %q", output)
	}
	if !strings.Contains(output, "WARN_CLI_COMPAT_FILTER_SKIPPED") {
		t.Fatalf("expected compat filter skipped warning, got %q", output)
	}
}

func TestCLIModuleRemoteRejectsLegacyRegistryURLConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	setLegacyRegistryURLConfigForCLI(t, configPath)

	output, code := runCLI(t, "module", "list", "--remote", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for legacy registry config, output=%s", output)
	}
	if !strings.Contains(output, "legacy module catalog config keys are no longer supported") {
		t.Fatalf("expected legacy module catalog rejection, got %q", output)
	}
	if !strings.Contains(output, "module_catalog_index_url") {
		t.Fatalf("expected module_catalog_index_url guidance, got %q", output)
	}
}

func TestCLIInstallLocalMissingPreservesFallbackCauseWithGuidance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "install", "missing", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected install to fail for missing local module, output=%s", output)
	}
	if !strings.Contains(output, "not found locally and registry fallback failed") {
		t.Fatalf("expected registry fallback failure to be preserved in output, got %q", output)
	}
	if strings.Contains(output, "module missing not found in modules path") {
		t.Fatalf("expected output not to collapse to local-only missing message, got %q", output)
	}
	if !strings.Contains(output, "choysum module fetch <module>@<version>") {
		t.Fatalf("expected module fetch guidance in output, got %q", output)
	}
	if !strings.Contains(output, "choysum install <module>@<version>") {
		t.Fatalf("expected registry install guidance in output, got %q", output)
	}
}

func TestCLIUninstallMissingModuleReportsFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	const missingModule = "__missing_copilot_module__"
	output, code := runCLI(t, "uninstall", missingModule, "--config", configPath)
	if code == 0 {
		t.Fatalf("expected uninstall to fail for missing module, output=%s", output)
	}
	if !strings.Contains(output, "module uninstall failed") {
		t.Fatalf("expected uninstall failure wrapper, got %q", output)
	}
	if !strings.Contains(output, missingModule) {
		t.Fatalf("expected missing module name in output, got %q", output)
	}
}

func TestCLIUpgradeMissingModuleReportsFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	const missingModule = "__missing_copilot_module__"
	output, code := runCLI(t, "upgrade", missingModule, "--config", configPath)
	if code == 0 {
		t.Fatalf("expected upgrade to fail for missing module, output=%s", output)
	}
	if !strings.Contains(output, "module upgrade failed") {
		t.Fatalf("expected upgrade failure wrapper, got %q", output)
	}
	if !strings.Contains(output, missingModule) {
		t.Fatalf("expected missing module input in output, got %q", output)
	}
}

func TestCLIUpgradeRejectsLegacyAliasSyntax(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "upgrade", "corp/demo@v1.0.0", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected registry upgrade to fail for legacy alias syntax, output=%s", output)
	}
	if !strings.Contains(output, "is no longer supported") {
		t.Fatalf("expected legacy alias syntax error, got %q", output)
	}
}

func TestCLIUpgradeFlowWithGlobalRegistryIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "upgrade", "demo@latest", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if code != 0 {
		t.Fatalf("upgrade failed, code=%d output=%s", code, output)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var upgraded meta.IrModule
	if err := db.Where("name = ?", "demo").Take(&upgraded).Error; err != nil {
		t.Fatalf("query upgraded module failed: %v", err)
	}
	if upgraded.Version != "v0.2.0" {
		t.Fatalf("unexpected upgraded version: %q", upgraded.Version)
	}
	if upgraded.Status != meta.Installed {
		t.Fatalf("unexpected upgraded status: %v", upgraded.Status)
	}

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock lookup failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	binding, ok, err := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath)).LookupBinding(workspaceStateRoot, "demo")
	if err != nil {
		t.Fatalf("lookup lock binding failed: %v", err)
	}
	if !ok {
		t.Fatal("expected lock binding for upgraded module")
	}
	if binding.OriginType != internalorigin.OriginTypeRegistry {
		t.Fatalf("unexpected origin type: %q", binding.OriginType)
	}
	if binding.OriginRef != "demo@v0.2.0" {
		t.Fatalf("unexpected origin ref: %q", binding.OriginRef)
	}
	if binding.ResolvedVersion != "v0.2.0" {
		t.Fatalf("unexpected resolved version: %q", binding.ResolvedVersion)
	}

	packageJSONRaw, err := os.ReadFile(filepath.Join(modulesPath, "demo", "package.json"))
	if err != nil {
		t.Fatalf("read fetched package.json failed: %v", err)
	}
	packageJSON := string(packageJSONRaw)
	if !strings.Contains(packageJSON, `"version":"0.2.0"`) && !strings.Contains(packageJSON, `"version": "0.2.0"`) {
		t.Fatalf("expected fetched package.json version 0.2.0, got %q", packageJSON)
	}
}

func TestCLIUpgradeLatestRequiresCompatVersionInDev(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "upgrade", "demo@latest", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when cli compat version is unresolved, output=%s", output)
	}
	if !strings.Contains(output, "ERR_CLI_COMPAT_VERSION_UNRESOLVED") {
		t.Fatalf("expected unresolved cli compat error, got %q", output)
	}
}

func TestCLIUpgradeLocalRegistryBindingUsesCompatFilter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock binding seed failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath))
	if err := store.UpsertBinding(workspaceStateRoot, internalorigin.Binding{
		ModuleName:      "demo",
		OriginType:      internalorigin.OriginTypeRegistry,
		OriginRef:       "demo@v0.1.0",
		ResolvedVersion: "v0.1.0",
		LocalPath:       filepath.Join(modulesPath, "demo"),
	}); err != nil {
		t.Fatalf("seed registry binding failed: %v", err)
	}

	output, code := runCLI(t, "upgrade", "demo", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if code != 0 {
		t.Fatalf("upgrade failed, code=%d output=%s", code, output)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var upgraded meta.IrModule
	if err := db.Where("name = ?", "demo").Take(&upgraded).Error; err != nil {
		t.Fatalf("query upgraded module failed: %v", err)
	}
	if upgraded.Version != "v0.2.0" {
		t.Fatalf("unexpected upgraded version: %q", upgraded.Version)
	}
}

func TestCLIUpgradeLocalRegistryBindingRequiresCompatVersionInDev(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock binding seed failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath))
	if err := store.UpsertBinding(workspaceStateRoot, internalorigin.Binding{
		ModuleName:      "demo",
		OriginType:      internalorigin.OriginTypeRegistry,
		OriginRef:       "demo@v0.1.0",
		ResolvedVersion: "v0.1.0",
		LocalPath:       filepath.Join(modulesPath, "demo"),
	}); err != nil {
		t.Fatalf("seed registry binding failed: %v", err)
	}

	output, code := runCLI(t, "upgrade", "demo", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when cli compat version is unresolved, output=%s", output)
	}
	if !strings.Contains(output, "ERR_CLI_COMPAT_VERSION_UNRESOLVED") {
		t.Fatalf("expected unresolved cli compat error, got %q", output)
	}
}

func TestCLIUpgradeLocalRegistryBindingNoCompatibleVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock binding seed failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath))
	if err := store.UpsertBinding(workspaceStateRoot, internalorigin.Binding{
		ModuleName:      "demo",
		OriginType:      internalorigin.OriginTypeRegistry,
		OriginRef:       "demo@v0.1.0",
		ResolvedVersion: "v0.1.0",
		LocalPath:       filepath.Join(modulesPath, "demo"),
	}); err != nil {
		t.Fatalf("seed registry binding failed: %v", err)
	}

	output, code := runCLI(t, "upgrade", "demo", "--cli-compat-version", "v9.0.0", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for no-compatible registry upgrade, output=%s", output)
	}
	if !strings.Contains(output, "ERR_MODULE_NO_COMPATIBLE_VERSION") {
		t.Fatalf("expected no-compatible-version error, got %q", output)
	}
}

func TestCLIInstallLatestRequiresCompatVersionInDev(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(clicompat.CLICompatVersionEnv, "")

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), modulesPath)
	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "install", "demo@latest", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when cli compat version is unresolved, output=%s", output)
	}
	if !strings.Contains(output, "ERR_CLI_COMPAT_VERSION_UNRESOLVED") {
		t.Fatalf("expected unresolved cli compat error, got %q", output)
	}
}

func TestCLIInstallLatestResolvesCompatibleVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "install", "demo@latest", "--cli-compat-version", "v0.0.0-0", "--config", configPath)
	if code != 0 {
		t.Fatalf("install failed, code=%d output=%s", code, output)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var installed meta.IrModule
	if err := db.Where("name = ?", "demo").Take(&installed).Error; err != nil {
		t.Fatalf("query installed module failed: %v", err)
	}
	if installed.Version != "v0.2.0" {
		t.Fatalf("unexpected installed version: %q", installed.Version)
	}
	if installed.Status != meta.Installed {
		t.Fatalf("unexpected installed status: %v", installed.Status)
	}

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock lookup failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	binding, ok, err := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath)).LookupBinding(workspaceStateRoot, "demo")
	if err != nil {
		t.Fatalf("lookup lock binding failed: %v", err)
	}
	if !ok {
		t.Fatal("expected lock binding for installed module")
	}
	if strings.TrimSpace(binding.LocalPath) == "" {
		t.Fatalf("expected lock binding to record local path, got %#v", binding)
	}

	packageJSONRaw, err := os.ReadFile(filepath.Join(modulesPath, "demo", "package.json"))
	if err != nil {
		t.Fatalf("read installed package.json failed: %v", err)
	}
	packageJSON := string(packageJSONRaw)
	if !strings.Contains(packageJSON, `"version":"0.2.0"`) && !strings.Contains(packageJSON, `"version": "0.2.0"`) {
		t.Fatalf("expected installed package.json version 0.2.0, got %q", packageJSON)
	}
}

func TestCLIModulePurgeRequiresUninstallWhenInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	if output, code := runCLI(t, "module", "fetch", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module fetch failed, code=%d output=%s", code, output)
	}

	output, code := runCLI(t, "module", "purge", "demo", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected purge to fail for installed module, output=%s", output)
	}
	if !strings.Contains(output, "run 'choysum uninstall demo' before purge") {
		t.Fatalf("expected uninstall guidance in purge output, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "demo")); err != nil {
		t.Fatalf("expected module directory to remain after blocked purge, err=%v", err)
	}
}

func TestCLIFetchUninstallPurgeFlowWithGlobalRegistryIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "demo", LatestVersion: "v0.1.0"}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	if output, code := runCLI(t, "module", "fetch", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module fetch failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "uninstall", "demo", "--config", configPath); code != 0 {
		t.Fatalf("uninstall failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "module", "purge", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module purge failed, code=%d output=%s", code, output)
	}

	if _, err := os.Stat(filepath.Join(modulesPath, "demo")); !os.IsNotExist(err) {
		t.Fatalf("expected purged module dir to be removed, stat err=%v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock lookup failed: %v", err)
	}
	if _, ok, err := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath)).LookupBinding(workspaceStateRoot, "demo"); err != nil {
		t.Fatalf("lookup binding after purge failed: %v", err)
	} else if ok {
		t.Fatal("expected module binding to be removed after purge")
	}
}

func seedModuleStatusForCLI(t *testing.T, dbPath, moduleName string, status meta.Status) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.IrModule{}); err != nil {
		t.Fatalf("auto-migrate ir module: %v", err)
	}

	var count int64
	if err := db.Model(&meta.IrModule{}).Where("name = ?", moduleName).Count(&count).Error; err != nil {
		t.Fatalf("count module fixture failed: %v", err)
	}
	if count == 0 {
		record := &meta.IrModule{
			Name:           moduleName,
			ApplicationStr: moduleName,
			Status:         status,
			Version:        "v0.1.0",
			Path:           moduleName,
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("create module status fixture: %v", err)
		}
		return
	}

	var existing meta.IrModule
	err = db.Where("name = ?", moduleName).Take(&existing).Error
	if err != nil {
		t.Fatalf("query module fixture failed: %v", err)
	}

	existing.Status = status
	if err := db.Save(&existing).Error; err != nil {
		t.Fatalf("update module status fixture: %v", err)
	}
}
func TestCLIErrorBlockUsesStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "run", "--config", " ")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	assertLastErrorBlock(t, stderr)
}
func TestCLIE2EWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "e2e", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLIInitErrorBlockIsLastAndStderrOnly(t *testing.T) {
	stdout, stderr, code := runCLISeparated(t, "init", "--admin-password-stdin")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command \"init\"") {
		t.Fatalf("expected unknown init command error, got %q", stderr)
	}
}
func TestCLIInstallWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "install", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLIRunInfoWritesToStderrOnly(t *testing.T) {
	configPath, _, _ := writeTempInitializedRunConfigWithDB(t, false)

	stdout, stderr, _ := runCLIUntilLineSeparated(t, func(line string) bool {
		return strings.Contains(line, "http server listening")
	}, "run", "--config", configPath)

	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "http server listening") {
		t.Fatalf("expected stderr to contain listening log, got %q", stderr)
	}
	if strings.Contains(stderr, "ERROR: ") || strings.Contains(stderr, "REASON: ") {
		t.Fatalf("did not expect error block in stderr, got %q", stderr)
	}
	if strings.Contains(stderr, "server starting; NEXT: open") {
		t.Fatalf("did not expect CLI startup hint in stderr, got %q", stderr)
	}
}
func TestCLITestWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
func TestCLITypecheckWritesToStderrOnlyOnUsageError(t *testing.T) {
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, "")

	stdout, stderr, code := runCLISeparated(t, "test", "typecheck", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout empty, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("expected stderr to have output")
	}
}
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
