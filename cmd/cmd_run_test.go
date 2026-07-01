// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/server"
	"github.com/choysum-dev/choysum/pkg/server/defaultserver"
	"github.com/spf13/cobra"
)

type runExitPanic struct {
	code int
}

func captureStderrWithPanic(t *testing.T, fn func()) (string, any) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = writer

	var recovered any
	func() {
		defer func() {
			recovered = recover()
			_ = writer.Close()
			os.Stderr = oldStderr
		}()
		fn()
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return string(data), recovered
}

func runtimeValidationToRunError(err *cliruntime.RunValidationError) *runError {
	if err == nil {
		return nil
	}
	code := err.ExitCode
	if code == 0 {
		code = 3
	}
	return &runError{exitCode: code, errMsg: err.ErrMsg, reason: err.Reason, next: err.Next}
}

func TestRunCommandTreatsContextCanceledAsGracefulStop(t *testing.T) {
	tests := []struct {
		name     string
		serveErr error
	}{
		{name: "direct context canceled", serveErr: context.Canceled},
		{name: "wrapped context canceled", serveErr: fmt.Errorf("wrapped: %w", context.Canceled)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, _, _ := writeTempInitializedRunConfig(t, false)
			stub := &runCommandStubServer{serveErr: tt.serveErr}

			originalFactory := runServerFactory
			runServerFactory = func(runtimeScope scope.Scope, _ ...defaultserver.Option) server.Server {
				if runtimeScope == nil {
					t.Fatal("expected non-nil runtime scope")
				}
				return stub
			}
			t.Cleanup(func() {
				runServerFactory = originalFactory
			})

			originalScopeFactory := runRuntimeScopeFactory
			runRuntimeScopeFactory = func(cliruntime.RunScopeInput, *config.LogConfig) (scope.Scope, error) {
				return &commandTestScope{}, nil
			}
			t.Cleanup(func() {
				runRuntimeScopeFactory = originalScopeFactory
			})

			originalExit := runExit
			runExit = func(code int) {
				panic(runExitPanic{code: code})
			}
			t.Cleanup(func() {
				runExit = originalExit
			})

			cmd := newRunCmd()
			cmd.Flags().String("config", "", "")
			if err := cmd.Flags().Set("config", configPath); err != nil {
				t.Fatalf("set config flag: %v", err)
			}

			output := ""
			var recovered any
			func() {
				defer func() {
					recovered = recover()
				}()
				output = captureStderr(t, func() {
					cmd.Run(cmd, []string{"web"})
				})
			}()

			if recovered != nil {
				exitPanic, ok := recovered.(runExitPanic)
				if !ok {
					panic(recovered)
				}
				t.Fatalf("unexpected exit code: %d, stderr=%q", exitPanic.code, output)
			}

			if !stub.serveCalled {
				t.Fatal("expected server Serve to be called")
			}
			if stub.serveCtx == nil {
				t.Fatal("expected non-nil serve context")
			}
			if len(stub.serveServices) != 1 || stub.serveServices[0] != "web" {
				t.Fatalf("unexpected services passed to Serve: %#v", stub.serveServices)
			}
			if strings.Contains(output, "server starting; NEXT: open") {
				t.Fatalf("did not expect CLI startup hint, got %q", output)
			}
			if strings.Contains(output, "ERROR: server exited unexpectedly") {
				t.Fatalf("expected no error block for context cancellation, got %q", output)
			}
		})
	}
}

