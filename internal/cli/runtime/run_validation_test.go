// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func assertRunValidationReason(t *testing.T, err *RunValidationError, wantReason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error with reason %q, got nil", wantReason)
	}
	if err.Reason != wantReason {
		t.Fatalf("RunValidationError.Reason = %q, want %q", err.Reason, wantReason)
	}
}

func TestValidateRunModulesPath(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		_, err := ValidateRunModulesPath("   ")
		assertRunValidationReason(t, err, "missing required fields")
	})

	t.Run("invalid value", func(t *testing.T) {
		cases := []string{" /tmp/modules ", "/tmp/modules\nname", "/tmp/one:/tmp/two"}
		for _, tc := range cases {
			_, err := ValidateRunModulesPath(tc)
			assertRunValidationReason(t, err, "invalid value")
		}
	})

	t.Run("path does not exist is auto-created", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "missing")
		gotPath, err := ValidateRunModulesPath(missingPath)
		if err != nil {
			t.Fatalf("ValidateRunModulesPath(%q) error = %#v", missingPath, err)
		}
		if filepath.Clean(gotPath) != filepath.Clean(missingPath) {
			t.Fatalf("ValidateRunModulesPath() = %q, want %q", gotPath, missingPath)
		}
		if st, statErr := os.Stat(missingPath); statErr != nil || !st.IsDir() {
			t.Fatalf("expected directory to be created, statErr=%v st=%#v", statErr, st)
		}
	})

	t.Run("relative path is normalized", func(t *testing.T) {
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

		normalized, runErr := ValidateRunModulesPath("modules")
		if runErr != nil {
			t.Fatalf("ValidateRunModulesPath(relative) error = %#v", runErr)
		}
		if !filepath.IsAbs(normalized) {
			t.Fatalf("ValidateRunModulesPath(relative) = %q, want absolute path", normalized)
		}
	})

	t.Run("path is symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realDir := filepath.Join(tmpDir, "real")
		linkDir := filepath.Join(tmpDir, "link")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatalf("mkdir real dir: %v", err)
		}
		if err := os.Symlink(realDir, linkDir); err != nil {
			t.Skipf("create symlink: %v", err)
		}
		_, err := ValidateRunModulesPath(linkDir)
		assertRunValidationReason(t, err, "invalid value")
	})

	t.Run("path is file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "modules.txt")
		if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		_, err := ValidateRunModulesPath(filePath)
		assertRunValidationReason(t, err, "invalid value")
	})
}

func TestValidateRunDBAndPrepareRunDatabase(t *testing.T) {
	t.Run("invalid dialect", func(t *testing.T) {
		err := ValidateRunDBWithWarning(RunDBOptions{Dialect: "oracle", DSN: "x"}, nil)
		assertRunValidationReason(t, err, "invalid value")
	})

	t.Run("missing dsn", func(t *testing.T) {
		err := ValidateRunDBWithWarning(RunDBOptions{Dialect: "postgres", DSN: "   "}, nil)
		assertRunValidationReason(t, err, "missing required fields")
	})

	t.Run("postgres dsn routed to ValidateRunDatabaseDSN", func(t *testing.T) {
		err := ValidateRunDBWithWarning(RunDBOptions{Dialect: "postgres", DSN: "postgres://127.0.0.1/app"}, nil)
		if err != nil {
			t.Fatalf("ValidateRunDBWithWarning(postgres valid) error = %#v", err)
		}
	})

	t.Run("sqlite dsn routed to ValidateRunSQLite", func(t *testing.T) {
		validPath := filepath.Join(t.TempDir(), "app.db")
		if err := os.WriteFile(validPath, []byte("sqlite"), 0o644); err != nil {
			t.Fatalf("write sqlite file: %v", err)
		}
		dsn := fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", validPath)
		err := ValidateRunDB(RunDBOptions{Dialect: "sqlite", DSN: dsn, AllowCreate: false})
		if err != nil {
			t.Fatalf("ValidateRunDB(sqlite valid) error = %#v", err)
		}
	})

	t.Run("prepare run database no-op for non sqlite", func(t *testing.T) {
		if err := PrepareRunDatabase(RunDBOptions{Dialect: "postgres", DSN: "postgres://127.0.0.1/app", AllowCreate: true}); err != nil {
			t.Fatalf("PrepareRunDatabase(non-sqlite) error = %#v", err)
		}
	})

	t.Run("prepare run database creates sqlite parent", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
		err := PrepareRunDatabase(RunDBOptions{Dialect: "sqlite", DSN: dbPath, AllowCreate: true})
		if err != nil {
			t.Fatalf("PrepareRunDatabase(sqlite) error = %#v", err)
		}
		if st, statErr := os.Stat(filepath.Dir(dbPath)); statErr != nil || !st.IsDir() {
			t.Fatalf("expected sqlite parent directory to exist, statErr=%v st=%#v", statErr, st)
		}
	})

	t.Run("prepare sqlite parse error returns nil", func(t *testing.T) {
		if err := PrepareRunSQLite("file://%zz"); err != nil {
			t.Fatalf("PrepareRunSQLite(parse error) = %#v, want nil", err)
		}
	})

	t.Run("prepare sqlite mkdir failure", func(t *testing.T) {
		base := t.TempDir()
		blocker := filepath.Join(base, "blocked")
		if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
			t.Fatalf("write blocker file: %v", err)
		}
		err := PrepareRunSQLite(filepath.Join(blocker, "app.db"))
		if err == nil {
			t.Fatal("PrepareRunSQLite() expected mkdir failure, got nil")
		}
		if err.Reason != "permission denied or not accessible" {
			t.Fatalf("PrepareRunSQLite() reason = %q, want %q", err.Reason, "permission denied or not accessible")
		}
	})
}

