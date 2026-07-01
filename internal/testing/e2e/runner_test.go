// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
)

func writeTempE2EConfig(t *testing.T, modulesPath string) string {
	t.Helper()
	runDir := t.TempDir()
	distDir := filepath.Join(runDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	configPath := filepath.Join(runDir, "config.yaml")
	configYAML := "default_choysum_path: \"" + filepath.Join(runDir, ".choysum") + "\"\n" +
		"modules_path: \"" + modulesPath + "\"\n" +
		"dist_path: \"" + distDir + "\"\n" +
		"npm_path: \"\"\n" +
		"log:\n  level: \"info\"\n" +
		"db:\n  dialect: \"sqlite\"\n  dsn: \"file:" + filepath.Join(runDir, "db.sqlite") + "?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL\"\n" +
		"server:\n  bindAddress: \"127.0.0.1\"\n  port: 18080\n  hotReload: false\n" +
		"compile:\n  production: false\n"
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeExecFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writePackageFile(t *testing.T, modulesPath, app, content string) {
	t.Helper()
	dir := filepath.Join(modulesPath, app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func TestRunModuleInputValidation(t *testing.T) {
	err := RunModule(context.Background(), RunOptions{})
	if err == nil || !strings.Contains(err.Error(), "module is required") {
		t.Fatalf("expected module required error, got %v", err)
	}

	err = RunModule(context.Background(), RunOptions{Module: "auth"})
	if err == nil || !strings.Contains(err.Error(), "modules_path is required") {
		t.Fatalf("expected modules path error, got %v", err)
	}

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	err = RunModule(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, Scenarios: []string{"Bad Name"}})
	if err == nil || !strings.Contains(err.Error(), "invalid scenario") {
		t.Fatalf("expected invalid scenario error, got %v", err)
	}

	err = RunModule(context.Background(), RunOptions{Module: "missing", ModulesPath: modulesPath})
	if err == nil || !strings.Contains(err.Error(), "unknown module") {
		t.Fatalf("expected unknown module error, got %v", err)
	}

	err = RunModule(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, Scenarios: []string{""}})
	if err == nil || !strings.Contains(err.Error(), "scenario name cannot be empty") {
		t.Fatalf("expected empty scenario error, got %v", err)
	}
}

func TestRunModuleFastFailsWhenPlaywrightMissing(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	oldRunOneScenarioHook := runOneScenarioHook
	runOneScenarioCalled := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		runOneScenarioCalled = true
		return nil
	}
	defer func() { runOneScenarioHook = oldRunOneScenarioHook }()

	t.Setenv("PATH", "")
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(t.TempDir(), "missing-global-node-modules"))
	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "missing 1 required module(s): @playwright/test") {
		t.Fatalf("expected playwright missing error, got %v", err)
	}
	if !strings.Contains(err.Error(), "npm install -g @playwright/test") {
		t.Fatalf("expected npm global install hint in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "npx playwright install --with-deps chromium") {
		t.Fatalf("expected playwright browser install hint in error, got %v", err)
	}
	if runOneScenarioCalled {
		t.Fatalf("expected fast-fail before runOneScenario")
	}
}

func TestRunModuleFastFailsWhenPlaywrightPackageMissing(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	oldRunOneScenarioHook := runOneScenarioHook
	runOneScenarioCalled := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		runOneScenarioCalled = true
		return nil
	}
	defer func() { runOneScenarioHook = oldRunOneScenarioHook }()

	binDir := t.TempDir()
	writeExecFile(t, filepath.Join(binDir, "playwright"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(t.TempDir(), "missing-global-node-modules"))

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "missing 1 required module(s): @playwright/test") {
		t.Fatalf("expected playwright package missing error, got %v", err)
	}
	if !strings.Contains(err.Error(), "npm install -g @playwright/test") {
		t.Fatalf("expected npm global install hint in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "npx playwright install --with-deps chromium") {
		t.Fatalf("expected playwright browser install hint in error, got %v", err)
	}
	if runOneScenarioCalled {
		t.Fatalf("expected fast-fail before runOneScenario")
	}
}

func TestRunModuleFastFailsWhenSpecImportDependencyMissing(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "deps.spec.ts"), []byte("import { test } from '@playwright/test'\nimport { createPromiseClient } from '@connectrpc/connect'\nvoid createPromiseClient\ntest('deps', async () => {})\n"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	oldRunOneScenarioHook := runOneScenarioHook
	runOneScenarioCalled := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		runOneScenarioCalled = true
		return nil
	}
	defer func() { runOneScenarioHook = oldRunOneScenarioHook }()

	binDir := t.TempDir()
	writeExecFile(t, filepath.Join(binDir, "playwright"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	globalRoot := filepath.Join(t.TempDir(), "global-node-modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "@playwright", "test"), 0o755); err != nil {
		t.Fatalf("mkdir global playwright package: %v", err)
	}
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "@connectrpc/connect") {
		t.Fatalf("expected spec-import dependency missing error, got %v", err)
	}
	if runOneScenarioCalled {
		t.Fatalf("expected fast-fail before runOneScenario")
	}
}