func TestRunCommandScopeInitErrorOutput(t *testing.T) {
	tests := []struct {
		name             string
		scopeErr         error
		wantErrorTitle   string
		wantReasonSubstr string
		avoidSubstr      string
	}{
		{
			name:             "generic scope init failure",
			scopeErr:         fmt.Errorf("scope factory not registered"),
			wantErrorTitle:   "ERROR: failed to initialize runtime scope",
			wantReasonSubstr: "scope factory not registered",
			avoidSubstr:      "ERROR: cannot connect to database",
		},
		{
			name:             "db-like scope init failure",
			scopeErr:         fmt.Errorf("database connection refused"),
			wantErrorTitle:   "ERROR: cannot connect to database (dialect=",
			wantReasonSubstr: "network unreachable / authentication failed / permission denied / database not found",
			avoidSubstr:      "ERROR: failed to initialize runtime scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, _, _ := writeTempInitializedRunConfig(t, false)

			originalScopeFactory := runRuntimeScopeFactory
			runRuntimeScopeFactory = func(cliruntime.RunScopeInput, *config.LogConfig) (scope.Scope, error) {
				return nil, tt.scopeErr
			}
			t.Cleanup(func() {
				runRuntimeScopeFactory = originalScopeFactory
			})

			serverCalled := false
			originalFactory := runServerFactory
			runServerFactory = func(scope.Scope, ...defaultserver.Option) server.Server {
				serverCalled = true
				return &runCommandStubServer{}
			}
			t.Cleanup(func() {
				runServerFactory = originalFactory
			})

			originalExit := runExit
			runExit = func(code int) {
				panic(runExitPanic{code: code})
			}
			t.Cleanup(func() {
				runExit = originalExit
			})

			cmd := newRunCmd()
			cmd.Flags().String("config", "", "")
			if err := cmd.Flags().Set("config", configPath); err != nil {
				t.Fatalf("set config flag: %v", err)
			}

			output, recovered := captureStderrWithPanic(t, func() {
				cmd.Run(cmd, nil)
			})

			exitPanic, ok := recovered.(runExitPanic)
			if !ok {
				if recovered != nil {
					panic(recovered)
				}
				t.Fatalf("expected runExit panic, got nil")
			}
			if exitPanic.code != 4 {
				t.Fatalf("exit code = %d, want 4", exitPanic.code)
			}
			if serverCalled {
				t.Fatal("server factory should not be called when scope initialization fails")
			}
			if !strings.Contains(output, tt.wantErrorTitle) {
				t.Fatalf("expected stderr to contain %q, got %q", tt.wantErrorTitle, output)
			}
			if !strings.Contains(output, tt.wantReasonSubstr) {
				t.Fatalf("expected stderr to contain %q, got %q", tt.wantReasonSubstr, output)
			}
			if tt.avoidSubstr != "" && strings.Contains(output, tt.avoidSubstr) {
				t.Fatalf("did not expect stderr to contain %q, got %q", tt.avoidSubstr, output)
			}
		})
	}
}

type runCommandStubServer struct {
	serveErr      error
	serveCalled   bool
	serveCtx      context.Context
	serveServices []string
}

func (s *runCommandStubServer) Serve(ctx context.Context, services ...string) error {
	s.serveCalled = true
	s.serveCtx = ctx
	s.serveServices = append([]string{}, services...)
	return s.serveErr
}
func TestSqlitePathFromDsn(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{name: "plain path trimmed", dsn: "  /tmp/test.db  ", want: "/tmp/test.db"},
		{name: "plain path with query", dsn: "  /tmp/test.db?mode=rwc&_fk=1  ", want: "/tmp/test.db"},
		{name: "file scheme path", dsn: "file:///tmp/test.db?mode=ro", want: "/tmp/test.db"},
		{name: "file opaque path", dsn: "file:test.db?cache=shared", want: "test.db"},
		{name: "unsupported scheme", dsn: "postgres://localhost/db", wantErr: "unsupported sqlite dsn scheme"},
		{name: "invalid dsn", dsn: "file://%zz", wantErr: "invalid URL escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cliruntime.SQLitePathFromDSN(tt.dsn)
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
		if got := cliruntime.HasPathListSeparator(tt.path); got != tt.want {
			t.Fatalf("HasPathListSeparator(%q) = %v, want %v", tt.path, got, tt.want)
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
		if got := cliruntime.ContainsControl(tt.value); got != tt.want {
			t.Fatalf("ContainsControl(%q) = %v, want %v", tt.value, got, tt.want)
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
		if got := cliruntime.URLScheme(tt.dsn); got != tt.want {
			t.Fatalf("URLScheme(%q) = %q, want %q", tt.dsn, got, tt.want)
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
			err := runtimeValidationToRunError(cliruntime.ValidateRunDatabaseDSN(tt.dialect, tt.dsn))
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("ValidateRunDatabaseDSN(%q, %q) = %v", tt.dialect, tt.dsn, err)
				}
				return
			}
			if err == nil || err.reason != tt.wantReason {
				t.Fatalf("ValidateRunDatabaseDSN(%q, %q) = %#v, want reason %q", tt.dialect, tt.dsn, err, tt.wantReason)
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
			err := runtimeValidationToRunError(cliruntime.ValidateRunSQLite(tt.dsn, false, nil))
			if tt.wantReason == "" {
				if err != nil {
					t.Fatalf("ValidateRunSQLite(%q) = %v", tt.dsn, err)
				}
				return
			}
			if err == nil || err.reason != tt.wantReason {
				t.Fatalf("ValidateRunSQLite(%q) = %#v, want reason %q", tt.dsn, err, tt.wantReason)
			}
		})
	}
}

