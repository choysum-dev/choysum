// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/cli/initialsetup"
	"github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// installSetupWorkspace is an isolated config + sqlite DB installing the repo
// modules; the ESM cache is shared with `choysum test` runs.
type installSetupWorkspace struct {
	configPath string
	dbPath     string
	env        []string
}

func newInstallSetupWorkspace(t *testing.T) installSetupWorkspace {
	t.Helper()
	if testing.Short() {
		t.Skip("installs real modules")
	}
	modulesPath, err := filepath.Abs(filepath.Join("..", "modules"))
	if err != nil {
		t.Fatalf("modules path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "auth", "package.json")); err != nil {
		t.Skipf("repo modules not available: %v", err)
	}
	dbPath := writeTempSqliteDB(t)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	tmpRoot, err := tmpdir.CLITestTmpRoot()
	if err != nil {
		t.Fatalf("CLI test tmp root: %v", err)
	}
	root := filepath.Dir(configPath)
	home := filepath.Join(root, ".choysum")
	if err := tmpdir.LinkCLITestingPkgCache(home, tmpRoot); err != nil {
		t.Fatalf("link pkg cache: %v", err)
	}
	return installSetupWorkspace{
		configPath: configPath,
		dbPath:     dbPath,
		// Pin path env vars so an inherited shell override cannot share dirs between workspaces.
		env: []string{
			initialsetup.PasswordEnv + "=",
			"CHOYSUM_DIST_PATH=" + filepath.Join(root, "dist"),
			"CHOYSUM_TMP_PATH=" + filepath.Join(home, "tmp"),
		},
	}
}

func (w installSetupWorkspace) install(t *testing.T, extraEnv []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestCLIErrorBlockHelper", "--")
	cmd.Args = append(cmd.Args, "install", "auth", "--no-web", "--config", w.configPath)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = cliE2EHelperEnv(t, append(append([]string{}, w.env...), extraEnv...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run install: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

func (w installSetupWorkspace) db(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(w.dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func (w installSetupWorkspace) appSetting(t *testing.T, key string) (string, bool) {
	t.Helper()
	var rows []struct{ Value string }
	if err := w.db(t).Raw("SELECT value FROM auth_app_setting WHERE key = ?", key).Scan(&rows).Error; err != nil {
		t.Fatalf("read auth_app_setting %s: %v", key, err)
	}
	if len(rows) == 0 {
		return "", false
	}
	return rows[0].Value, true
}

type installSetupAdmin struct {
	PasswordHash string
	Email        *string
	LanguageCode string
}

func (w installSetupWorkspace) admin(t *testing.T) installSetupAdmin {
	t.Helper()
	var row installSetupAdmin
	err := w.db(t).Raw(`SELECT u.password_hash AS password_hash, u.email AS email, l.code AS language_code
		FROM auth_user u LEFT JOIN base_language l ON l.id = u.language_id
		WHERE u.username = ?`, initialsetup.AdminUsername).Scan(&row).Error
	if err != nil {
		t.Fatalf("read admin: %v", err)
	}
	return row
}

// assertBrowserLoginPassword checks the stored hash accepts what the web client
// sends for (password, admin) after stripping the "$CH$" marker.
func assertBrowserLoginPassword(t *testing.T, storedHash string, password string) {
	t.Helper()
	sum := sha256.Sum256([]byte(password + ":" + initialsetup.AdminUsername))
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(hex.EncodeToString(sum[:]))); err != nil {
		t.Fatalf("stored admin hash does not accept password %q: %v", password, err)
	}
}

func TestCLIInstallCompletesSetupWithRandomPassword(t *testing.T) {
	t.Parallel()
	ws := newInstallSetupWorkspace(t)

	stdout, stderr, code := ws.install(t, nil)
	if code != 0 {
		t.Fatalf("install exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	match := regexp.MustCompile(`Admin password : (\S+)\s+\(shown once`).FindStringSubmatch(stdout)
	if match == nil {
		t.Fatalf("random password not printed on stdout: %q", stdout)
	}
	password := match[1]
	if strings.Contains(stderr, password) {
		t.Fatal("random password must not reach stderr/logs")
	}
	if !strings.Contains(stdout, "Organization   : 主公司 · CNY · Asia/Shanghai") {
		t.Fatalf("unexpected organization summary: %q", stdout)
	}
	if v, ok := ws.appSetting(t, "setup.completed_at"); !ok || strings.TrimSpace(v) == "" {
		t.Fatal("setup.completed_at must be written")
	}
	admin := ws.admin(t)
	if admin.LanguageCode != "zh_CN" {
		t.Fatalf("admin language = %q, want zh_CN", admin.LanguageCode)
	}
	if admin.Email != nil && *admin.Email != "" {
		t.Fatalf("seed admin email must be cleared, got %q", *admin.Email)
	}
	assertBrowserLoginPassword(t, admin.PasswordHash, password)

	stdout, stderr, code = ws.install(t, nil, "--admin-password", "ignored-now")
	if code != 0 {
		t.Fatalf("rerun exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "already set up; admin password unchanged") {
		t.Fatalf("rerun on Ready should report unchanged password: %q", stdout)
	}
	if !strings.Contains(stderr, "--admin-password is visible") {
		t.Fatalf("flag password should warn on stderr: %q", stderr)
	}
	if got := ws.admin(t).PasswordHash; got != admin.PasswordHash {
		t.Fatal("admin password must not change once Ready")
	}
}

func TestCLIInstallWizardModeThenComplete(t *testing.T) {
	t.Parallel()
	ws := newInstallSetupWorkspace(t)

	stdout, stderr, code := ws.install(t, nil, "--setup=wizard")
	if code != 0 {
		t.Fatalf("wizard install exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "left to the web wizard") {
		t.Fatalf("wizard hint missing: %q", stdout)
	}
	if v, _ := ws.appSetting(t, "setup.mode"); v != "wizard" {
		t.Fatalf("setup.mode = %q, want wizard", v)
	}
	if _, ok := ws.appSetting(t, "setup.completed_at"); ok {
		t.Fatal("wizard mode must stay SetupPending")
	}

	stdout, stderr, code = ws.install(t, nil)
	if code != 0 {
		t.Fatalf("unset rerun exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "pass --setup=complete") {
		t.Fatalf("remembered wizard choice should be reported: %q", stdout)
	}
	if _, ok := ws.appSetting(t, "setup.completed_at"); ok {
		t.Fatal("remembered wizard choice must keep SetupPending")
	}

	const password = "Env-Secret-42"
	stdout, stderr, code = ws.install(t, []string{initialsetup.PasswordEnv + "=" + password}, "--setup=complete")
	if code != 0 {
		t.Fatalf("complete exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, password) {
		t.Fatal("provided password must never be printed")
	}
	if !strings.Contains(stdout, "(from "+initialsetup.PasswordEnv+")") {
		t.Fatalf("completion should name the password source: %q", stdout)
	}
	if _, ok := ws.appSetting(t, "setup.mode"); ok {
		t.Fatal("--setup=complete must clear the wizard marker")
	}
	if v, ok := ws.appSetting(t, "setup.completed_at"); !ok || strings.TrimSpace(v) == "" {
		t.Fatal("setup.completed_at must be written")
	}
	assertBrowserLoginPassword(t, ws.admin(t).PasswordHash, password)
}