func TestRunModuleBuildsStaticDependencyCache(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "cache.spec.ts"), []byte("import { test } from '@playwright/test'\nimport { createPromiseClient } from '@connectrpc/connect'\nvoid createPromiseClient\ntest('cache', async () => {})\n"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	npmPath := filepath.Join(t.TempDir(), "node_modules")
	if err := os.MkdirAll(filepath.Join(npmPath, "@playwright", "test"), 0o755); err != nil {
		t.Fatalf("mkdir playwright package dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(npmPath, "@connectrpc", "connect"), 0o755); err != nil {
		t.Fatalf("mkdir connect package dir: %v", err)
	}
	writeExecFile(t, filepath.Join(npmPath, ".bin", "playwright"), "#!/bin/sh\nexit 0\n")

	oldRunOneScenarioHook := runOneScenarioHook
	defer func() { runOneScenarioHook = oldRunOneScenarioHook }()

	scenarioCalls := 0
	var staticRequired []string
	var staticSpecRequired []string
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		scenarioCalls++
		staticRequired = append([]string{}, opts.staticRequiredModules...)
		staticSpecRequired = append([]string{}, opts.staticSpecRequiredModules...)
		return nil
	}

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		NpmPath:     npmPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err != nil {
		t.Fatalf("RunModule error: %v", err)
	}
	if scenarioCalls != 1 {
		t.Fatalf("expected one scenario invocation, got %d", scenarioCalls)
	}
	if !slices.Contains(staticSpecRequired, "@playwright/test") || !slices.Contains(staticSpecRequired, "@connectrpc/connect") {
		t.Fatalf("expected cached spec required modules, got %#v", staticSpecRequired)
	}
	if !slices.Contains(staticRequired, "@playwright/test") || !slices.Contains(staticRequired, "@connectrpc/connect") {
		t.Fatalf("expected cached runtime required modules, got %#v", staticRequired)
	}
}

func TestRunModuleCanceledContextCleansResources(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "cancel.spec.ts"), []byte("import { test } from '@playwright/test'\ntest('cancel', async () => {})\n"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	npmPath := filepath.Join(t.TempDir(), "node_modules")
	if err := os.MkdirAll(filepath.Join(npmPath, "@playwright", "test"), 0o755); err != nil {
		t.Fatalf("mkdir global playwright package dir: %v", err)
	}
	writeExecFile(t, filepath.Join(npmPath, ".bin", "playwright"), "#!/bin/sh\nexit 0\n")

	workDir := t.TempDir()
	tmpPath := t.TempDir()

	oldInstallForE2EHook := installForE2EHook
	oldApplyScenarioFixturesHook := applyScenarioFixturesHook
	oldSeedModuleIndexHook := seedModuleIndexHook
	oldStartServerHook := startServerHook
	oldWaitForHTTP200Hook := waitForHTTP200Hook
	oldRunPlaywrightHook := runPlaywrightHook
	defer func() {
		installForE2EHook = oldInstallForE2EHook
		applyScenarioFixturesHook = oldApplyScenarioFixturesHook
		seedModuleIndexHook = oldSeedModuleIndexHook
		startServerHook = oldStartServerHook
		waitForHTTP200Hook = oldWaitForHTTP200Hook
		runPlaywrightHook = oldRunPlaywrightHook
	}()

	installCalls := 0
	capturedRunDir := ""
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stderr strings.Builder
	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
		installCalls++
		capturedRunDir = filepath.Dir(configPath)
		cancel()
		return context.Canceled
	}
	applyScenarioFixturesHook = func(context.Context, string, []string, map[string]*sourceModulePackage, string, string, bool, io.Writer, *[]string) error {
		t.Fatalf("applyScenarioFixturesHook should not run after canceled install")
		return nil
	}
	seedModuleIndexHook = func(context.Context, string, map[string]*sourceModulePackage) error {
		t.Fatalf("seedModuleIndexHook should not run after canceled install")
		return nil
	}
	startServerHook = func(string, string, string, string) (*exec.Cmd, error) {
		t.Fatalf("startServerHook should not run after canceled install")
		return nil, nil
	}
	waitForHTTP200Hook = func(context.Context, string, time.Duration) error {
		t.Fatalf("waitForHTTP200Hook should not run after canceled install")
		return nil
	}
	runPlaywrightHook = func(context.Context, RunOptions, string, string, string) error {
		t.Fatalf("runPlaywrightHook should not run after canceled install")
		return nil
	}

	err := RunModule(ctx, RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		NpmPath:     npmPath,
		TmpPath:     tmpPath,
		WorkDir:     workDir,
		Stdout:      io.Discard,
		Stderr:      &stderr,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	if installCalls != 1 {
		t.Fatalf("expected exactly one install call before cancel exit, got %d", installCalls)
	}
	if strings.TrimSpace(capturedRunDir) == "" {
		t.Fatal("expected install hook to capture run dir")
	}
	if _, statErr := os.Stat(capturedRunDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected run dir cleanup on cancel, stat err=%v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(workDir, "modules", "node_modules", "@playwright", "test")); !os.IsNotExist(statErr) {
		t.Fatalf("expected temporary global module link cleanup on cancel, lstat err=%v", statErr)
	}
	if !strings.Contains(stderr.String(), "canceled stage=scenario module=auth scenario=default") {
		t.Fatalf("expected scenario cancellation stage log, got %q", stderr.String())
	}
}

func TestRunModuleCanceledContextFastFailLogsEntryStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr strings.Builder
	err := RunModule(ctx, RunOptions{Module: "auth", Stderr: &stderr, Stdout: io.Discard})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled fast-fail, got %v", err)
	}
	if !strings.Contains(stderr.String(), "canceled stage=entry module=auth") {
		t.Fatalf("expected entry cancellation stage log, got %q", stderr.String())
	}
}