func TestValidateRunSqliteAllowsDefaultCreate(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	missingDSN := fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", missingPath)
	if err := runtimeValidationToRunError(cliruntime.ValidateRunSQLite(missingDSN, true, nil)); err != nil {
		t.Fatalf("ValidateRunSQLite(default create) = %#v", err)
	}
}

func TestValidateRunSqliteAllowCreateStillValidatesPragmas(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	missingDSN := fmt.Sprintf("file:%s?mode=rwc", missingPath)

	err := runtimeValidationToRunError(cliruntime.ValidateRunSQLite(missingDSN, true, nil))
	if err == nil || err.reason != "sqlite dsn missing required params: _fk=1, _busy_timeout>0, _journal_mode=WAL" {
		t.Fatalf("ValidateRunSQLite(allowCreate with missing pragmas) = %#v, want missing pragmas error", err)
	}
}

func TestValidateRunSQLitePragmasAndDSNQueryErrors(t *testing.T) {
	t.Run("invalid file uri query parse", func(t *testing.T) {
		err := runtimeValidationToRunError(cliruntime.ValidateRunSQLitePragmas("file://%zz"))
		if err == nil || err.reason != "sqlite dsn query params are invalid" {
			t.Fatalf("ValidateRunSQLitePragmas(file://%%zz) = %#v, want query params invalid error", err)
		}
	})

	t.Run("invalid plain query encoding", func(t *testing.T) {
		if _, err := cliruntime.SQLiteDSNQueryParams("/tmp/choysum.sqlite?mode=%zz"); err == nil {
			t.Fatal("expected SQLiteDSNQueryParams() to fail for invalid query escape")
		}
	})
}

func TestPrepareRunDatabaseCreatesDefaultSqliteParent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state", "choysum.sqlite")
	err := runtimeValidationToRunError(cliruntime.PrepareRunDatabase(runDBRuntimeOptions{Dialect: "sqlite", DSN: dbPath, AllowCreate: true}))
	if err != nil {
		t.Fatalf("PrepareRunDatabase() = %#v", err)
	}
	if info, statErr := os.Stat(filepath.Dir(dbPath)); statErr != nil || !info.IsDir() {
		t.Fatalf("expected sqlite parent dir to exist, stat err=%v info=%#v", statErr, info)
	}
}

