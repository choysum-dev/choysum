// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/spf13/cobra"
)

func TestSqlitePathFromDsn(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{name: "plain path trimmed", dsn: "  /tmp/test.db  ", want: "/tmp/test.db"},
		{name: "file scheme path", dsn: "file:///tmp/test.db?mode=ro", want: "/tmp/test.db"},
		{name: "file opaque path", dsn: "file:test.db?cache=shared", want: "test.db"},
		{name: "unsupported scheme", dsn: "postgres://localhost/db", wantErr: "unsupported sqlite dsn scheme"},
		{name: "invalid dsn", dsn: "file://%zz", wantErr: "invalid URL escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sqlitePathFromDsn(tt.dsn)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("sqlitePathFromDsn(%q) error = %v, want substring %q", tt.dsn, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("sqlitePathFromDsn(%q) error = %v", tt.dsn, err)
			}
			if got != tt.want {
				t.Fatalf("sqlitePathFromDsn(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestHasPathListSeparator(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix branch only")
	}

	tests := []struct {
		path string
		want bool
	}{
		{path: "/tmp/one:/tmp/two", want: true},
		{path: "/tmp/one;/tmp/two", want: false},
		{path: "/tmp/one", want: false},
	}

	for _, tt := range tests {
		if got := hasPathListSeparator(tt.path); got != tt.want {
			t.Fatalf("hasPathListSeparator(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestContainsControl(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "plain text", want: false},
		{value: "line\nbreak", want: true},
		{value: "carriage\rreturn", want: true},
		{value: "nul\x00byte", want: true},
	}

	for _, tt := range tests {
		if got := containsControl(tt.value); got != tt.want {
			t.Fatalf("containsControl(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestURLScheme(t *testing.T) {
	tests := []struct {
		dsn  string
		want string
	}{
		{dsn: "postgres://localhost/app", want: "postgres"},
		{dsn: "postgresql://localhost/app", want: "postgresql"},
		{dsn: "mysql://localhost/app", want: "mysql"},
		{dsn: "sqlite:///tmp/test.db", want: ""},
		{dsn: "plain-dsn", want: ""},
	}

	for _, tt := range tests {
		if got := urlScheme(tt.dsn); got != tt.want {
			t.Fatalf("urlScheme(%q) = %q, want %q", tt.dsn, got, tt.want)
		}
	}
}

func TestValidateRunDatabaseDsn(t *testing.T) {
	tests := []struct {
		name       string
		dialect    string
		dsn        string
		wantReason string
	}{
		{name: "control chars", dialect: "postgres", dsn: "postgres://db\nname", wantReason: "dsn contains NUL (\\x00) or newline (\\n/\\r)"},
		{name: "whitespace", dialect: "postgres", dsn: " postgres://localhost/app ", wantReason: "dsn has leading or trailing whitespace"},
		{name: "scheme conflict", dialect: "mysql", dsn: "postgres://localhost/app", wantReason: "db.dialect conflicts with dsn scheme"},
		{name: "valid postgres", dialect: "postgres", dsn: "postgresql://localhost/app", wantReason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRunDatabaseDsn(tt.dialect, tt.dsn)
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("validateRunDatabaseDsn(%q, %q) = %v", tt.dialect, tt.dsn, err)
				}
				return
			}
			if err == nil || err.reason != tt.wantReason {
				t.Fatalf("validateRunDatabaseDsn(%q, %q) = %#v, want reason %q", tt.dialect, tt.dsn, err, tt.wantReason)
			}
		})
	}
}

func TestValidateRunSqlite(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "app.db")
	if err := os.WriteFile(validPath, []byte("sqlite"), 0o644); err != nil {
		t.Fatalf("write sqlite file: %v", err)
	}
	symlinkPath := filepath.Join(tmpDir, "app-link.db")
	if err := os.Symlink(validPath, symlinkPath); err != nil {
		t.Fatalf("create sqlite symlink: %v", err)
	}

	tests := []struct {
		name       string
		dsn        string
		wantReason string
	}{
		{name: "empty dsn", dsn: " ", wantReason: "sqlite dsn must not be empty or whitespace"},
		{name: "memory dsn", dsn: ":memory:", wantReason: "sqlite dsn must not be :memory:"},
		{name: "whitespace path", dsn: " " + validPath + " ", wantReason: "path has leading or trailing whitespace"},
		{name: "relative path", dsn: "relative.db", wantReason: "path is not absolute"},
		{name: "unsupported scheme", dsn: "postgres://localhost/app", wantReason: "sqlite dsn must be a file: URI or plain file path (other URI schemes are not allowed)"},
		{name: "path missing", dsn: filepath.Join(tmpDir, "missing.db"), wantReason: "path does not exist"},
		{name: "path is directory", dsn: tmpDir, wantReason: "path is a directory"},
		{name: "path is symlink", dsn: symlinkPath, wantReason: "path is a symlink"},
		{name: "file uri missing sqlite pragmas", dsn: "file://" + validPath, wantReason: "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL"},
		{name: "file path missing sqlite pragmas", dsn: validPath, wantReason: "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL"},
		{name: "valid file uri with sqlite pragmas", dsn: fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", validPath), wantReason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRunSqlite(tt.dsn, false)
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("validateRunSqlite(%q) = %v", tt.dsn, err)
				}
				return
			}
			if err == nil || err.reason != tt.wantReason {
				t.Fatalf("validateRunSqlite(%q) = %#v, want reason %q", tt.dsn, err, tt.wantReason)
			}
		})
	}
}

