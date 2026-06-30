// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	pkge2e "github.com/choysum-dev/choysum/internal/testing/e2e"
	pkgrunner "github.com/choysum-dev/choysum/internal/testing/runner"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

const commandConstructorHelperEnv = "CHOYSUM_CMD_CONSTRUCTOR_HELPER"

type commandTestScope struct {
	ctx context.Context
	cfg *config.Config
}

type commandExitScope struct {
	commandTestScope
}

type commandRunErrScope struct {
	commandExitScope
	runErr error
}

type commandBufferLoggerScope struct {
	*commandTestScope
	logger *slog.Logger
}

type commandErrorTransactor struct{ err error }

func (e *commandTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *commandTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *commandTestScope) Session() *scope.Session { return nil }
func (e *commandTestScope) WithContext(ctx context.Context) scope.Scope {
	return &commandTestScope{ctx: ctx, cfg: e.cfg}
}
func (e *commandTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *commandTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *commandTestScope) FactoryInput() scope.FactoryInput {
	if e == nil || e.cfg == nil {
		return nil
	}
	options := newScopeInputConfigOptions(snapshot.New(e.cfg))
	runtimeOptions := newCliRuntimeOptionsFromScopeInputOptions(options)
	return newCommandRuntimeScopeInput(options, runtimeOptions)
}

func (e *commandExitScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func (e *commandRunErrScope) Run(fn func(runtimeScope scope.Scope) error) error {
	if e.runErr != nil {
		return e.runErr
	}
	return fn(e)
}

func (e *commandRunErrScope) Transactor() scope.Transactor {
	return commandErrorTransactor{err: e.runErr}
}

func (e *commandRunErrScope) WithContext(ctx context.Context) scope.Scope {
	return &commandRunErrScope{
		commandExitScope: commandExitScope{commandTestScope: commandTestScope{ctx: ctx, cfg: e.cfg}},
		runErr:           e.runErr,
	}
}

func (e *commandBufferLoggerScope) Logger() *slog.Logger {
	if e != nil && e.logger != nil {
		return e.logger
	}
	if e != nil && e.commandTestScope != nil {
		return e.commandTestScope.Logger()
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *commandBufferLoggerScope) WithContext(ctx context.Context) scope.Scope {
	if e == nil {
		return &commandBufferLoggerScope{commandTestScope: &commandTestScope{ctx: ctx}}
	}
	if e.commandTestScope == nil {
		return &commandBufferLoggerScope{commandTestScope: &commandTestScope{ctx: ctx}, logger: e.logger}
	}
	return &commandBufferLoggerScope{
		commandTestScope: &commandTestScope{ctx: ctx, cfg: e.cfg},
		logger:           e.logger,
	}
}

func (t commandErrorTransactor) Do(context.Context, scope.TransactionOptions, scope.TxFunc) error {
	return t.err
}

func (t commandErrorTransactor) Required(context.Context, scope.TxFunc) error {
	return t.err
}

func (t commandErrorTransactor) RequiresNew(context.Context, scope.TxFunc) error {
	return t.err
}

func (t commandErrorTransactor) Nested(context.Context, scope.TxFunc) error {
	return t.err
}

type commandHelperEngine struct{}

func (commandHelperEngine) Load([]*jsengine.JsScript) error { return nil }
func (commandHelperEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: "helper"}, nil
}
func (commandHelperEngine) Close() error { return nil }

type commandFailLoadEngine struct{}

func (commandFailLoadEngine) Load([]*jsengine.JsScript) error {
	return errors.New("engine init failed")
}
func (commandFailLoadEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (commandFailLoadEngine) Close() error { return nil }

func registerCommandHelperEngines() {
	jsengine.Register("cmd-helper-engine", func(jsengine.ScopeProvider, auth.Authenticator, ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
		return func() (jsengine.JsEngine, error) { return commandHelperEngine{}, nil }
	})
	jsengine.Register("cmd-helper-fail-start", func(jsengine.ScopeProvider, auth.Authenticator, ...jsengine.JsEngineOption) jsengine.JsEngineFactory {
		return func() (jsengine.JsEngine, error) { return commandFailLoadEngine{}, nil }
	})
}

func TestRequireCliRuntimeOptions(t *testing.T) {
	if _, err := requireCliRuntimeOptions(nil); err == nil || !strings.Contains(err.Error(), "getter is not initialized") {
		t.Fatalf("expected nil getter error, got %v", err)
	}

	if _, err := requireCliRuntimeOptions(func() cliRuntimeOptions { return cliRuntimeOptions{} }); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("expected Validate() error from getter options, got %v", err)
	}

	want := cliRuntimeOptions{
		defaultChoysumPath: "/workspace/.choysum",
		modulesPath:        "/workspace/modules",
		tmpPath:            "/workspace/.choysum/tmp",
	}
	got, err := requireCliRuntimeOptions(func() cliRuntimeOptions { return want })
	if err != nil {
		t.Fatalf("requireCliRuntimeOptions(valid) error = %v", err)
	}
	if got != want {
		t.Fatalf("requireCliRuntimeOptions(valid) = %#v, want %#v", got, want)
	}
}

func newCommandExitConfig(jsEngineFactory string) *config.Config {
	serverCfg := config.NewDefaultServerConfig()
	serverCfg.JsEngineFactory = jsEngineFactory
	return &config.Config{Server: serverCfg, Log: config.NewDefaultLogConfig()}
}

func newCommandTestConfig(modulesPath string) *config.Config {
	defaultChoysumPath := filepath.Join(modulesPath, ".choysum")
	return &config.Config{
		ModulesPath:        modulesPath,
		DistPath:           filepath.Join(modulesPath, "dist"),
		DefaultChoysumPath: defaultChoysumPath,
		TmpPath:            filepath.Join(defaultChoysumPath, "tmp"),
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(modulesPath, "command-test.db"),
		},
		Server: config.NewDefaultServerConfig(),
		Log:    config.NewDefaultLogConfig(),
	}
}

func commandRuntimeOptionsFromScope(scopeGetter func() scope.Scope) func() cliRuntimeOptions {
	return func() cliRuntimeOptions {
		runtimeScope := scopeGetter()
		if runtimeScope == nil {
			return newCliRuntimeOptions(scope.PathsRuntimeOptions{}, false)
		}
		pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
		return newCliRuntimeOptions(pathOpts, hasPathOpts)
	}
}

func newTestUnitCmdFromScope(scopeGetter func() scope.Scope) *cobra.Command {
	return newTestUnitCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
}