func TestDiscoverSourcePackagesAndResolveModules(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","depends":["base"],"e2e":{"specs":"e2e/specs"}}}`)
	writePackageFile(t, modulesPath, "base", `{"name":"@choysum-dev/base","version":"0.0.0","choysum":{"moduleName":"base","application":"base"}}`)
	if err := os.WriteFile(filepath.Join(modulesPath, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	packages, err := discoverSourcePackages(modulesPath)
	if err != nil {
		t.Fatalf("discoverSourcePackages error: %v", err)
	}
	if len(packages) != 2 {
		t.Fatalf("unexpected package count: %d", len(packages))
	}
	if packages["auth"] == nil || packages["auth"].DirName != "auth" {
		t.Fatalf("auth package not parsed correctly: %#v", packages["auth"])
	}

	mods, err := ResolveE2EModules(modulesPath)
	if err != nil {
		t.Fatalf("ResolveE2EModules error: %v", err)
	}
	if !reflect.DeepEqual(mods, []string{"auth"}) {
		t.Fatalf("unexpected e2e modules: %#v", mods)
	}
}

func TestResolveE2EModulesRequiresModulesPath(t *testing.T) {
	if _, err := ResolveE2EModules("  "); err == nil || !strings.Contains(err.Error(), "modules_path is required") {
		t.Fatalf("expected modules_path required error, got %v", err)
	}
}

func TestDiscoverSourcePackagesReadModulesDirError(t *testing.T) {
	missingModulesPath := filepath.Join(t.TempDir(), "missing")
	if _, err := discoverSourcePackages(missingModulesPath); err == nil || !strings.Contains(err.Error(), "read modules dir") {
		t.Fatalf("expected read modules dir error, got %v", err)
	}
}

func TestTopoClosureAndScenarioFixtures(t *testing.T) {
	manifests := map[string]*sourceModulePackage{
		"auth": {Depends: []string{"base"}},
		"base": {},
	}

	order, err := topoClosure("auth", manifests)
	if err != nil {
		t.Fatalf("topoClosure error: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"base", "auth"}) {
		t.Fatalf("unexpected topo order: %#v", order)
	}

	_, err = topoClosure("auth", map[string]*sourceModulePackage{"auth": {Depends: []string{"missing"}}})
	if err == nil || !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}

	_, err = topoClosure("auth", map[string]*sourceModulePackage{"auth": {Depends: []string{"base"}}, "base": {Depends: []string{"auth"}}})
	if err == nil || !strings.Contains(err.Error(), "depends cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}

	m := &sourceModulePackage{E2E: &packageE2E{Scenarios: map[string]packageScene{
		"base":   {Fixtures: []string{"fixtures/base.json"}},
		"child":  {Extends: "base", Fixtures: []string{"fixtures/child.json"}},
		"broken": {Extends: "missing"},
	}}}

	paths, defined, err := resolveScenarioFixtures(m, "child")
	if err != nil || !defined {
		t.Fatalf("expected resolved fixtures, got defined=%v err=%v", defined, err)
	}
	if !reflect.DeepEqual(paths, []string{"fixtures/base.json", "fixtures/child.json"}) {
		t.Fatalf("unexpected fixtures: %#v", paths)
	}

	_, defined, err = resolveScenarioFixtures(m, "unknown")
	if err != nil || defined {
		t.Fatalf("unknown scenario should be undefined without error, defined=%v err=%v", defined, err)
	}

	_, _, err = resolveScenarioFixtures(m, "broken")
	if err == nil || !strings.Contains(err.Error(), "extends missing parent") {
		t.Fatalf("expected missing parent error, got %v", err)
	}
}

func TestE2EUtilityHelpers(t *testing.T) {
	if level, err := resolveRuntimeLogLevel("", false); err != nil || level != "warn" {
		t.Fatalf("default runtime log level = %q, %v; want warn", level, err)
	}
	if level, err := resolveRuntimeLogLevel("", true); err != nil || level != "debug" {
		t.Fatalf("verbose runtime log level = %q, %v; want debug", level, err)
	}
	if level, err := resolveRuntimeLogLevel("info", false); err != nil || level != "info" {
		t.Fatalf("explicit runtime log level = %q, %v; want info", level, err)
	}
	if _, err := resolveRuntimeLogLevel("trace", false); err == nil {
		t.Fatalf("expected invalid runtime log level error")
	}

	h := randHex(8)
	if len(h) != 16 {
		t.Fatalf("unexpected randHex length: %d", len(h))
	}

	if p, err := pickFreePort(); err != nil || p <= 0 {
		t.Fatalf("pickFreePort failed: p=%d err=%v", p, err)
	}

	if !shouldSkipModuleDir("tmp") || !shouldSkipModuleDir(".choysum") || shouldSkipModuleDir("auth") {
		t.Fatalf("shouldSkipModuleDir cases failed")
	}

	repoRoot := t.TempDir()
	packagePath := filepath.Join(repoRoot, "package.json")
	if err := os.WriteFile(packagePath, []byte(`{"name":"@choysum-dev/auth","version":"1.2.3","choysum":{"moduleName":"auth","application":"auth"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	_, version, err := readPackageVersion(packagePath)
	if err != nil || version != "v1.2.3" {
		t.Fatalf("readPackageVersion got version=%q err=%v", version, err)
	}

	logFile := filepath.Join(t.TempDir(), "server.log")
	if err := os.WriteFile(logFile, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	tail := readLogTail(logFile, 16)
	if !strings.Contains(tail, "line3") {
		t.Fatalf("expected tail to contain latest line, got %q", tail)
	}
	wrapped := includeLogTail(context.DeadlineExceeded, logFile)
	if !strings.Contains(wrapped.Error(), "server log tail") {
		t.Fatalf("expected wrapped log tail error, got %v", wrapped)
	}

	runtimePath := filepath.Join(t.TempDir(), "runtime.json")
	info := runtimeInfo{PID: 1, Port: 8080, BaseURL: "http://127.0.0.1:8080"}
	writeRuntime(runtimePath, info)
	raw, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatalf("read runtime file: %v", err)
	}
	var decoded runtimeInfo
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode runtime json: %v", err)
	}
	if decoded.Port != 8080 || decoded.PID != 1 {
		t.Fatalf("unexpected runtime info: %#v", decoded)
	}
}

func TestNewE2ERuntimeScopeIgnoresAmbientChoysumConfigEnv(t *testing.T) {
	modulesPath := t.TempDir()
	configPath := writeTempE2EConfig(t, modulesPath)
	t.Setenv("CHOYSUM_DB_DIALECT", "postgres")
	t.Setenv("CHOYSUM_DB_DSN", "postgres://ambient/db")

	runtimeScope, _, err := newE2ERuntimeScope(context.Background(), configPath)
	if err != nil {
		t.Fatalf("newE2ERuntimeScope error: %v", err)
	}

	dbOpts, ok := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	if !ok {
		t.Fatalf("expected database runtime options")
	}
	if dbOpts.Dialect != "sqlite" {
		t.Fatalf("expected sqlite dialect from temp config, got %q", dbOpts.Dialect)
	}
	if !strings.Contains(dbOpts.DSN, "db.sqlite") {
		t.Fatalf("expected sqlite dsn from temp config, got %q", dbOpts.DSN)
	}
}

func TestFilteredE2EEnvDropsAmbientChoysumConfigOverrides(t *testing.T) {
	env := filteredE2EEnv([]string{
		"PATH=/usr/bin",
		"CHOYSUM_DB_DIALECT=postgres",
		"CHOYSUM_SERVER_HOT_RELOAD=true",
		"CHOYSUM_E2E_BASE_URL=http://127.0.0.1:18080",
	})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "CHOYSUM_DB_DIALECT=postgres") {
		t.Fatalf("expected ambient db override to be removed, got %q", joined)
	}
	if strings.Contains(joined, "CHOYSUM_SERVER_HOT_RELOAD=true") {
		t.Fatalf("expected ambient server override to be removed, got %q", joined)
	}
	if !strings.Contains(joined, "CHOYSUM_E2E_BASE_URL=http://127.0.0.1:18080") {
		t.Fatalf("expected e2e env to be preserved, got %q", joined)
	}
	if !strings.Contains(joined, "PATH=/usr/bin") {
		t.Fatalf("expected unrelated env to be preserved, got %q", joined)
	}
}

func TestFilteredE2EEnv_EmptyAndMalformedEntries(t *testing.T) {
	if env := filteredE2EEnv(nil); env != nil {
		t.Fatalf("filteredE2EEnv(nil) = %#v, want nil", env)
	}

	filtered := filteredE2EEnv([]string{"MALFORMED", "CHOYSUM_TOKEN=abc", "CHOYSUM_E2E_TOKEN=ok"})
	joined := strings.Join(filtered, "\n")
	if !strings.Contains(joined, "MALFORMED") {
		t.Fatalf("expected malformed env entry to be preserved, got %q", joined)
	}
	if strings.Contains(joined, "CHOYSUM_TOKEN=abc") {
		t.Fatalf("expected CHOYSUM_ non-e2e env to be removed, got %q", joined)
	}
	if !strings.Contains(joined, "CHOYSUM_E2E_TOKEN=ok") {
		t.Fatalf("expected CHOYSUM_E2E_ env to be preserved, got %q", joined)
	}
}

func TestNewE2ERuntimeOptionsAndValidate(t *testing.T) {
	noPath := newE2ERuntimeOptions(scope.PathsRuntimeOptions{}, false)
	if noPath.modulesPath != "" {
		t.Fatalf("newE2ERuntimeOptions(no path).modulesPath = %q, want empty", noPath.modulesPath)
	}
	if err := noPath.Validate(); err == nil || !strings.Contains(err.Error(), "modulesPath is required") {
		t.Fatalf("Validate(no path) error = %v, want modulesPath required", err)
	}

	valid := newE2ERuntimeOptions(scope.PathsRuntimeOptions{ModulesPath: "/workspace/modules"}, true)
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid path) error = %v", err)
	}

	blank := newE2ERuntimeOptions(scope.PathsRuntimeOptions{ModulesPath: "   "}, true)
	if err := blank.Validate(); err == nil || !strings.Contains(err.Error(), "modulesPath is required") {
		t.Fatalf("Validate(blank path) error = %v, want modulesPath required", err)
	}
}

func TestWriteE2EProgressAndRuntimeScopeValidation(t *testing.T) {
	writeE2EProgress(nil, "ignored %s", "output")

	var out strings.Builder
	writeE2EProgress(&out, "# %s %d\n", "prepare", 1)
	if out.String() != "# prepare 1\n" {
		t.Fatalf("writeE2EProgress() output = %q, want %q", out.String(), "# prepare 1\\n")
	}

	missingConfigPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	if _, _, err := newE2ERuntimeScope(context.Background(), missingConfigPath); err == nil {
		t.Fatal("expected config load error for missing e2e config path")
	}
}

func TestRunPlaywrightNoSpecs(t *testing.T) {
	specsDir := t.TempDir()
	err := runPlaywright(context.Background(), RunOptions{WorkDir: t.TempDir()}, specsDir, "http://127.0.0.1:9999", filepath.Join(t.TempDir(), "runtime.json"))
	if err == nil || !strings.Contains(err.Error(), "no playwright specs found") {
		t.Fatalf("expected no specs error, got %v", err)
	}
}

func TestRunPlaywrightBranches(t *testing.T) {
	t.Setenv("PATH", "")

	specsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(specsDir, "ok.spec.ts"), []byte("import { test } from '@playwright/test'\nimport { createPromiseClient } from '@connectrpc/connect'\nvoid createPromiseClient\ntest('ok', async () => {})\n"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}
	runtimePath := filepath.Join(t.TempDir(), "runtime.json")
	runtimeGeneratedPB := filepath.Join(filepath.Dir(runtimePath), ".choysum", "generated", "web", "auth", "pb", "auth_pb.ts")
	if err := os.MkdirAll(filepath.Dir(runtimeGeneratedPB), 0o755); err != nil {
		t.Fatalf("create runtime generated dir: %v", err)
	}
	if err := os.WriteFile(runtimeGeneratedPB, []byte("import { Message } from '@bufbuild/protobuf'\nexport const authPbMessage = Message\n"), 0o644); err != nil {
		t.Fatalf("write runtime generated pb file: %v", err)
	}
	runtimeGeneratedModules, err := collectRuntimeGeneratedModules(runtimePath)
	if err != nil {
		t.Fatalf("collect runtime generated modules: %v", err)
	}
	if !slices.Contains(runtimeGeneratedModules, "@bufbuild/protobuf") {
		t.Fatalf("expected runtime generated modules to include @bufbuild/protobuf, got %#v", runtimeGeneratedModules)
	}
	requiredFromSpecs, err := requiredPlaywrightModulesFromSpecFiles([]string{filepath.Join(specsDir, "ok.spec.ts")})
	if err != nil {
		t.Fatalf("collect required modules from specs: %v", err)
	}
	requiredSet := map[string]struct{}{}
	for _, moduleName := range requiredFromSpecs {
		requiredSet[moduleName] = struct{}{}
	}
	for _, moduleName := range runtimeGeneratedModules {
		requiredSet[moduleName] = struct{}{}
	}
	requiredCombined := make([]string, 0, len(requiredSet))
	for moduleName := range requiredSet {
		requiredCombined = append(requiredCombined, moduleName)
	}
	runtimeMissing := missingRequiredNodeModules(requiredCombined, runtimeE2EModuleRoots(runtimePath)...)
	if !slices.Contains(runtimeMissing, "@bufbuild/protobuf") {
		t.Fatalf("expected runtime roots missing @bufbuild/protobuf before linking, got %#v", runtimeMissing)
	}

	err = runPlaywright(context.Background(), RunOptions{WorkDir: t.TempDir(), NpmPath: t.TempDir()}, specsDir, "http://127.0.0.1:9999", runtimePath)
	if err == nil || !strings.Contains(err.Error(), "missing 1 required module(s): @playwright/test") {
		t.Fatalf("expected missing playwright error, got %v", err)
	}

	repoRoot := t.TempDir()
	npmPath := filepath.Join(t.TempDir(), "node_modules")
	binPath := filepath.Join(npmPath, ".bin", "playwright")
	envPath := filepath.Join(t.TempDir(), "playwright-env.txt")
	linkStatePath := filepath.Join(t.TempDir(), "playwright-link-state.txt")
	if err := os.MkdirAll(filepath.Join(npmPath, "@playwright", "test"), 0o755); err != nil {
		t.Fatalf("create global playwright package dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(npmPath, "@connectrpc", "connect"), 0o755); err != nil {
		t.Fatalf("create global connect package dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(npmPath, "@bufbuild", "protobuf"), 0o755); err != nil {
		t.Fatalf("create global protobuf package dir: %v", err)
	}
	writeExecFile(t, binPath, "#!/bin/sh\nrun_dir=${CHOYSUM_E2E_RUNTIME_JSON%/*}\npw=0\nconnect=0\nrepo_pb=0\nruntime_pb=0\nif [ -L \"$PWD/modules/node_modules/@playwright/test\" ]; then pw=1; fi\nif [ -L \"$PWD/modules/node_modules/@connectrpc/connect\" ]; then connect=1; fi\nif [ -L \"$PWD/modules/node_modules/@bufbuild/protobuf\" ]; then repo_pb=1; fi\nif [ -L \"$run_dir/.choysum/generated/node_modules/@bufbuild/protobuf\" ]; then runtime_pb=1; fi\nif [ \"$pw\" = \"1\" ] && [ \"$connect\" = \"1\" ] && [ \"$repo_pb\" = \"1\" ] && [ \"$runtime_pb\" = \"1\" ]; then printf '%s' \"linked pw=$pw connect=$connect repo_pb=$repo_pb runtime_pb=$runtime_pb run_dir=$run_dir runtime=$CHOYSUM_E2E_RUNTIME_JSON\" > \""+linkStatePath+"\"; else printf '%s' \"missing pw=$pw connect=$connect repo_pb=$repo_pb runtime_pb=$runtime_pb run_dir=$run_dir runtime=$CHOYSUM_E2E_RUNTIME_JSON\" > \""+linkStatePath+"\"; fi\nprintf '%s' \"$PW_DISABLE_TS_ESM\" > \""+envPath+"\"\nexit 0\n")

	err = runPlaywright(context.Background(), RunOptions{WorkDir: repoRoot, NpmPath: npmPath}, specsDir, "http://127.0.0.1:9999", runtimePath)
	if err != nil {
		t.Fatalf("expected playwright success, got %v", err)
	}
	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read playwright env file: %v", err)
	}
	if string(raw) != "1" {
		t.Fatalf("expected PW_DISABLE_TS_ESM=1, got %q", string(raw))
	}
	rawLinkState, err := os.ReadFile(linkStatePath)
	if err != nil {
		t.Fatalf("read playwright link state file: %v", err)
	}
	if !strings.HasPrefix(string(rawLinkState), "linked") {
		t.Fatalf("expected temporary playwright/connect/protobuf package links, got %q", string(rawLinkState))
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, "modules", "node_modules", "@playwright", "test")); !os.IsNotExist(err) {
		t.Fatalf("expected temporary playwright package link cleaned, got err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, "modules", "node_modules", "@connectrpc", "connect")); !os.IsNotExist(err) {
		t.Fatalf("expected temporary connect package link cleaned, got err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, "modules", "node_modules", "@bufbuild", "protobuf")); !os.IsNotExist(err) {
		t.Fatalf("expected temporary protobuf package link cleaned, got err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(runtimePath), ".choysum", "generated", "node_modules", "@bufbuild", "protobuf")); !os.IsNotExist(err) {
		t.Fatalf("expected temporary runtime protobuf package link cleaned, got err=%v", err)
	}
}

func TestResolvePlaywrightCommandSearchesAcceptedPreflightRoots(t *testing.T) {
	t.Run("finds playwright in modules node_modules bin", func(t *testing.T) {
		workDir := t.TempDir()
		t.Setenv("PATH", "")
		playwrightPath := filepath.Join(workDir, "modules", "node_modules", ".bin", "playwright")
		writeExecFile(t, playwrightPath, "#!/bin/sh\nexit 0\n")

		gotBin, gotBinDir, err := resolvePlaywrightCommand(RunOptions{WorkDir: workDir})
		if err != nil {
			t.Fatalf("resolvePlaywrightCommand() error = %v", err)
		}
		if gotBin != playwrightPath {
			t.Fatalf("playwright bin = %q, want %q", gotBin, playwrightPath)
		}
		if gotBinDir != filepath.Dir(playwrightPath) {
			t.Fatalf("playwright bin dir = %q, want %q", gotBinDir, filepath.Dir(playwrightPath))
		}
	})

	t.Run("finds playwright in global npm root bin", func(t *testing.T) {
		workDir := t.TempDir()
		globalRoot := filepath.Join(t.TempDir(), "global-node-modules")
		t.Setenv("PATH", "")
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)
		playwrightPath := filepath.Join(globalRoot, ".bin", "playwright")
		writeExecFile(t, playwrightPath, "#!/bin/sh\nexit 0\n")

		gotBin, gotBinDir, err := resolvePlaywrightCommand(RunOptions{WorkDir: workDir})
		if err != nil {
			t.Fatalf("resolvePlaywrightCommand() error = %v", err)
		}
		if gotBin != playwrightPath {
			t.Fatalf("playwright bin = %q, want %q", gotBin, playwrightPath)
		}
		if gotBinDir != filepath.Dir(playwrightPath) {
			t.Fatalf("playwright bin dir = %q, want %q", gotBinDir, filepath.Dir(playwrightPath))
		}
	})
}

func TestEnsureE2EGlobalModuleLinksAtCleansUpOnSymlinkFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permission semantics differ on windows")
	}

	localRoot := filepath.Join(t.TempDir(), "local", "node_modules")
	globalRoot := filepath.Join(t.TempDir(), "global", "node_modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "left-pad"), 0o755); err != nil {
		t.Fatalf("mkdir global left-pad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(globalRoot, "@scoped", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir global @scoped/pkg: %v", err)
	}
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		t.Fatalf("mkdir local node_modules root: %v", err)
	}

	blockedScopeDir := filepath.Join(localRoot, "@scoped")
	if err := os.MkdirAll(blockedScopeDir, 0o755); err != nil {
		t.Fatalf("mkdir local @scoped dir: %v", err)
	}
	if err := os.Chmod(blockedScopeDir, 0o555); err != nil {
		t.Fatalf("chmod local @scoped dir readonly: %v", err)
	}
	defer func() {
		_ = os.Chmod(blockedScopeDir, 0o755)
	}()

	cleanup, err := ensureE2EGlobalModuleLinksAt(localRoot, globalRoot, []string{"left-pad", "@scoped/pkg"})
	if err == nil {
		cleanup()
		t.Fatal("expected symlink failure for readonly scoped directory")
	}
	if _, statErr := os.Lstat(filepath.Join(localRoot, "left-pad")); !os.IsNotExist(statErr) {
		t.Fatalf("expected rollback to remove previously created left-pad link, lstat err=%v", statErr)
	}
}

func TestRequiredPlaywrightModulesFromSpecFiles(t *testing.T) {
	specFile := filepath.Join(t.TempDir(), "imports.spec.ts")
	content := `
import { test } from '@playwright/test'
import { createPromiseClient } from '@connectrpc/connect'
import { rpc } from '@choysum-dev/core/web/rpc'
import path from 'node:path'
import alias from '@/lib/client'
import rel from './local'
const debounceModule = import('lodash/debounce')
const workspaceImport = import('@choysum-dev/base/web/demo')
void test
void createPromiseClient
void rpc
void path
void alias
void rel
void debounceModule
void workspaceImport
`
	if err := os.WriteFile(specFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	required, err := requiredPlaywrightModulesFromSpecFiles([]string{specFile})
	if err != nil {
		t.Fatalf("requiredPlaywrightModulesFromSpecFiles error: %v", err)
	}
	expected := []string{"@connectrpc/connect", "@playwright/test", "lodash"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("required modules = %#v, want %#v", required, expected)
	}
}

func TestCollectPackageModuleDependencies_IgnoresWorkspaceScopedModules(t *testing.T) {
	pkg := &sourceModulePackage{
		Dependencies: map[string]string{
			"@choysum-dev/core":   "workspace:*",
			"@connectrpc/connect": "^2.1.1",
		},
		PeerDependencies: map[string]string{
			"@choysum-dev/base": "workspace:*",
			"lodash":            "^4.17.21",
		},
	}

	got := collectPackageModuleDependencies(pkg)
	want := []string{"@connectrpc/connect", "lodash"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectPackageModuleDependencies = %#v, want %#v", got, want)
	}
}

func TestParseJSImportSpecifiers_TrimsQuotedLiterals(t *testing.T) {
	source := `
import '@playwright/test'
const deferred = import("@connectrpc/connect")
void deferred
`

	paths := parseJSImportSpecifiers(source)
	if !slices.Contains(paths, "@playwright/test") {
		t.Fatalf("expected @playwright/test in parsed specifiers, got %v", paths)
	}
	if !slices.Contains(paths, "@connectrpc/connect") {
		t.Fatalf("expected @connectrpc/connect in parsed specifiers, got %v", paths)
	}
	for _, p := range paths {
		if strings.Contains(p, "\"") || strings.Contains(p, "'") {
			t.Fatalf("parsed specifier should not include quotes, got %q", p)
		}
	}
}

func TestRequiredPlaywrightModulesFromSpecFilesIgnoresCommentedImports(t *testing.T) {
	specFile := filepath.Join(t.TempDir(), "comments.spec.ts")
	content := `
// import { broken } from '@comment/line'
/*
import { blocked } from '@comment/block'
const deferred = import('@comment/dynamic')
void deferred
*/
import { test } from '@playwright/test'
import { createPromiseClient } from '@connectrpc/connect'
void test
void createPromiseClient
`
	if err := os.WriteFile(specFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	required, err := requiredPlaywrightModulesFromSpecFiles([]string{specFile})
	if err != nil {
		t.Fatalf("requiredPlaywrightModulesFromSpecFiles error: %v", err)
	}
	expected := []string{"@connectrpc/connect", "@playwright/test"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("required modules = %#v, want %#v", required, expected)
	}
}

func TestRequiredPlaywrightModulesFromSpecFiles_CommentTokensInsideStrings(t *testing.T) {
	specFile := filepath.Join(t.TempDir(), "strings.spec.ts")
	content := `
const globPattern = "src/**/*.spec.ts/*not-a-comment*/"
const url = "https://cdn.example.com/a//b"
import { test } from '@playwright/test'
const deferred = import('lodash/debounce')
void globPattern
void url
void test
void deferred
`
	if err := os.WriteFile(specFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	required, err := requiredPlaywrightModulesFromSpecFiles([]string{specFile})
	if err != nil {
		t.Fatalf("requiredPlaywrightModulesFromSpecFiles error: %v", err)
	}
	expected := []string{"@playwright/test", "lodash"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("required modules = %#v, want %#v", required, expected)
	}
}

func TestRequiredPlaywrightModulesFromSpecFiles_IgnoresBuiltinModulesWithoutNodePrefix(t *testing.T) {
	specFile := filepath.Join(t.TempDir(), "builtins.spec.ts")
	content := `
import net from 'net'
import dns from 'dns/promises'
import vm from 'vm'
import { Worker } from 'worker_threads'
import punycode from 'punycode'
import querystring from 'querystring'
import { StringDecoder } from 'string_decoder'
const debounceModule = import('lodash/debounce')
void net
void dns
void vm
void Worker
void punycode
void querystring
void StringDecoder
void debounceModule
`
	if err := os.WriteFile(specFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	required, err := requiredPlaywrightModulesFromSpecFiles([]string{specFile})
	if err != nil {
		t.Fatalf("requiredPlaywrightModulesFromSpecFiles error: %v", err)
	}
	expected := []string{"@playwright/test", "lodash"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("required modules = %#v, want %#v", required, expected)
	}
}

func TestCollectRuntimeGeneratedModules(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "runtime", "runtime.json")
	generatedRoot := filepath.Join(filepath.Dir(runtimePath), ".choysum", "generated", "web", "auth", "pb")
	if err := os.MkdirAll(generatedRoot, 0o755); err != nil {
		t.Fatalf("mkdir generated root: %v", err)
	}

	if err := os.WriteFile(filepath.Join(generatedRoot, "auth_pb.ts"), []byte(`
import { Message } from '@bufbuild/protobuf'
import path from 'node:path'
import local from './local'
const debounceModule = import('lodash/debounce')
void Message
void path
void local
void debounceModule
`), 0o644); err != nil {
		t.Fatalf("write auth_pb.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(generatedRoot, "types.d.ts"), []byte("import '@ignored/from-dts'\n"), 0o644); err != nil {
		t.Fatalf("write types.d.ts: %v", err)
	}

	required, err := collectRuntimeGeneratedModules(runtimePath)
	if err != nil {
		t.Fatalf("collectRuntimeGeneratedModules error: %v", err)
	}
	expected := []string{"@bufbuild/protobuf", "lodash"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("runtime generated modules = %#v, want %#v", required, expected)
	}
}

func TestWaitForHTTP200Timeout(t *testing.T) {
	err := waitForHTTP200(context.Background(), "http://127.0.0.1:1/readyz", 200*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestWaitForHTTP200SuccessAndIncludeLogTailFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := waitForHTTP200(context.Background(), srv.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("expected waitForHTTP200 success, got %v", err)
	}

	baseErr := errors.New("base")
	wrapped := includeLogTail(baseErr, filepath.Join(t.TempDir(), "missing.log"))
	if !errors.Is(wrapped, baseErr) {
		t.Fatalf("expected original error when log missing, got %v", wrapped)
	}
}

func TestInstallApplySeedConfigLoadErrors(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing-config.yaml")

	err := installForE2E(ctx, missing, "auth", false)
	if err == nil || !strings.Contains(err.Error(), "load temp config") {
		t.Fatalf("expected install config load error, got %v", err)
	}

	loaded := []string{}
	err = applyScenarioFixtures(ctx, missing, []string{"auth"}, map[string]*sourceModulePackage{}, "default", "auth", false, io.Discard, &loaded)
	if err == nil || !strings.Contains(err.Error(), "load temp config") {
		t.Fatalf("expected apply fixtures config load error, got %v", err)
	}

	err = seedModuleIndexForE2E(ctx, missing, map[string]*sourceModulePackage{})
	if err == nil || !strings.Contains(err.Error(), "load temp config") {
		t.Fatalf("expected seed module index config load error, got %v", err)
	}
}

func TestStartAndStopServerBranches(t *testing.T) {
	workDir := t.TempDir()
	missingLogDir := filepath.Join(workDir, "missing", "server.log")
	_, err := startServer(workDir, filepath.Join(workDir, "config.yaml"), missingLogDir, "")
	if err == nil || !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("expected open log file error, got %v", err)
	}

	logPath := filepath.Join(workDir, "server.log")
	_, err = startServer(workDir, filepath.Join(workDir, "config.yaml"), logPath, filepath.Join(workDir, "missing-binary"))
	if err == nil || !strings.Contains(err.Error(), "start server") {
		t.Fatalf("expected start server error, got %v", err)
	}

	stopServer(nil)
	stopServer(&exec.Cmd{})
}

func TestStopServerSignalsRunningProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix process-group signaling is covered in !windows builds")
	}

	cmd := exec.Command("sleep", "30")
	setServerProcessAttrs(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	done := make(chan struct{})
	go func() {
		stopServer(cmd)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		t.Fatal("stopServer did not return in time")
	}
}

func TestSignalServerProcessFallbackWithoutSetpgid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fallback SIGTERM path is covered in !windows builds")
	}

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	signalServerProcess(cmd)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		t.Fatal("signalServerProcess fallback did not terminate process in time")
	case <-done:
	}
}

func TestServerProcessHelpersNilSafety(t *testing.T) {
	setServerProcessAttrs(nil)
	signalServerProcess(nil)
	signalServerProcess(&exec.Cmd{})
}

func TestSetServerProcessAttrsPreservesExistingSysProcAttr(t *testing.T) {
	original := &syscall.SysProcAttr{}
	cmd := &exec.Cmd{SysProcAttr: original}

	setServerProcessAttrs(cmd)

	if cmd.SysProcAttr != original {
		t.Fatalf("expected existing SysProcAttr pointer to be preserved")
	}
}

func TestApplyScenarioFixturesLogOnlyBranches(t *testing.T) {
	modulesPath := t.TempDir()
	configPath := writeTempE2EConfig(t, modulesPath)

	manifests := map[string]*sourceModulePackage{
		"auth": {
			DirName: "auth",
			E2E: &packageE2E{Scenarios: map[string]packageScene{
				"empty": {Fixtures: []string{}},
			}},
		},
		"base": {
			DirName: "base",
			E2E:     &packageE2E{Scenarios: map[string]packageScene{}},
		},
	}

	loaded := []string{}
	var stderr strings.Builder
	err := applyScenarioFixtures(context.Background(), configPath, []string{"auth", "base"}, manifests, "missing", "auth", true, &stderr, &loaded)
	if err != nil {
		t.Fatalf("applyScenarioFixtures missing scenario error: %v", err)
	}
	out := stderr.String()
	if !strings.Contains(out, "no fixtures loaded for module=auth scenario=missing") {
		t.Fatalf("expected target missing scenario log, got %q", out)
	}
	if !strings.Contains(out, "module=base scenario=missing not defined") {
		t.Fatalf("expected verbose dependency missing scenario log, got %q", out)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected no loaded fixtures, got %#v", loaded)
	}

	stderr.Reset()
	err = applyScenarioFixtures(context.Background(), configPath, []string{"auth"}, manifests, "empty", "auth", false, &stderr, &loaded)
	if err != nil {
		t.Fatalf("applyScenarioFixtures empty fixtures error: %v", err)
	}
	if !strings.Contains(stderr.String(), "module=auth scenario=empty defined but has no fixtures") {
		t.Fatalf("expected target empty fixtures log, got %q", stderr.String())
	}
}

func TestApplyScenarioFixturesFixturePathErrorUsesModuleRoot(t *testing.T) {
	modulesPath := t.TempDir()
	configPath := writeTempE2EConfig(t, modulesPath)

	manifests := map[string]*sourceModulePackage{
		"auth": {
			DirName: "auth",
			E2E: &packageE2E{Scenarios: map[string]packageScene{
				"default": {Fixtures: []string{"fixtures/missing.json"}},
			}},
		},
	}

	loaded := []string{}
	err := applyScenarioFixtures(context.Background(), configPath, []string{"auth"}, manifests, "default", "auth", false, io.Discard, &loaded)
	if err == nil {
		t.Fatal("expected applyScenarioFixtures to fail for missing fixture path")
	}
	if len(loaded) != 0 {
		t.Fatalf("expected no loaded fixtures on error, got %#v", loaded)
	}
}

func TestInstallForE2EAndSeedModuleIndexRuntimeBranches(t *testing.T) {
	modulesPath := t.TempDir()
	configPath := writeTempE2EConfig(t, modulesPath)

	err := installForE2E(context.Background(), configPath, "missing-module", false)
	if err == nil {
		t.Fatalf("expected installForE2E to fail for missing module")
	}

	if err := os.MkdirAll(filepath.Join(modulesPath, "auth"), 0o755); err != nil {
		t.Fatalf("mkdir auth dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "auth", "package.json"), []byte(`{"name":"@choysum-dev/auth","version":"1.0.0","choysum":{"moduleName":"auth","application":"auth"}}`), 0o644); err != nil {
		t.Fatalf("write auth package.json: %v", err)
	}

	manifests := map[string]*sourceModulePackage{
		"auth": {DirName: "auth"},
		"tmp":  {DirName: "tmp"},
	}
	err = seedModuleIndexForE2E(context.Background(), configPath, manifests)
	if err != nil {
		t.Fatalf("seedModuleIndexForE2E error: %v", err)
	}
}

func TestMergeRequiredE2ERuntimeModulesUsesDirNameMapping(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","dependencies":{"left-pad":"^1.3.0"}}`)

	closure := []string{"auth-module-key"}
	packages := map[string]*sourceModulePackage{
		"auth-module-key": {DirName: "auth"},
	}

	got, err := mergeRequiredE2ERuntimeModules(modulesPath, closure, packages, []string{"@playwright/test"})
	if err != nil {
		t.Fatalf("mergeRequiredE2ERuntimeModules error: %v", err)
	}

	want := []string{"@playwright/test", "left-pad"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeRequiredE2ERuntimeModules() = %#v, want %#v", got, want)
	}
}