func TestValidateRunSqliteAllowsDefaultCreate(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	missingDSN := fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", missingPath)
	if err := validateRunSqlite(missingDSN, true); err != nil {
		t.Fatalf("validateRunSqlite(default create) = %#v", err)
	}
}

func TestValidateRunSqliteAllowCreateStillValidatesPragmas(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	missingDSN := fmt.Sprintf("file:%s?mode=rwc", missingPath)

	err := validateRunSqlite(missingDSN, true)
	if err == nil || err.reason != "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL" {
		t.Fatalf("validateRunSqlite(allowCreate with missing pragmas) = %#v, want missing pragmas error", err)
	}
}

func TestValidateRunSQLitePragmasAndDSNQueryErrors(t *testing.T) {
	t.Run("invalid file uri query parse", func(t *testing.T) {
		err := validateRunSQLitePragmas("file://%zz")
		if err == nil || err.reason != "sqlite dsn query params are invalid" {
			t.Fatalf("validateRunSQLitePragmas(file://%%zz) = %#v, want query params invalid error", err)
		}
	})

	t.Run("invalid plain query encoding", func(t *testing.T) {
		if _, err := sqliteDSNQueryParams("/tmp/choysum.sqlite?mode=%zz"); err == nil {
			t.Fatal("expected sqliteDSNQueryParams() to fail for invalid query escape")
		}
	})
}

func TestPrepareRunDatabaseCreatesDefaultSqliteParent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	err := prepareRunDatabase(runDBRuntimeOptions{dialect: "sqlite", dsn: dbPath, allowCreate: true})
	if err != nil {
		t.Fatalf("prepareRunDatabase() = %#v", err)
	}
	if info, statErr := os.Stat(filepath.Dir(dbPath)); statErr != nil || !info.IsDir() {
		t.Fatalf("expected sqlite parent dir to exist, stat err=%v info=%#v", statErr, info)
	}
}