func writeCommandPackage(t *testing.T, modulesPath string, moduleName string, packageJSON string) {
	t.Helper()
	moduleDir := filepath.Join(modulesPath, moduleName)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func TestCommandConstructorExitHelper(t *testing.T) {
	if os.Getenv(commandConstructorHelperEnv) != "1" {
		return
	}
	scenario := os.Getenv("CHOYSUM_CMD_CONSTRUCTOR_SCENARIO")
	registerCommandHelperEngines()
	exitScope := func() scope.Scope { return &commandExitScope{} }

	switch scenario {
	case "install_prerun_nil_env":
		cmd := newInstallCmd(func() scope.Scope { return nil })
		cmd.PreRun(cmd, []string{"base"})
	case "install_prerun_missing_args":
		cmd := newInstallCmd(exitScope)
		cmd.PreRun(cmd, nil)
	case "install_run_nil_env":
		cmd := newInstallCmd(func() scope.Scope { return nil })
		cmd.Run(cmd, []string{"base"})
	case "install_run_executor_create_error":
		cmd := newInstallCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("missing-cmd-engine")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "install_run_executor_start_error":
		cmd := newInstallCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-fail-start")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "install_run_env_run_error":
		cmd := newInstallCmd(func() scope.Scope {
			return &commandRunErrScope{
				commandExitScope: commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-engine")}},
				runErr:           errors.New("tx failed"),
			}
		})
		cmd.Run(cmd, []string{"base"})
	case "upgrade_prerun_nil_env":
		cmd := newUpgradeCmd(func() scope.Scope { return nil })
		cmd.PreRun(cmd, []string{"base"})
	case "upgrade_prerun_missing_args":
		cmd := newUpgradeCmd(exitScope)
		cmd.PreRun(cmd, nil)
	case "upgrade_run_nil_env":
		cmd := newUpgradeCmd(func() scope.Scope { return nil })
		cmd.Run(cmd, []string{"base"})
	case "upgrade_run_executor_create_error":
		cmd := newUpgradeCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("missing-cmd-engine")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "upgrade_run_executor_start_error":
		cmd := newUpgradeCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-fail-start")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "upgrade_run_env_run_error":
		cmd := newUpgradeCmd(func() scope.Scope {
			return &commandRunErrScope{
				commandExitScope: commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-engine")}},
				runErr:           errors.New("upgrade failed"),
			}
		})
		cmd.Run(cmd, []string{"base"})
	case "uninstall_prerun_nil_env":
		cmd := newUninstallCmd(func() scope.Scope { return nil })
		cmd.PreRun(cmd, []string{"base"})
	case "uninstall_prerun_missing_args":
		cmd := newUninstallCmd(exitScope)
		cmd.PreRun(cmd, nil)
	case "uninstall_run_nil_env":
		cmd := newUninstallCmd(func() scope.Scope { return nil })
		cmd.Run(cmd, []string{"base"})
	case "uninstall_run_executor_create_error":
		cmd := newUninstallCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("missing-cmd-engine")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "uninstall_run_executor_start_error":
		cmd := newUninstallCmd(func() scope.Scope {
			return &commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-fail-start")}}
		})
		cmd.Run(cmd, []string{"base"})
	case "uninstall_run_env_run_error":
		cmd := newUninstallCmd(func() scope.Scope {
			return &commandRunErrScope{
				commandExitScope: commandExitScope{commandTestScope: commandTestScope{cfg: newCommandExitConfig("cmd-helper-engine")}},
				runErr:           errors.New("uninstall failed"),
			}
		})
		cmd.Run(cmd, []string{"base"})
	default:
		os.Exit(2)
	}

	os.Exit(0)
}

func TestNewTypeFetchCmdHidden(t *testing.T) {
	cmd := newTypeFetchCmd(func() scope.Scope { return nil })
	if !cmd.Hidden {
		t.Fatalf("expected type-fetch command to be hidden")
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleCachedSummary(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "app", `{"dependencies":{"dep":"1.0.0"}}`)

	typesDir := filepath.Join(cfg.DefaultChoysumPath, "pkg", "types")
	cacheFile := filepath.Join(typesDir, "dep@1.0.0.d.ts")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("export declare const x: number;"), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"app", "--offline"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"Ensured tsconfig exists:",
		"[app] completed: direct targets=1 (cached=1, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)",
		"Type fetch complete: direct targets=1 (cached=1, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0).",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
	if strings.Contains(output, "[app] cached dep@1.0.0") || strings.Contains(output, "[app] fetched dep@1.0.0") {
		t.Fatalf("expected per-package cached/fetched lines to be suppressed, got %q", output)
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleNoDependenciesUsesNeutralSummary(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "empty", `{}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"empty", "--offline"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "[empty] completed: direct targets=0 (cached=0, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)") {
		t.Fatalf("expected neutral zero-result summary, got %q", output)
	}
	if strings.Contains(output, "no dependencies found") {
		t.Fatalf("unexpected no-dependencies phrasing, got %q", output)
	}
}

func TestNewTypeFetchCmd_Run_OfflineSingleAppReturnsError(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "broken", `{"dependencies":`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"broken", "--offline"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected type-fetch to return error for invalid package.json")
	}
	if !strings.Contains(err.Error(), "parse package.json") {
		t.Fatalf("expected parse package.json error, got %v", err)
	}
	if !strings.Contains(out.String(), "resolve module depends closure") {
		t.Fatalf("expected command output to include closure resolution error, got %q", out.String())
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleIncludesDependsClosureByDefault(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)
	writeCommandPackage(t, modulesPath, "base", `{"dependencies":{"dep":"1.0.0"}}`)

	typesDir := filepath.Join(cfg.DefaultChoysumPath, "pkg", "types")
	cacheFile := filepath.Join(typesDir, "dep@1.0.0.d.ts")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("export declare const x: number;"), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth", "--offline"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"[auth] completed: direct targets=0 (cached=0, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)",
		"[base] completed: direct targets=1 (cached=1, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)",
		"Type fetch complete: direct targets=1 (cached=1, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0).",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleMissingDependsDefaultError(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	cmd.SetArgs([]string{"auth", "--offline"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing depends modules error")
	}
	if !strings.Contains(err.Error(), "missing depends modules") || !strings.Contains(err.Error(), "base") {
		t.Fatalf("expected missing depends modules error for base, got %v", err)
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleMissingDependsWarn(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth", "--offline", "--missing-dep-policy", "warn"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Warning: missing depends modules (skipped): base") {
		t.Fatalf("expected warning for missing depends module, got %q", output)
	}
	if !strings.Contains(output, "[auth] completed: direct targets=0 (cached=0, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)") {
		t.Fatalf("expected auth summary line, got %q", output)
	}
}

func TestNewTypeFetchCmd_Run_SingleModuleWithDependsDisabledSkipsClosure(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth", "--offline", "--with-depends=false"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "missing depends modules") {
		t.Fatalf("expected no missing depends warning, got %q", output)
	}
	if !strings.Contains(output, "[auth] completed: direct targets=0 (cached=0, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)") {
		t.Fatalf("expected auth summary line, got %q", output)
	}
}

func TestNewTypeFetchCmd_Run_InvalidMissingDepPolicy(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	cmd.SetArgs([]string{"auth", "--offline", "--missing-dep-policy", "skip"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid missing-dep-policy error")
	}
	if !strings.Contains(err.Error(), "invalid --missing-dep-policy") {
		t.Fatalf("expected invalid policy error, got %v", err)
	}
}

func TestResolveTypeFetchDependsClosure_RejectsTraversalDependsPath(t *testing.T) {
	modulesPath := t.TempDir()
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["../escape"]}}`)

	_, _, err := resolveTypeFetchDependsClosure(modulesPath, "auth")
	if err == nil {
		t.Fatal("expected traversal depends module path to be rejected")
	}
	if !strings.Contains(err.Error(), `invalid module path "../escape"`) {
		t.Fatalf("expected invalid module path error, got %v", err)
	}
}

func TestTypeFetchModulePackagePath(t *testing.T) {
	modulesPath := t.TempDir()

	t.Run("resolves module package path inside modules root", func(t *testing.T) {
		got, err := typeFetchModulePackagePath(modulesPath, "auth")
		if err != nil {
			t.Fatalf("typeFetchModulePackagePath error: %v", err)
		}
		if !strings.HasSuffix(filepath.ToSlash(got), "/auth/package.json") {
			t.Fatalf("unexpected package path: %s", got)
		}
	})

	t.Run("rejects empty modules path", func(t *testing.T) {
		_, err := typeFetchModulePackagePath("", "auth")
		if err == nil || !strings.Contains(err.Error(), "modules path is required") {
			t.Fatalf("expected modules path required error, got %v", err)
		}
	})

	t.Run("rejects empty module name", func(t *testing.T) {
		_, err := typeFetchModulePackagePath(modulesPath, "")
		if err == nil || !strings.Contains(err.Error(), "module name is required") {
			t.Fatalf("expected module name required error, got %v", err)
		}
	})

	t.Run("rejects traversal module path", func(t *testing.T) {
		_, err := typeFetchModulePackagePath(modulesPath, "../escape")
		if err == nil || !strings.Contains(err.Error(), `invalid module path "../escape"`) {
			t.Fatalf("expected invalid module path error, got %v", err)
		}
	})
}

func TestReadTypeFetchModulePackage_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, []byte("\n  \n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	pkg, err := readTypeFetchModulePackage(pkgPath)
	if err != nil {
		t.Fatalf("readTypeFetchModulePackage failed: %v", err)
	}
	if len(pkg.Choysum.Depends) != 0 {
		t.Fatalf("expected no depends for empty package json, got %+v", pkg.Choysum.Depends)
	}
}

func TestValidateTypeFetchDependsCompleteness_RejectsTraversalDependsPath(t *testing.T) {
	modulesPath := t.TempDir()
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["../escape"]}}`)

	_, err := validateTypeFetchDependsCompleteness(modulesPath, []string{"auth"})
	if err == nil {
		t.Fatal("expected traversal depends module path to be rejected")
	}
	if !strings.Contains(err.Error(), `invalid module path "../escape"`) {
		t.Fatalf("expected invalid module path error, got %v", err)
	}
}

func TestNewTypeFetchCmd_Run_AllModulesMissingDependsDefaultWarn(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--all", "--offline"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("type-fetch execute error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Warning: missing depends modules (skipped): base") {
		t.Fatalf("expected warning for missing depends module, got %q", output)
	}
	if !strings.Contains(output, "[auth] completed: direct targets=0 (cached=0, fetched=0, reused=0, failed=0), transitive (cached=0, fetched=0)") {
		t.Fatalf("expected auth summary line, got %q", output)
	}
}

func TestNewTypeFetchCmd_Run_AllModulesMissingDependsError(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "auth", `{"choysum":{"depends":["base"]}}`)

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	cmd.SetArgs([]string{"--all", "--offline", "--missing-dep-policy", "error"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing depends modules error")
	}
	if !strings.Contains(err.Error(), "missing depends modules") || !strings.Contains(err.Error(), "base") {
		t.Fatalf("expected missing depends modules error for base, got %v", err)
	}
}

func TestNewTypeFetchCmd_Run_ContextCanceledReturnsContextError(t *testing.T) {
	modulesPath := t.TempDir()
	cfg := newCommandTestConfig(modulesPath)
	writeCommandPackage(t, modulesPath, "app", `{"dependencies":{"dep":"1.0.0"}}`)

	requestStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestStarted <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	cmd := newTypeFetchCmd(func() scope.Scope { return &commandTestScope{cfg: cfg} })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"app", "--upstream", server.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)
	go func() {
		select {
		case <-requestStarted:
			cancel()
		case <-time.After(2 * time.Second):
			cancel()
		}
	}()

	err := cmd.Execute()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	output := out.String()
	if strings.Contains(output, "[app] error: context canceled") || strings.Contains(output, "[app] error: context cancelled") {
		t.Fatalf("unexpected generic cancellation output, got %q", output)
	}
}

func runCommandConstructorExit(t *testing.T, scenario string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCommandConstructorExitHelper")
	cmd.Env = append(os.Environ(),
		commandConstructorHelperEnv+"=1",
		"CHOYSUM_CMD_CONSTRUCTOR_SCENARIO="+scenario,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(output), exitErr.ExitCode()
	}
	return string(output), 1
}

func TestNewTestCmd_NamespaceAndUsageError(t *testing.T) {
	scopeGetter := func() scope.Scope { return nil }
	cmd := newTestCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
	if cmd.Use != "test" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	seenSubcommands := map[string]bool{"unit": false, "typecheck": false, "e2e": false}
	for _, sub := range cmd.Commands() {
		if _, ok := seenSubcommands[sub.Name()]; ok {
			seenSubcommands[sub.Name()] = true
		}
	}
	for name, seen := range seenSubcommands {
		if !seen {
			t.Fatalf("expected test subcommand %q to be registered", name)
		}
	}
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires a subcommand") {
		t.Fatalf("expected missing subcommand error, got %v", err)
	}
}

func TestNewTestUnitCmd_ArgsAndEarlyRunE(t *testing.T) {
	cmd := newTestUnitCmdFromScope(func() scope.Scope { return nil })
	if cmd.Use != "unit <app>" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.Flags().Lookup("all") == nil || cmd.Flags().Lookup("coverage") == nil || cmd.Flags().Lookup("timeout") == nil || cmd.Flags().Lookup("keep") == nil || cmd.Flags().Lookup("runtime-log-level") == nil {
		t.Fatal("expected test command flags to be registered")
	}

	if err := cmd.Args(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires exactly 1 app argument") {
		t.Fatalf("expected missing app argument error, got %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set --all: %v", err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("expected --all with no args to pass, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--all cannot be used with an app argument") {
		t.Fatalf("expected --all arg conflict, got %v", err)
	}

	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected nil scope error, got %v", err)
	}

	cmd = newTestUnitCmdFromScope(func() scope.Scope { return &commandTestScope{} })
	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "test unit: invalid runtime options") {
		t.Fatalf("expected invalid runtime options error, got %v", err)
	}

	cmd = newTestUnitCmdFromScope(func() scope.Scope {
		return &commandTestScope{cfg: &config.Config{ModulesPath: "   ", TmpPath: "/tmp/choysum", DefaultChoysumPath: "/tmp/.choysum"}}
	})
	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "test unit: invalid runtime options") || !strings.Contains(err.Error(), "modulesPath is required") {
		t.Fatalf("expected missing modulesPath validation error, got %v", err)
	}
}

func TestNewTestUnitCmd_KeepFlagBehavior(t *testing.T) {
	oldRunner := runTestRunnerWithDefaults
	defer func() { runTestRunnerWithDefaults = oldRunner }()

	captured := pkgrunner.RunOptions{}
	runTestRunnerWithDefaults = func(ctx context.Context, opts pkgrunner.RunOptions) error {
		captured = opts
		return nil
	}

	newCmd := func() *cobra.Command {
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{cfg: newCommandTestConfig(t.TempDir())}
		})
		if err := cmd.Flags().Set("tap-stdout", "false"); err != nil {
			t.Fatalf("set --tap-stdout=false: %v", err)
		}
		return cmd
	}

	cmd := newCmd()
	if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
		t.Fatalf("run default keep flags: %v", err)
	}
	if captured.Keep {
		t.Fatalf("expected keep=false by default")
	}

	cmd = newCmd()
	if err := cmd.Flags().Set("keep", "true"); err != nil {
		t.Fatalf("set --keep=true: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
		t.Fatalf("run --keep=true: %v", err)
	}
	if !captured.Keep {
		t.Fatalf("expected keep=true when --keep is set")
	}
}

func TestNewTestUnitCmd_RuntimeLogLevel(t *testing.T) {
	oldRunner := runTestRunnerWithDefaults
	defer func() { runTestRunnerWithDefaults = oldRunner }()

	runTestRunnerWithDefaults = func(ctx context.Context, opts pkgrunner.RunOptions) error {
		opts.Env.Logger().Info("runtime info log")
		opts.Env.Logger().Warn("runtime warn log")
		return nil
	}

	newCmd := func(t *testing.T) *cobra.Command {
		t.Helper()
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{cfg: newCommandTestConfig(t.TempDir())}
		})
		if err := cmd.Flags().Set("tap-stdout", "false"); err != nil {
			t.Fatalf("set --tap-stdout=false: %v", err)
		}
		return cmd
	}

	t.Run("defaults to warn", func(t *testing.T) {
		cmd := newCmd(t)
		stderr := captureStderr(t, func() {
			if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
				t.Fatalf("run default runtime log level: %v", err)
			}
		})
		if strings.Contains(stderr, "runtime info log") {
			t.Fatalf("did not expect info log in stderr, got %q", stderr)
		}
		if !strings.Contains(stderr, "runtime warn log") {
			t.Fatalf("expected warn log in stderr, got %q", stderr)
		}
	})

	t.Run("info can be enabled explicitly", func(t *testing.T) {
		cmd := newCmd(t)
		if err := cmd.Flags().Set("runtime-log-level", "info"); err != nil {
			t.Fatalf("set --runtime-log-level=info: %v", err)
		}
		stderr := captureStderr(t, func() {
			if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
				t.Fatalf("run info runtime log level: %v", err)
			}
		})
		if !strings.Contains(stderr, "runtime info log") {
			t.Fatalf("expected info log in stderr, got %q", stderr)
		}
		if !strings.Contains(stderr, "runtime warn log") {
			t.Fatalf("expected warn log in stderr, got %q", stderr)
		}
	})

	t.Run("rejects invalid runtime log level", func(t *testing.T) {
		cmd := newCmd(t)
		if err := cmd.Flags().Set("runtime-log-level", "trace"); err != nil {
			t.Fatalf("set --runtime-log-level=trace: %v", err)
		}
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), "invalid --runtime-log-level") {
			t.Fatalf("expected invalid runtime log level error, got %v", err)
		}
	})
}

func TestNewTestUnitCmd_AdditionalRunEPaths(t *testing.T) {
	t.Run("explicitly disabling be and fe returns no tests selected", func(t *testing.T) {
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{cfg: newCommandTestConfig(t.TempDir())}
		})
		if err := cmd.Flags().Set("be", "false"); err != nil {
			t.Fatalf("set --be=false: %v", err)
		}
		if err := cmd.Flags().Set("fe", "false"); err != nil {
			t.Fatalf("set --fe=false: %v", err)
		}
		if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
			t.Fatalf("expected no-tests-selected path to return nil, got %v", err)
		}
	})

	t.Run("canceled context is wrapped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{ctx: ctx, cfg: newCommandTestConfig(t.TempDir())}
		})
		if err := cmd.Flags().Set("tap-stdout", "false"); err != nil {
			t.Fatalf("set --tap-stdout=false: %v", err)
		}
		cmd.SetContext(ctx)
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), "test run canceled") {
			t.Fatalf("expected canceled error wrapper, got %v", err)
		}
	})

	t.Run("expired deadline is wrapped with timeout message", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{ctx: ctx, cfg: newCommandTestConfig(t.TempDir())}
		})
		if err := cmd.Flags().Set("tap-stdout", "false"); err != nil {
			t.Fatalf("set --tap-stdout=false: %v", err)
		}
		if err := cmd.Flags().Set("timeout", "1s"); err != nil {
			t.Fatalf("set --timeout=1s: %v", err)
		}
		cmd.SetContext(ctx)
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), "test run timed out after 1s") {
			t.Fatalf("expected timed out error wrapper, got %v", err)
		}
	})

	t.Run("tap stdout split env init failure is surfaced", func(t *testing.T) {
		serverCfg := config.NewDefaultServerConfig()
		serverCfg.Environment = "missing-command-env"
		cfg := newCommandTestConfig(t.TempDir())
		cfg.Server = serverCfg
		cmd := newTestUnitCmdFromScope(func() scope.Scope {
			return &commandTestScope{cfg: cfg}
		})
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), "failed to initialize scope for tap stdout") {
			t.Fatalf("expected tap stdout init error, got %v", err)
		}
	})

	for _, tt := range []struct {
		name    string
		envKey  string
		envVal  string
		wantErr string
	}{
		{name: "invalid coverage lines env is rejected", envKey: "CHOYSUM_TEST_COVERAGE_LINES", envVal: "oops", wantErr: `invalid CHOYSUM_TEST_COVERAGE_LINES="oops"`},
		{name: "invalid coverage functions env is rejected", envKey: "CHOYSUM_TEST_COVERAGE_FUNCTIONS", envVal: "oops", wantErr: `invalid CHOYSUM_TEST_COVERAGE_FUNCTIONS="oops"`},
		{name: "invalid coverage branches env is rejected", envKey: "CHOYSUM_TEST_COVERAGE_BRANCHES", envVal: "oops", wantErr: `invalid CHOYSUM_TEST_COVERAGE_BRANCHES="oops"`},
		{name: "invalid coverage statements env is rejected", envKey: "CHOYSUM_TEST_COVERAGE_STATEMENTS", envVal: "oops", wantErr: `invalid CHOYSUM_TEST_COVERAGE_STATEMENTS="oops"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)
			cmd := newTestUnitCmdFromScope(func() scope.Scope {
				return &commandTestScope{cfg: newCommandTestConfig(t.TempDir())}
			})
			if err := cmd.Flags().Set("tap-stdout", "false"); err != nil {
				t.Fatalf("set --tap-stdout=false: %v", err)
			}
			if err := cmd.Flags().Set("coverage", "true"); err != nil {
				t.Fatalf("set --coverage=true: %v", err)
			}
			err := cmd.RunE(cmd, []string{"auth"})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected coverage env parse error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewE2ECmd_ArgsAndEarlyRunE(t *testing.T) {
	scopeGetter := func() scope.Scope { return nil }
	cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
	if cmd.Use != "e2e <module> [-- <playwrightArgs...>]" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if got := cmd.Flags().Lookup("startup-timeout"); got == nil {
		t.Fatal("expected startup-timeout flag to be registered")
	}
	if got := cmd.Flags().Lookup("runtime-log-level"); got == nil {
		t.Fatal("expected runtime-log-level flag to be registered")
	}
	if startupTimeout, err := cmd.Flags().GetDuration("startup-timeout"); err != nil || startupTimeout != 3*time.Minute {
		t.Fatalf("startup-timeout = %v, %v; want %v", startupTimeout, err, 3*time.Minute)
	}

	if err := cmd.Args(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires <module>") {
		t.Fatalf("expected missing module error, got %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set --all: %v", err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("expected --all with no args to pass, got %v", err)
	}

	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected nil scope error, got %v", err)
	}

	scopeGetter = func() scope.Scope { return &commandTestScope{} }
	cmd = newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "e2e: invalid runtime options") {
		t.Fatalf("expected invalid runtime options error, got %v", err)
	}
}

func TestNewE2ECmd_AdditionalRunEPaths(t *testing.T) {
	t.Run("runtime log level defaults to warn and supports override", func(t *testing.T) {
		oldRun := runE2EModule
		defer func() { runE2EModule = oldRun }()

		cfg := newCommandTestConfig(t.TempDir())
		// cfg.NpmPath removed
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }

		t.Run("default warn", func(t *testing.T) {
			var got pkge2e.RunOptions
			runE2EModule = func(ctx context.Context, opts pkge2e.RunOptions) error {
				got = opts
				return nil
			}
			cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
			if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
				t.Fatalf("RunE error: %v", err)
			}
			if got.RuntimeLogLevel != "warn" {
				t.Fatalf("runtime log level = %q, want warn", got.RuntimeLogLevel)
			}
		})

		t.Run("verbose implies debug when flag omitted", func(t *testing.T) {
			var got pkge2e.RunOptions
			runE2EModule = func(ctx context.Context, opts pkge2e.RunOptions) error {
				got = opts
				return nil
			}
			cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
			if err := cmd.Flags().Set("verbose", "true"); err != nil {
				t.Fatalf("set --verbose=true: %v", err)
			}
			if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
				t.Fatalf("RunE error: %v", err)
			}
			if got.RuntimeLogLevel != "debug" {
				t.Fatalf("runtime log level = %q, want debug", got.RuntimeLogLevel)
			}
		})

		t.Run("explicit runtime log level overrides verbose", func(t *testing.T) {
			var got pkge2e.RunOptions
			runE2EModule = func(ctx context.Context, opts pkge2e.RunOptions) error {
				got = opts
				return nil
			}
			cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
			if err := cmd.Flags().Set("verbose", "true"); err != nil {
				t.Fatalf("set --verbose=true: %v", err)
			}
			if err := cmd.Flags().Set("runtime-log-level", "info"); err != nil {
				t.Fatalf("set --runtime-log-level=info: %v", err)
			}
			if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
				t.Fatalf("RunE error: %v", err)
			}
			if got.RuntimeLogLevel != "info" {
				t.Fatalf("runtime log level = %q, want info", got.RuntimeLogLevel)
			}
		})

		t.Run("invalid runtime log level is rejected", func(t *testing.T) {
			cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
			if err := cmd.Flags().Set("runtime-log-level", "trace"); err != nil {
				t.Fatalf("set --runtime-log-level=trace: %v", err)
			}
			err := cmd.RunE(cmd, []string{"auth"})
			if err == nil || !strings.Contains(err.Error(), "invalid --runtime-log-level") {
				t.Fatalf("expected invalid runtime log level error, got %v", err)
			}
		})
	})

	t.Run("uses command context for runner invocation", func(t *testing.T) {
		oldRun := runE2EModule
		defer func() { runE2EModule = oldRun }()

		cfg := newCommandTestConfig(t.TempDir())
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }

		type ctxKey string
		const key ctxKey = "e2e-command-context"
		const wantValue = "propagated"

		runE2EModule = func(ctx context.Context, opts pkge2e.RunOptions) error {
			if got := ctx.Value(key); got != wantValue {
				t.Fatalf("runner context value = %v, want %q", got, wantValue)
			}
			return nil
		}

		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		cmd.SetContext(context.WithValue(context.Background(), key, wantValue))
		if err := cmd.RunE(cmd, []string{"auth"}); err != nil {
			t.Fatalf("RunE error: %v", err)
		}
	})

	t.Run("all uses a shared command-level run-id", func(t *testing.T) {
		oldResolve := resolveE2EModules
		oldRun := runE2EModule
		defer func() {
			resolveE2EModules = oldResolve
			runE2EModule = oldRun
		}()

		resolveE2EModules = func(modulesPath string) ([]string, error) {
			return []string{"auth", "meta", "task"}, nil
		}

		modules := make([]string, 0, 3)
		runIDs := make([]string, 0, 3)
		runE2EModule = func(ctx context.Context, opts pkge2e.RunOptions) error {
			modules = append(modules, opts.Module)
			runIDs = append(runIDs, testingpathing.TestingRunIDFromContext(ctx))
			return nil
		}

		cfg := newCommandTestConfig(t.TempDir())
		// cfg.NpmPath removed
		scopeGetter := func() scope.Scope {
			return &commandTestScope{cfg: cfg}
		}
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE(--all) error: %v", err)
		}

		if len(modules) != 3 || len(runIDs) != 3 {
			t.Fatalf("expected 3 module runs, got modules=%d runIDs=%d", len(modules), len(runIDs))
		}
		if runIDs[0] == "" || runIDs[1] == "" || runIDs[2] == "" {
			t.Fatalf("expected non-empty run-id for each module run, got %v", runIDs)
		}
		if runIDs[0] != runIDs[1] || runIDs[1] != runIDs[2] {
			t.Fatalf("expected shared run-id across --all modules, got %v", runIDs)
		}
	})

	t.Run("all with no runnable modules returns helpful error", func(t *testing.T) {
		cfg := newCommandTestConfig(t.TempDir())
		// cfg.NpmPath removed
		scopeGetter := func() scope.Scope {
			return &commandTestScope{cfg: cfg}
		}
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		err := cmd.RunE(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "e2e: no runnable modules found under") {
			t.Fatalf("expected no runnable modules error, got %v", err)
		}
	})

	t.Run("unknown module is rejected", func(t *testing.T) {
		modulesPath := t.TempDir()
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), `unknown module "auth"`) {
			t.Fatalf("expected unknown module error, got %v", err)
		}
	})

	t.Run("module without e2e specs is skipped", func(t *testing.T) {
		modulesPath := t.TempDir()
		writeCommandPackage(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth"}}`)
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		if err != nil {
			t.Fatalf("expected missing specs to be skipped, got %v", err)
		}
	})

	t.Run("module with invalid specs path is rejected", func(t *testing.T) {
		modulesPath := t.TempDir()
		writeCommandPackage(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"../specs"}}}`)
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), `invalid package.json choysum.e2e.specs for "auth"`) {
			t.Fatalf("expected invalid specs path error, got %v", err)
		}
	})

	t.Run("invalid scenario name is rejected", func(t *testing.T) {
		modulesPath := t.TempDir()
		writeCommandPackage(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"specs"}}}`)
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		if err := cmd.Flags().Set("scenario", "BadScenario"); err != nil {
			t.Fatalf("set --scenario: %v", err)
		}
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), `invalid scenario "BadScenario"`) {
			t.Fatalf("expected invalid scenario error, got %v", err)
		}
	})
}