func TestValidateRunDatabaseDSN(t *testing.T) {
	tests := []struct {
		name       string
		dialect    string
		dsn        string
		wantReason string
	}{
		{name: "control chars", dialect: "postgres", dsn: "postgres://db\nname", wantReason: "dsn contains NUL (\\x00) or newline (\\n/\\r)"},
		{name: "empty dsn", dialect: "postgres", dsn: " ", wantReason: "missing required fields"},
		{name: "whitespace", dialect: "postgres", dsn: " postgres://localhost/app ", wantReason: "dsn has leading or trailing whitespace"},
		{name: "scheme conflict", dialect: "mysql", dsn: "postgres://localhost/app", wantReason: "db.dialect conflicts with dsn scheme"},
		{name: "postgresql alias is valid", dialect: "postgres", dsn: "postgresql://localhost/app", wantReason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunDatabaseDSN(tt.dialect, tt.dsn)
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("ValidateRunDatabaseDSN(%q, %q) error = %#v", tt.dialect, tt.dsn, err)
				}
				return
			}
			assertRunValidationReason(t, err, tt.wantReason)
		})
	}
}

func TestValidateRunSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "app.db")
	if err := os.WriteFile(validPath, []byte("sqlite"), 0o644); err != nil {
		t.Fatalf("write sqlite file: %v", err)
	}
	symlinkPath := filepath.Join(tmpDir, "app-link.db")
	if err := os.Symlink(validPath, symlinkPath); err != nil {
		t.Skipf("create sqlite symlink: %v", err)
	}

	tests := []struct {
		name        string
		dsn         string
		allowCreate bool
		wantReason  string
	}{
		{name: "empty dsn", dsn: " ", wantReason: "sqlite dsn must not be empty or whitespace"},
		{name: "memory dsn", dsn: ":memory:", wantReason: "sqlite dsn must not be :memory:"},
		{name: "whitespace path", dsn: " " + validPath + " ", wantReason: "path has leading or trailing whitespace"},
		{name: "relative path", dsn: "relative.db", wantReason: "path is not absolute"},
		{name: "unsupported scheme", dsn: "postgres://localhost/app", wantReason: "sqlite dsn must be a file: URI or plain file path (other URI schemes are not allowed)"},
		{name: "path missing without create", dsn: filepath.Join(tmpDir, "missing.db"), allowCreate: false, wantReason: "path does not exist"},
		{name: "path missing with create but missing pragmas", dsn: filepath.Join(tmpDir, "missing-create.db"), allowCreate: true, wantReason: "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL"},
		{name: "path is directory", dsn: tmpDir, wantReason: "path is a directory"},
		{name: "path is symlink", dsn: symlinkPath, wantReason: "path is a symlink"},
		{name: "file uri missing pragmas", dsn: "file://" + validPath, wantReason: "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL"},
		{name: "file path missing pragmas", dsn: validPath, wantReason: "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL"},
		{name: "valid file uri with pragmas", dsn: fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", validPath), wantReason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunSQLite(tt.dsn, tt.allowCreate, nil)
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("ValidateRunSQLite(%q) error = %#v", tt.dsn, err)
				}
				return
			}
			assertRunValidationReason(t, err, tt.wantReason)
		})
	}
}