func TestValidateRunModulesPath(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: "   "})
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing required fields error, got %#v", err)
		}
	})

	t.Run("empty path returns missing required fields", func(t *testing.T) {
		err := validateRunModulesPath(&cliRuntimeOptions{})
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing required fields error for empty path, got %#v", err)
		}
	})

	t.Run("invalid value paths", func(t *testing.T) {
		cases := []struct {
			name string
			path string
		}{
			{name: "whitespace", path: " /tmp/modules "},
			{name: "control", path: "/tmp/modules\nname"},
			{name: "path list", path: "/tmp/one:/tmp/two"},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: tt.path})
				if err == nil || err.reason != "invalid value" {
					t.Fatalf("validateRunModulesPath(%q) = %#v, want invalid value", tt.path, err)
				}
			})
		}
	})

	t.Run("path does not exist (auto-created)", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "missing")
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: missingPath})
		if err != nil {
			t.Fatalf("expected auto-created path to succeed, got %#v", err)
		}
		// Verify the directory was actually created.
		if st, statErr := os.Stat(missingPath); statErr != nil || !st.IsDir() {
			t.Fatalf("expected %s to be a directory after auto-creation", missingPath)
		}
	})

	t.Run("mkdir failure due to read-only parent", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "readonly")
		if err := os.MkdirAll(parent, 0o500); err != nil {
			t.Skipf("mkdir parent: %v", err)
		}
		defer func() { _ = os.Chmod(parent, 0o700) }()
		missingPath := filepath.Join(parent, "subdir")
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: missingPath})
		if err == nil || !strings.Contains(err.reason, "cannot be created") {
			t.Fatalf("expected mkdir failure, got %#v", err)
		}
	})

	t.Run("unreadable directory after creation", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "readfail")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Chmod(dir, 0o000); err != nil {
			t.Skipf("chmod: %v", err)
		}
		defer func() { _ = os.Chmod(dir, 0o755) }()
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: dir})
		if err == nil || err.reason != "permission denied or not accessible" {
			t.Fatalf("expected permission error, got %#v", err)
		}
	})

	t.Run("path is symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realDir := filepath.Join(tmpDir, "real-modules")
		linkDir := filepath.Join(tmpDir, "modules-link")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatalf("mkdir real dir: %v", err)
		}
		if err := os.Symlink(realDir, linkDir); err != nil {
			t.Fatalf("create symlink: %v", err)
		}
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: linkDir})
		if err == nil || err.reason != "invalid value" {
			t.Fatalf("expected symlink path error, got %#v", err)
		}
	})

	t.Run("path is file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "modules.txt")
		if err := os.WriteFile(filePath, []byte("not a dir"), 0o644); err != nil {
			t.Fatalf("write file path: %v", err)
		}
		err := validateRunModulesPath(&cliRuntimeOptions{modulesPath: filePath})
		if err == nil || err.reason != "invalid value" {
			t.Fatalf("expected non-directory error, got %#v", err)
		}
	})

	t.Run("valid relative path is normalized", func(t *testing.T) {
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		workDir := t.TempDir()
		modulesDir := filepath.Join(workDir, "modules")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
		if err := os.Chdir(workDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		runtimeOptions := &cliRuntimeOptions{modulesPath: "modules"}
		if err := validateRunModulesPath(runtimeOptions); err != nil {
			t.Fatalf("validateRunModulesPath(valid relative) = %#v", err)
		}
		wantPath, err := filepath.EvalSymlinks(modulesDir)
		if err != nil {
			t.Fatalf("evalsymlinks modules dir: %v", err)
		}
		gotPath, err := filepath.EvalSymlinks(runtimeOptions.modulesPath)
		if err != nil {
			t.Fatalf("evalsymlinks cfg modules path: %v", err)
		}
		if !filepath.IsAbs(runtimeOptions.modulesPath) || filepath.Clean(gotPath) != filepath.Clean(wantPath) {
			t.Fatalf("expected relative modules path to normalize to %q, got %q", wantPath, gotPath)
		}
	})
}

func TestHasParentSymlink(t *testing.T) {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("evalsymlinks temp dir: %v", err)
	}
	regularParent := filepath.Join(tmpDir, "regular")
	if err := os.MkdirAll(regularParent, 0o755); err != nil {
		t.Fatalf("mkdir regular parent: %v", err)
	}
	if got := hasParentSymlink(filepath.Join(regularParent, "app.db")); got {
		t.Fatal("expected regular parent directories to have no symlink")
	}

	realDir := filepath.Join(tmpDir, "real")
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}
	if got := hasParentSymlink(filepath.Join(linkDir, "app.db")); !got {
		t.Fatal("expected symlinked parent directory to be detected")
	}
}