func TestNewTypecheckCmd_ArgsAndEarlyRunE(t *testing.T) {
	scopeGetter := func() scope.Scope { return nil }
	cmd := newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
	if cmd.Use != "typecheck <app>" {
		t.Fatalf("unexpected command use: %q", cmd.Use)
	}
	if cmd.Flags().Lookup("all") == nil || cmd.Flags().Lookup("keep") == nil {
		t.Fatal("expected --all and --keep flags to be registered")
	}

	if err := cmd.Args(cmd, nil); err == nil || !strings.Contains(err.Error(), "requires exactly 1 app argument") {
		t.Fatalf("expected missing app argument error, got %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set --all: %v", err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("expected --all with no args to pass, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--all cannot be used with an app argument") {
		t.Fatalf("expected --all arg conflict, got %v", err)
	}

	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "typecheck: invalid scope") {
		t.Fatalf("expected invalid scope error, got %v", err)
	}

	scopeGetter = func() scope.Scope { return &commandTestScope{} }
	cmd = newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
	if err := cmd.RunE(cmd, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "typecheck: invalid runtime options") {
		t.Fatalf("expected invalid runtime options error with nil config, got %v", err)
	}
}

func TestNewTypecheckCmd_AdditionalRunEPaths(t *testing.T) {
	t.Run("all with no apps returns success", func(t *testing.T) {
		cfg := newCommandTestConfig(t.TempDir())
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		err := cmd.RunE(cmd, nil)
		if err != nil {
			t.Fatalf("expected success for no apps to check, got %v", err)
		}
	})

	t.Run("unknown target app is rejected", func(t *testing.T) {
		cfg := newCommandTestConfig(t.TempDir())
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), `typecheck: unknown app "auth"`) {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})
}