func TestValidateRunSQLiteWarningAndHelpers(t *testing.T) {
	t.Run("warn when parent is symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realDir := filepath.Join(tmpDir, "real")
		linkDir := filepath.Join(tmpDir, "link")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatalf("mkdir real dir: %v", err)
		}
		if err := os.Symlink(realDir, linkDir); err != nil {
			t.Skipf("create symlink dir: %v", err)
		}
		dbPath := filepath.Join(linkDir, "app.db")
		if err := os.WriteFile(dbPath, []byte("sqlite"), 0o644); err != nil {
			t.Fatalf("write sqlite file: %v", err)
		}

		var warnings []string
		err := ValidateRunSQLite(
			fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", dbPath),
			false,
			func(msg string) { warnings = append(warnings, msg) },
		)
		if err != nil {
			t.Fatalf("ValidateRunSQLite(with warning) error = %#v", err)
		}
		if len(warnings) == 0 || warnings[0] != "sqlite parent directory is a symlink" {
			t.Fatalf("warnings = %#v, want sqlite parent symlink warning", warnings)
		}
	})

	t.Run("run sqlite path next", func(t *testing.T) {
		if got := RunSQLitePathNext(true); !strings.Contains(got, "absolute and accessible") {
			t.Fatalf("RunSQLitePathNext(true) = %q", got)
		}
		if got := RunSQLitePathNext(false); !strings.Contains(got, "already exists") {
			t.Fatalf("RunSQLitePathNext(false) = %q", got)
		}
	})

	t.Run("sqlite dsn query params parse errors", func(t *testing.T) {
		if _, err := SQLiteDSNQueryParams("/tmp/choysum.sqlite?mode=%zz"); err == nil {
			t.Fatal("expected SQLiteDSNQueryParams to fail for invalid query escape")
		}
	})

	t.Run("validate sqlite pragmas parse errors", func(t *testing.T) {
		err := ValidateRunSQLitePragmas("file://%zz")
		assertRunValidationReason(t, err, "sqlite dsn query params are invalid")
	})

	t.Run("is default sqlite path helper", func(t *testing.T) {
		defaultPath := filepath.Join(t.TempDir(), "choysum.sqlite")
		dsn := defaultPath + "?mode=rwc"
		if !IsDefaultRunSQLitePath(dsn, defaultPath) {
			t.Fatalf("IsDefaultRunSQLitePath(%q, %q) = false, want true", dsn, defaultPath)
		}
		if IsDefaultRunSQLitePath("", defaultPath) {
			t.Fatal("IsDefaultRunSQLitePath(empty dsn) should be false")
		}
		if IsDefaultRunSQLitePath("file://%zz", defaultPath) {
			t.Fatal("IsDefaultRunSQLitePath(invalid dsn) should be false")
		}
	})
}

func TestRuntimeValidationMiscHelpers(t *testing.T) {
	if runtime.GOOS != "windows" {
		if got := HasPathListSeparator("/tmp/one:/tmp/two"); !got {
			t.Fatal("HasPathListSeparator on unix path list should be true")
		}
		if got := HasPathListSeparator("/tmp/one;/tmp/two"); got {
			t.Fatal("HasPathListSeparator on unix semicolon should be false")
		}
	}

	if !ContainsControl("line\nbreak") || !ContainsControl("nul\x00byte") || ContainsControl("plain") {
		t.Fatal("ContainsControl() returned unexpected result")
	}

	if got := URLScheme("postgres://localhost/app"); got != "postgres" {
		t.Fatalf("URLScheme(postgres) = %q, want %q", got, "postgres")
	}
	if got := URLScheme("postgresql://localhost/app"); got != "postgresql" {
		t.Fatalf("URLScheme(postgresql) = %q, want %q", got, "postgresql")
	}
	if got := URLScheme("mysql://localhost/app"); got != "mysql" {
		t.Fatalf("URLScheme(mysql) = %q, want %q", got, "mysql")
	}
	if got := URLScheme("sqlite:///tmp/test.db"); got != "" {
		t.Fatalf("URLScheme(sqlite) = %q, want empty", got)
	}

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("evalsymlinks temp dir: %v", err)
	}
	regularParent := filepath.Join(tmpDir, "regular")
	if err := os.MkdirAll(regularParent, 0o755); err != nil {
		t.Fatalf("mkdir regular parent: %v", err)
	}
	if HasParentSymlink(filepath.Join(regularParent, "app.db")) {
		t.Fatal("HasParentSymlink should be false for regular parent")
	}

	realDir := filepath.Join(tmpDir, "real")
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("create symlink dir: %v", err)
	}
	if !HasParentSymlink(filepath.Join(linkDir, "app.db")) {
		t.Fatal("HasParentSymlink should be true for symlinked parent")
	}
}

func TestSQLitePathFromDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{name: "plain path", dsn: "  /tmp/choysum.sqlite  ", want: "/tmp/choysum.sqlite"},
		{name: "plain path with query", dsn: " /tmp/choysum.sqlite?mode=rwc&_fk=1 ", want: "/tmp/choysum.sqlite"},
		{name: "file uri path", dsn: "file:///tmp/choysum.sqlite?mode=rwc", want: "/tmp/choysum.sqlite"},
		{name: "file uri opaque", dsn: "file:choysum.sqlite?mode=rwc", want: "choysum.sqlite"},
		{name: "unsupported uri scheme", dsn: "postgres://127.0.0.1/app", wantErr: "unsupported sqlite dsn scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SQLitePathFromDSN(tt.dsn)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("SQLitePathFromDSN(%q) error = %v, want substring %q", tt.dsn, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SQLitePathFromDSN(%q) error = %v", tt.dsn, err)
			}
			if got != tt.want {
				t.Fatalf("SQLitePathFromDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestSQLiteDSNQueryParams(t *testing.T) {
	params, err := SQLiteDSNQueryParams("/tmp/choysum.sqlite?mode=rwc&_fk=1")
	if err != nil {
		t.Fatalf("SQLiteDSNQueryParams() error = %v", err)
	}
	if params.Get("mode") != "rwc" || params.Get("_fk") != "1" {
		t.Fatalf("SQLiteDSNQueryParams() = %#v, want mode=rwc and _fk=1", params)
	}
}