func TestResolveRunConfigPath(t *testing.T) {
	newCommand := func(t *testing.T, configValue string) *cobra.Command {
		t.Helper()
		cmd := &cobra.Command{Use: "run"}
		cmd.Flags().String("config", configValue, "")
		if err := cmd.Flags().Set("config", configValue); err != nil {
			t.Fatalf("set config flag: %v", err)
		}
		return cmd
	}

	t.Run("control characters", func(t *testing.T) {
		_, err := resolveRunConfigPath(newCommand(t, "bad\npath.yaml"))
		if err == nil || err.reason != "path contains NUL (\\x00) or newline (\\n/\\r)" {
			t.Fatalf("expected control-char error, got %#v", err)
		}
	})

	t.Run("leading trailing whitespace", func(t *testing.T) {
		_, err := resolveRunConfigPath(newCommand(t, " config.yaml "))
		if err == nil || err.reason != "path has leading or trailing whitespace" {
			t.Fatalf("expected whitespace error, got %#v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "missing.yaml")
		_, err := resolveRunConfigPath(newCommand(t, missingPath))
		if err == nil || err.reason != "file not found" {
			t.Fatalf("expected file-not-found error, got %#v", err)
		}
	})

	t.Run("missing default config falls back to built-in defaults", func(t *testing.T) {
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		workDir := t.TempDir()
		if err := os.Chdir(workDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		got, runErr := resolveRunConfigPath(newCommand(t, ""))
		if runErr != nil {
			t.Fatalf("resolveRunConfigPath(default fallback) = %#v", runErr)
		}
		if got != "" {
			t.Fatalf("expected empty config path for built-in defaults, got %q", got)
		}
	})

	t.Run("path is symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realPath := filepath.Join(tmpDir, "real.yaml")
		linkPath := filepath.Join(tmpDir, "link.yaml")
		if err := os.WriteFile(realPath, []byte("db:\n  dialect: sqlite\n"), 0o644); err != nil {
			t.Fatalf("write real config: %v", err)
		}
		if err := os.Symlink(realPath, linkPath); err != nil {
			t.Fatalf("create config symlink: %v", err)
		}
		_, err := resolveRunConfigPath(newCommand(t, linkPath))
		if err == nil || err.reason != "config path is a symlink" {
			t.Fatalf("expected symlink error, got %#v", err)
		}
	})

	t.Run("path is directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := resolveRunConfigPath(newCommand(t, dir))
		if err == nil || err.reason != "config path is a directory" {
			t.Fatalf("expected directory error, got %#v", err)
		}
	})

	t.Run("relative path is normalized", func(t *testing.T) {
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		workDir := t.TempDir()
		cfgPath := filepath.Join(workDir, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte("db:\n  dialect: sqlite\n"), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		if err := os.Chdir(workDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		got, runErr := resolveRunConfigPath(newCommand(t, "config.yaml"))
		if runErr != nil {
			t.Fatalf("resolveRunConfigPath(relative) = %#v", runErr)
		}
		wantPath, err := filepath.EvalSymlinks(cfgPath)
		if err != nil {
			t.Fatalf("evalsymlinks config path: %v", err)
		}
		gotPath, err := filepath.EvalSymlinks(got)
		if err != nil {
			t.Fatalf("evalsymlinks resolved config path: %v", err)
		}
		if filepath.Clean(gotPath) != filepath.Clean(wantPath) {
			t.Fatalf("expected normalized config path %q, got %q", wantPath, gotPath)
		}
	})
}

func TestLoadRunConfig(t *testing.T) {
	writeConfig := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		return path
	}

	t.Run("missing default config uses built-in defaults", func(t *testing.T) {
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		workDir := t.TempDir()
		homeDir := t.TempDir()
		modulesDir := filepath.Join(workDir, "modules")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
		if err := os.Chdir(workDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		t.Setenv("HOME", homeDir)
		t.Setenv("CHOYSUM_DEFAULT_CHOYSUM_PATH", "")
		t.Setenv("CHOYSUM_DB_DIALECT", "")
		t.Setenv("CHOYSUM_DB_DSN", "")
		t.Setenv("CHOYSUM_AUTH_INTERNAL_KEY", "dev-internal-key")

		loaded, runErr := loadRunConfig("")
		if runErr != nil {
			t.Fatalf("loadRunConfig(default fallback) = %#v", runErr)
		}
		wantDefaultChoysumPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
		if got := loaded.scopeInput.DefaultChoysumPath(); filepath.Clean(got) != filepath.Clean(wantDefaultChoysumPath) {
			t.Fatalf("default choysum path = %q, want %q", got, wantDefaultChoysumPath)
		}
		gotModulesPath, err := filepath.EvalSymlinks(loaded.scopeInput.ModulesPath())
		if err != nil {
			t.Fatalf("evalsymlinks modules path: %v", err)
		}
		wantModulesPath, err := filepath.EvalSymlinks(modulesDir)
		if err != nil {
			t.Fatalf("evalsymlinks want modules path: %v", err)
		}
		if filepath.Clean(gotModulesPath) != filepath.Clean(wantModulesPath) {
			t.Fatalf("modules path = %q, want %q", gotModulesPath, wantModulesPath)
		}
		if got := loaded.scopeInput.dbOptions.dialect; got != "sqlite" {
			t.Fatalf("db dialect = %q, want sqlite", got)
		}
		wantDBRoot, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
		wantDBDSN := config.DefaultSQLiteDSN(wantDBRoot)
		if got := loaded.scopeInput.dbOptions.dsn; got != wantDBDSN {
			t.Fatalf("db dsn = %q, want %q", got, wantDBDSN)
		}
		if !loaded.scopeInput.dbOptions.allowCreate {
			t.Fatal("expected default sqlite path to allow first-run creation")
		}
	})

	t.Run("defaults modules path when key missing", func(t *testing.T) {
		cfgPath := writeConfig(t, `
default_choysum_path: ./.choysum
db:
  dialect: sqlite
  dsn: /tmp/app.db
`)
		loaded, err := loadRunConfig(cfgPath)
		if err != nil {
			t.Fatalf("loadRunConfig(valid) = %#v", err)
		}
		wantAbs := filepath.Join(filepath.Dir(cfgPath), ".choysum", "modules")
		if got := loaded.scopeInput.ModulesPath(); filepath.Clean(got) != filepath.Clean(wantAbs) {
			t.Fatalf("expected modules path fallback to %q, got %q", wantAbs, got)
		}
	})

	t.Run("invalid yaml format", func(t *testing.T) {
		cfgPath := writeConfig(t, `modules_path: [`)
		_, err := loadRunConfig(cfgPath)
		if err == nil || err.reason != "invalid config format (YAML parse failed)" {
			t.Fatalf("expected YAML parse error mapping, got %#v", err)
		}
	})

	t.Run("invalid config values", func(t *testing.T) {
		cfgPath := writeConfig(t, `
default_choysum_path: ./.choysum
compile:
  bundleMode: invalid
db:
  dialect: sqlite
  dsn: /tmp/app.db
`)
		_, err := loadRunConfig(cfgPath)
		if err == nil || err.reason != "invalid config values" {
			t.Fatalf("expected invalid config values mapping, got %#v", err)
		}
	})
}

func TestValidateRunConfig(t *testing.T) {
	t.Run("missing db", func(t *testing.T) {
		err := validateRunConfig(nil)
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing required fields error, got %#v", err)
		}
	})

	t.Run("valid config", func(t *testing.T) {
		modulesDir := filepath.Join(t.TempDir(), "modules")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
		dbPath := filepath.Join(t.TempDir(), "app.db")
		if err := os.WriteFile(dbPath, []byte("sqlite"), 0o644); err != nil {
			t.Fatalf("write sqlite file: %v", err)
		}
		dbDSN := fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", dbPath)

		cfg := &config.Config{
			ModulesPath: modulesDir,
			Db: &config.DbConfig{
				Dialect: "sqlite",
				DSN:     dbDSN,
			},
		}
		scopeInput := newRunRuntimeScopeInput(
			newScopeInputConfigOptions(snapshot.New(cfg)),
			newCliRuntimeOptionsFromScopeInputOptions(newScopeInputConfigOptions(snapshot.New(cfg))),
			newRunServerRuntimeOptions(cfg.Server),
			newRunDBRuntimeOptions(cfg),
		)
		if err := validateRunConfig(&scopeInput); err != nil {
			t.Fatalf("validateRunConfig(valid) = %#v", err)
		}
	})
}

func TestValidateRunDb(t *testing.T) {
	t.Run("missing dialect", func(t *testing.T) {
		err := validateRunDb(runDBRuntimeOptions{dsn: "postgres://localhost/app"})
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing dialect error, got %#v", err)
		}
	})

	t.Run("invalid dialect", func(t *testing.T) {
		err := validateRunDb(runDBRuntimeOptions{dialect: "oracle", dsn: "oracle://localhost/app"})
		if err == nil || err.reason != "invalid value" {
			t.Fatalf("expected invalid dialect error, got %#v", err)
		}
	})

	t.Run("missing dsn", func(t *testing.T) {
		err := validateRunDb(runDBRuntimeOptions{dialect: "postgres", dsn: "   "})
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing dsn error, got %#v", err)
		}
	})

	t.Run("postgres valid", func(t *testing.T) {
		if err := validateRunDb(runDBRuntimeOptions{dialect: "postgres", dsn: "postgres://localhost/app"}); err != nil {
			t.Fatalf("validateRunDb(valid postgres) = %#v", err)
		}
	})
}