func TestInstallUpgradeUninstallCommandConstruction(t *testing.T) {
	envGetter := func() scope.Scope { return nil }

	installCmd := newInstallCmd(envGetter)
	if installCmd.Use != "install <module|module@version> [<module|module@version>...]" || installCmd.PreRun == nil || installCmd.Run == nil {
		t.Fatalf("unexpected install command shape: %#v", installCmd)
	}
	if installCmd.Flags().Lookup("with-demo") == nil {
		t.Fatal("expected install command to register --with-demo")
	}
	if got := installCmd.Flags().Lookup("with-demo").Usage; !strings.Contains(got, "package.json") || strings.Contains(strings.ToLower(got), "manifest") {
		t.Fatalf("unexpected install --with-demo usage: %q", got)
	}

	upgradeCmd := newUpgradeCmd(envGetter)
	if upgradeCmd.Use != "upgrade <module|module@version> [<module|module@version>...]" || upgradeCmd.PreRun == nil || upgradeCmd.Run == nil {
		t.Fatalf("unexpected upgrade command shape: %#v", upgradeCmd)
	}
	if upgradeCmd.Flags().Lookup("with-demo") == nil {
		t.Fatal("expected upgrade command to register --with-demo")
	}
	if got := upgradeCmd.Flags().Lookup("with-demo").Usage; !strings.Contains(got, "package.json") || strings.Contains(strings.ToLower(got), "manifest") {
		t.Fatalf("unexpected upgrade --with-demo usage: %q", got)
	}

	uninstallCmd := newUninstallCmd(envGetter)
	if uninstallCmd.Use != "uninstall" || uninstallCmd.PreRun == nil || uninstallCmd.Run == nil {
		t.Fatalf("unexpected uninstall command shape: %#v", uninstallCmd)
	}
}