func TestValidateRunModulesPath(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		_, e := cliruntime.ValidateRunModulesPath("   ")
		err := runtimeValidationToRunError(e)
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing required fields error, got %#v", err)
		}
	})

	t.Run("empty path returns missing required fields", func(t *testing.T) {
		_, e := cliruntime.ValidateRunModulesPath("")
		err := runtimeValidationToRunError(e)
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
				_, e := cliruntime.ValidateRunModulesPath(tt.path)
				err := runtimeValidationToRunError(e)
				if err == nil || err.reason != "invalid value" {
					t.Fatalf("ValidateRunModulesPath(%q) = %#v, want invalid value", tt.path, err)
				}
			})
		}
	})

	t.Run("path does not exist (auto-created)", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "missing")
		_, e := cliruntime.ValidateRunModulesPath(missingPath)
		err := runtimeValidationToRunError(e)
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
		_, e := cliruntime.ValidateRunModulesPath(missingPath)
		err := runtimeValidationToRunError(e)
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
		_, e := cliruntime.ValidateRunModulesPath(dir)
		err := runtimeValidationToRunError(e)
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
		_, e := cliruntime.ValidateRunModulesPath(linkDir)
		err := runtimeValidationToRunError(e)
		if err == nil || err.reason != "invalid value" {
			t.Fatalf("expected symlink path error, got %#v", err)
		}
	})

	t.Run("path is file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "modules.txt")
		if err := os.WriteFile(filePath, []byte("not a dir"), 0o644); err != nil {
			t.Fatalf("write file path: %v", err)
		}
		_, e := cliruntime.ValidateRunModulesPath(filePath)
		err := runtimeValidationToRunError(e)
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

		normalized, e := cliruntime.ValidateRunModulesPath("modules")
		runErr := runtimeValidationToRunError(e)
		if runErr != nil {
			t.Fatalf("ValidateRunModulesPath(valid relative) = %#v", runErr)
		}
		wantPath, err := filepath.EvalSymlinks(modulesDir)
		if err != nil {
			t.Fatalf("evalsymlinks modules dir: %v", err)
		}
		gotPath, err := filepath.EvalSymlinks(normalized)
		if err != nil {
			t.Fatalf("evalsymlinks cfg modules path: %v", err)
		}
		if !filepath.IsAbs(normalized) || filepath.Clean(gotPath) != filepath.Clean(wantPath) {
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
	if got := cliruntime.HasParentSymlink(filepath.Join(regularParent, "app.db")); got {
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
	if got := cliruntime.HasParentSymlink(filepath.Join(linkDir, "app.db")); !got {
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
		dbOptions := loaded.scopeInput.DBOptions()
		if got := dbOptions.Dialect; got != "sqlite" {
			t.Fatalf("db dialect = %q, want sqlite", got)
		}
		wantDBRoot, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
		wantDBDSN := config.DefaultSQLiteDSN(wantDBRoot)
		if got := dbOptions.DSN; got != wantDBDSN {
			t.Fatalf("db dsn = %q, want %q", got, wantDBDSN)
		}
		if !dbOptions.AllowCreate {
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

	t.Run("missing config file returns file not found error", func(t *testing.T) {
		missingPath := filepath.Join(t.TempDir(), "no-such-config.yaml")
		_, err := loadRunConfig(missingPath)
		if err == nil || err.reason != "file not found or permission denied" {
			t.Fatalf("expected file not found error, got %#v", err)
		}
	})
}

func TestCloneRunLogConfig(t *testing.T) {
	t.Run("nil returns default", func(t *testing.T) {
		got := cliruntime.CloneLogConfig(nil)
		if got == nil {
			t.Fatal("expected non-nil default log config for nil input")
		}
	})

	t.Run("non-nil returns a clone", func(t *testing.T) {
		cfg := &config.LogConfig{Level: "debug"}
		got := cliruntime.CloneLogConfig(cfg)
		if got == cfg {
			t.Fatal("expected a clone, not the same pointer")
		}
		if got.Level != "debug" {
			t.Fatalf("expected Level=debug, got %q", got.Level)
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
		cfgOptions := cliruntime.NewScopeInputConfigOptions(snapshot.New(cfg))
		scopeInput := cliruntime.NewRunScopeInput(
			cfgOptions,
			cliruntime.Options{
				DefaultChoysumPath:    cfgOptions.DefaultChoysumPath,
				ModulesPath:           cfgOptions.ModulesPath,
				TmpPath:               cfgOptions.TmpPath,
				ModuleCatalogIndexURL: strings.TrimSpace(cfgOptions.ModuleCatalogIndexURL),
			},
			cliruntime.NewRunServerOptions(cfg.Server),
			cliruntime.NewRunDBOptions(cfg),
		)
		if err := validateRunConfig(&scopeInput); err != nil {
			t.Fatalf("validateRunConfig(valid) = %#v", err)
		}
	})
}

func TestValidateRunDb(t *testing.T) {
	t.Run("missing dialect", func(t *testing.T) {
		err := runtimeValidationToRunError(cliruntime.ValidateRunDB(runDBRuntimeOptions{DSN: "postgres://localhost/app"}))
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing dialect error, got %#v", err)
		}
	})

	t.Run("invalid dialect", func(t *testing.T) {
		err := runtimeValidationToRunError(cliruntime.ValidateRunDB(runDBRuntimeOptions{Dialect: "oracle", DSN: "oracle://localhost/app"}))
		if err == nil || err.reason != "invalid value" {
			t.Fatalf("expected invalid dialect error, got %#v", err)
		}
	})

	t.Run("missing dsn", func(t *testing.T) {
		err := runtimeValidationToRunError(cliruntime.ValidateRunDB(runDBRuntimeOptions{Dialect: "postgres", DSN: "   "}))
		if err == nil || err.reason != "missing required fields" {
			t.Fatalf("expected missing dsn error, got %#v", err)
		}
	})

	t.Run("postgres valid", func(t *testing.T) {
		if err := runtimeValidationToRunError(cliruntime.ValidateRunDB(runDBRuntimeOptions{Dialect: "postgres", DSN: "postgres://localhost/app"})); err != nil {
			t.Fatalf("ValidateRunDB(valid postgres) = %#v", err)
		}
	})
}