func TestInstallUpgradeUninstallCommandExitPaths(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
		want     string
		wantCode int
	}{
		{name: "install prerun nil env", scenario: "install_prerun_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "install prerun missing args", scenario: "install_prerun_missing_args", want: "Please specify the module name", wantCode: 1},
		{name: "install run nil env", scenario: "install_run_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "install run executor create error", scenario: "install_run_executor_create_error", want: "Error creating compiler executor", wantCode: 1},
		{name: "install run executor start error", scenario: "install_run_executor_start_error", want: "Error starting compiler executor", wantCode: 1},
		{name: "install run env.Run error", scenario: "install_run_env_run_error", want: "module install failed", wantCode: 1},
		{name: "upgrade prerun nil env", scenario: "upgrade_prerun_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "upgrade prerun missing args", scenario: "upgrade_prerun_missing_args", want: "Please specify the module name", wantCode: 1},
		{name: "upgrade run nil env", scenario: "upgrade_run_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "upgrade run executor create error", scenario: "upgrade_run_executor_create_error", want: "Error creating compiler executor", wantCode: 1},
		{name: "upgrade run executor start error", scenario: "upgrade_run_executor_start_error", want: "Error starting compiler executor", wantCode: 1},
		{name: "upgrade run env.Run error", scenario: "upgrade_run_env_run_error", want: "module upgrade failed", wantCode: 1},
		{name: "uninstall prerun nil env", scenario: "uninstall_prerun_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "uninstall prerun missing args", scenario: "uninstall_prerun_missing_args", want: "Please specify the module name", wantCode: 1},
		{name: "uninstall run nil env", scenario: "uninstall_run_nil_env", want: "scope is not initialized", wantCode: 1},
		{name: "uninstall run executor create error", scenario: "uninstall_run_executor_create_error", want: "Error creating compiler executor", wantCode: 1},
		{name: "uninstall run executor start error", scenario: "uninstall_run_executor_start_error", want: "Error starting compiler executor", wantCode: 1},
		{name: "uninstall run env.Run error", scenario: "uninstall_run_env_run_error", want: "module uninstall failed", wantCode: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := runCommandConstructorExit(t, tt.scenario)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; output=%q", code, tt.wantCode, output)
			}
			if !strings.Contains(output, tt.want) {
				t.Fatalf("expected output to contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestNewCommander_StructureAndPersistentPreRun(t *testing.T) {
	commander := NewCommander(context.Background(), "test-version")
	if commander == nil || commander.rootCmd == nil {
		t.Fatal("expected commander and root command to be initialized")
	}
	if commander.rootCmd.Use != "choysum" {
		t.Fatalf("unexpected root use: %q", commander.rootCmd.Use)
	}
	if flag := commander.rootCmd.PersistentFlags().Lookup("config"); flag == nil || flag.Shorthand != "c" {
		t.Fatalf("expected persistent --config/-c flag, got %#v", flag)
	}

	wantCommands := map[string]bool{
		"install":    false,
		"upgrade":    false,
		"uninstall":  false,
		"module":     false,
		"run":        false,
		"test":       false,
		"type-fetch": false,
	}
	for _, sub := range commander.rootCmd.Commands() {
		if _, ok := wantCommands[sub.Name()]; ok {
			wantCommands[sub.Name()] = true
		}
	}
	for name, seen := range wantCommands {
		if !seen {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}

	t.Run("test subtree and type-fetch carry lightweight annotation", func(t *testing.T) {
		testCmd, _, err := commander.rootCmd.Find([]string{"test"})
		if err != nil {
			t.Fatalf("find test subcommand: %v", err)
		}
		if testCmd == nil {
			t.Fatal("expected test subcommand")
		}
		if got := testCmd.Annotations[lightweightScopeAnnotation]; got != "true" {
			t.Fatalf("test annotation %q = %q, want %q", lightweightScopeAnnotation, got, "true")
		}

		typecheckCmd, _, err := commander.rootCmd.Find([]string{"test", "typecheck"})
		if err != nil {
			t.Fatalf("find test typecheck subcommand: %v", err)
		}
		if !shouldUseLightweightRuntimeScope(typecheckCmd) {
			t.Fatal("expected lightweight scope for test subtree")
		}

		typeFetchCmd, _, err := commander.rootCmd.Find([]string{"type-fetch"})
		if err != nil {
			t.Fatalf("find type-fetch subcommand: %v", err)
		}
		if typeFetchCmd == nil {
			t.Fatal("expected type-fetch subcommand")
		}
		if got := typeFetchCmd.Annotations[lightweightScopeAnnotation]; got != "true" {
			t.Fatalf("type-fetch annotation %q = %q, want %q", lightweightScopeAnnotation, got, "true")
		}
		if !shouldUseLightweightRuntimeScope(typeFetchCmd) {
			t.Fatal("expected lightweight scope for type-fetch")
		}
	})

	t.Run("lightweight scope detection ignores bare command name", func(t *testing.T) {
		root := &cobra.Command{Use: "root"}
		plainTest := &cobra.Command{Use: "test"}
		leaf := &cobra.Command{Use: "leaf"}
		root.AddCommand(plainTest)
		plainTest.AddCommand(leaf)

		if shouldUseLightweightRuntimeScope(leaf) {
			t.Fatal("expected lightweight scope to remain disabled without annotation")
		}
	})

	t.Run("run skips config bootstrap", func(t *testing.T) {
		c := NewCommander(context.Background(), "test-version")
		sub, _, err := c.rootCmd.Find([]string{"run"})
		if err != nil {
			t.Fatalf("find run subcommand: %v", err)
		}
		if err := c.rootCmd.PersistentPreRunE(sub, nil); err != nil {
			t.Fatalf("PersistentPreRunE(run) = %v, want nil", err)
		}
	})

	t.Run("test subtree pre-run does not initialize database", func(t *testing.T) {
		var err error
		workDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(workDir, "modules"), 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
		t.Chdir(workDir)

		t.Setenv("HOME", t.TempDir())
		t.Setenv("CHOYSUM_DB_DIALECT", "postgres")
		t.Setenv("CHOYSUM_DB_DSN", "postgres://127.0.0.1:1/choysum?sslmode=disable")
		t.Setenv("CHOYSUM_AUTH_INTERNAL_KEY", "dev-internal-key")

		c := NewCommander(context.Background(), "test-version")
		sub, _, err := c.rootCmd.Find([]string{"test", "typecheck"})
		if err != nil {
			t.Fatalf("find test typecheck subcommand: %v", err)
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("PersistentPreRunE(test typecheck) panicked: %v", recovered)
			}
		}()

		if err := c.rootCmd.PersistentPreRunE(sub, nil); err != nil {
			t.Fatalf("PersistentPreRunE(test typecheck) = %v, want nil", err)
		}
		if c.runtimeScope == nil || scope.FactoryInputFromScope(c.runtimeScope) == nil {
			t.Fatal("expected environment to be initialized for test subtree")
		}
	})

	t.Run("type-fetch pre-run does not initialize database", func(t *testing.T) {
		var err error
		workDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(workDir, "modules"), 0o755); err != nil {
			t.Fatalf("mkdir modules: %v", err)
		}
		t.Chdir(workDir)

		t.Setenv("HOME", t.TempDir())
		t.Setenv("CHOYSUM_DB_DIALECT", "postgres")
		t.Setenv("CHOYSUM_DB_DSN", "postgres://127.0.0.1:1/choysum?sslmode=disable")
		t.Setenv("CHOYSUM_AUTH_INTERNAL_KEY", "dev-internal-key")

		c := NewCommander(context.Background(), "test-version")
		sub, _, err := c.rootCmd.Find([]string{"type-fetch"})
		if err != nil {
			t.Fatalf("find type-fetch subcommand: %v", err)
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("PersistentPreRunE(type-fetch) panicked: %v", recovered)
			}
		}()

		if err := c.rootCmd.PersistentPreRunE(sub, nil); err != nil {
			t.Fatalf("PersistentPreRunE(type-fetch) = %v, want nil", err)
		}
		if c.runtimeScope == nil || scope.FactoryInputFromScope(c.runtimeScope) == nil {
			t.Fatal("expected environment to be initialized for type-fetch")
		}
	})

	t.Run("missing default config falls back to built-in defaults", func(t *testing.T) {
		var err error
		workDir := t.TempDir()
		homeDir := t.TempDir()
		modulesDir := filepath.Join(workDir, "modules")
		npmDir := filepath.Join(workDir, "node_modules")
		dbPath := filepath.Join(workDir, "app.db")
		for _, dir := range []string{modulesDir, npmDir} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatalf("write db file: %v", err)
		}
		t.Chdir(workDir)
		t.Setenv("HOME", homeDir)
		t.Setenv("CHOYSUM_DB_DIALECT", "sqlite")
		t.Setenv("CHOYSUM_DB_DSN", dbPath)
		t.Setenv("CHOYSUM_AUTH_INTERNAL_KEY", "dev-internal-key")

		c := NewCommander(context.Background(), "test-version")
		sub, _, err := c.rootCmd.Find([]string{"test"})
		if err != nil {
			t.Fatalf("find test subcommand: %v", err)
		}
		err = c.rootCmd.PersistentPreRunE(sub, nil)
		if err != nil {
			t.Fatalf("PersistentPreRunE(test) = %v", err)
		}
		if c.runtimeScope == nil || scope.FactoryInputFromScope(c.runtimeScope) == nil {
			t.Fatal("expected environment to be initialized after default fallback")
		}
	})

	t.Run("explicit config initializes environment", func(t *testing.T) {
		workDir := t.TempDir()
		modulesDir := filepath.Join(workDir, "modules")
		distDir := filepath.Join(workDir, "dist")
		npmDir := filepath.Join(workDir, "node_modules")
		dbPath := filepath.Join(workDir, "app.db")
		for _, dir := range []string{modulesDir, distDir, npmDir} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatalf("write db file: %v", err)
		}
		cfgPath := filepath.Join(workDir, "config.yaml")
		cfgBody := strings.Join([]string{
			"default_choysum_path: " + filepath.Join(workDir, ".choysum"),
			"modules_path: " + modulesDir,
			"dist_path: " + distDir,
			"db:",
			"  dialect: sqlite",
			"  dsn: " + dbPath,
		}, "\n") + "\n"
		if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
			t.Fatalf("write config file: %v", err)
		}

		c := NewCommander(context.Background(), "test-version")
		if err := c.rootCmd.PersistentFlags().Set("config", cfgPath); err != nil {
			t.Fatalf("set persistent config flag: %v", err)
		}
		sub, _, err := c.rootCmd.Find([]string{"test"})
		if err != nil {
			t.Fatalf("find test subcommand: %v", err)
		}
		if err := c.rootCmd.PersistentPreRunE(sub, nil); err != nil {
			t.Fatalf("PersistentPreRunE(test) = %v", err)
		}
		if c.runtimeScope == nil || scope.FactoryInputFromScope(c.runtimeScope) == nil {
			t.Fatal("expected environment to be initialized after PersistentPreRunE")
		}
	})
}

func TestPrintErrorBlock(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	printErrorBlock("boom", "because", "retry")
	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	out := string(data)
	for _, want := range []string{"ERROR: boom", "REASON: because", "NEXT: retry"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestPersistentConfigFlagExistsOnRoot(t *testing.T) {
	cmd := &cobra.Command{Use: "choysum"}
	cmd.PersistentFlags().StringP("config", "c", "", "config")
	if flag := cmd.PersistentFlags().Lookup("config"); flag == nil || flag.Shorthand != "c" {
		t.Fatalf("expected config shorthand c, got %#v", flag)
	}
}
