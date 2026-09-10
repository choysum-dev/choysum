// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFilterE2ESpecsByArgs(t *testing.T) {
	files := []string{
		"/m/auth/e2e/smoke.spec.ts",
		"/m/auth/e2e/register.spec.ts",
		"/m/auth/e2e/timezone_wallclock.spec.ts",
	}
	filtered, pass := filterE2ESpecsByArgs(files, []string{"smoke.spec.ts", "--workers=1"})
	if len(filtered) != 1 || !strings.HasSuffix(filtered[0], "smoke.spec.ts") {
		t.Fatalf("filtered=%v", filtered)
	}
	if len(pass) != 1 || pass[0] != "--workers=1" {
		t.Fatalf("passthrough=%v", pass)
	}
	reFiltered, _ := filterE2ESpecsByArgs(files, []string{".*smoke.*"})
	if len(reFiltered) != 1 || !strings.HasSuffix(reFiltered[0], "smoke.spec.ts") {
		t.Fatalf("regex filtered=%v", reFiltered)
	}
	all, pass2 := filterE2ESpecsByArgs(files, nil)
	if len(all) != 3 || len(pass2) != 0 {
		t.Fatalf("all=%v pass=%v", all, pass2)
	}
	// Empty args skipped; path substring / full path / patBase matches.
	filtered2, pass3 := filterE2ESpecsByArgs(files, []string{"", "  ", "/m/auth/e2e/register.spec.ts", "--headed"})
	if len(filtered2) != 1 || !strings.HasSuffix(filtered2[0], "register.spec.ts") {
		t.Fatalf("path filter=%v", filtered2)
	}
	if len(pass3) != 1 || pass3[0] != "--headed" {
		t.Fatalf("pass3=%v", pass3)
	}
	none, _ := filterE2ESpecsByArgs(files, []string{"no-such-spec"})
	if len(none) != 0 {
		t.Fatalf("expected empty, got %v", none)
	}
	// Invalid regex still falls through without panic (Compile fails → skip).
	_, _ = filterE2ESpecsByArgs(files, []string{"[invalid"})
}

func TestPartitionE2ESpecFiles(t *testing.T) {
	dir := t.TempDir()
	pw := filepath.Join(dir, "pw.spec.ts")
	qjs := filepath.Join(dir, "qjs.spec.ts")
	_ = os.WriteFile(pw, []byte("import { test } from '@playwright/test';\n"), 0o644)
	_ = os.WriteFile(qjs, []byte("import { test } from '@choysum/e2e';\n"), 0o644)
	pwFiles, qjsFiles, err := partitionE2ESpecFiles([]string{pw, qjs})
	if err != nil {
		t.Fatal(err)
	}
	if len(pwFiles) != 1 || len(qjsFiles) != 1 {
		t.Fatalf("pw=%v qjs=%v", pwFiles, qjsFiles)
	}
	_, _, err = partitionE2ESpecFiles([]string{filepath.Join(dir, "missing.spec.ts")})
	if err == nil {
		t.Fatal("expected read error")
	}
	_, err = specImportsPlaywright(filepath.Join(dir, "missing.spec.ts"))
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestPartitionAndRunOrderQJSThenPW(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)

	oldInstall := installForE2EHook
	oldApply := applyScenarioFixturesHook
	oldSeed := seedModuleIndexHook
	oldStart := startServerHook
	oldStop := stopServerHook
	oldWait := waitForHTTP200Hook
	oldRunPW := runPlaywrightHook
	oldRunHost := runE2EHostHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runPlaywrightHook = oldRunPW
		runE2EHostHook = oldRunHost
	}()

	var order []string
	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
		return nil
	}
	applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
		return nil
	}
	seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
		return nil
	}
	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
	}
	stopServerHook = func(cmd *exec.Cmd) {}
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error {
		return nil
	}
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		order = append(order, "qjs")
		if len(qjsSpecFiles) != 1 || !strings.HasSuffix(qjsSpecFiles[0], "qjs.spec.ts") {
			t.Fatalf("unexpected qjs files: %v", qjsSpecFiles)
		}
		return nil
	}
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		order = append(order, "pw")
		if len(onlyFiles) != 1 || !strings.HasSuffix(onlyFiles[0], "pw.spec.ts") {
			t.Fatalf("unexpected pw files: %v", onlyFiles)
		}
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "pw.spec.ts"), []byte("import { test } from '@playwright/test';\ntest('pw', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "qjs.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('qjs', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifests := map[string]*sourceModulePackage{
		"auth": {
			DirName: "auth",
			E2E:     &packageE2E{Specs: "e2e"},
		},
	}
	err := runOneScenario(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		TmpPath:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}, manifests, "default")
	if err != nil {
		t.Fatalf("runOneScenario: %v", err)
	}
	if strings.Join(order, ",") != "qjs,pw" {
		t.Fatalf("order=%v want qjs,pw", order)
	}
}

func TestPartitionSmokeFilterSkipsPlaywright(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)

	oldInstall := installForE2EHook
	oldApply := applyScenarioFixturesHook
	oldSeed := seedModuleIndexHook
	oldStart := startServerHook
	oldStop := stopServerHook
	oldWait := waitForHTTP200Hook
	oldRunPW := runPlaywrightHook
	oldRunHost := runE2EHostHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runPlaywrightHook = oldRunPW
		runE2EHostHook = oldRunHost
	}()

	var order []string
	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error { return nil }
	applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
		return nil
	}
	seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
		return nil
	}
	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
	}
	stopServerHook = func(cmd *exec.Cmd) {}
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		order = append(order, "qjs")
		return nil
	}
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		order = append(order, "pw")
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "pw.spec.ts"), []byte("import { test } from '@playwright/test';\ntest('pw', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "smoke.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('smoke', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifests := map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}
	err := runOneScenario(context.Background(), RunOptions{
		Module:         "auth",
		ModulesPath:    modulesPath,
		WorkDir:        t.TempDir(),
		TmpPath:        t.TempDir(),
		Stdout:         io.Discard,
		Stderr:         io.Discard,
		PlaywrightArgs: []string{"smoke.spec.ts"},
	}, manifests, "default")
	if err != nil {
		t.Fatalf("runOneScenario: %v", err)
	}
	if strings.Join(order, ",") != "qjs" {
		t.Fatalf("order=%v want qjs only", order)
	}
}

func TestSpecImportsPlaywright(t *testing.T) {
	dir := t.TempDir()
	pw := filepath.Join(dir, "a.spec.ts")
	qjs := filepath.Join(dir, "b.spec.ts")
	_ = os.WriteFile(pw, []byte("import { test } from '@playwright/test';\n"), 0o644)
	_ = os.WriteFile(qjs, []byte("import { test } from '@choysum/e2e';\n"), 0o644)
	ok, err := specImportsPlaywright(pw)
	if err != nil || !ok {
		t.Fatalf("pw: ok=%v err=%v", ok, err)
	}
	ok, err = specImportsPlaywright(qjs)
	if err != nil || ok {
		t.Fatalf("qjs: ok=%v err=%v", ok, err)
	}
}

func TestFilterE2ENodePreflightModules(t *testing.T) {
	got := filterE2ENodePreflightModules([]string{
		"  ",
		"@choysum/e2e",
		"@playwright/test",
		" lodash ",
	})
	if len(got) != 2 || got[0] != "@playwright/test" || got[1] != "lodash" {
		t.Fatalf("got %v", got)
	}
}

func TestCollectRequiredPlaywrightModulesBranches(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-e2e-dir")
	mods, err := collectRequiredPlaywrightModules(missing)
	if err != nil || mods != nil {
		t.Fatalf("missing dir: mods=%v err=%v", mods, err)
	}

	empty := t.TempDir()
	mods, err = collectRequiredPlaywrightModules(empty)
	if err != nil || mods != nil {
		t.Fatalf("empty: mods=%v err=%v", mods, err)
	}

	qjsOnly := t.TempDir()
	if err := os.WriteFile(filepath.Join(qjsOnly, "q.spec.ts"), []byte("import { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err = collectRequiredPlaywrightModules(qjsOnly)
	if err != nil || mods != nil {
		t.Fatalf("qjs-only: mods=%v err=%v", mods, err)
	}

	pwDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pwDir, "p.spec.ts"), []byte("import { test } from '@playwright/test';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err = collectRequiredPlaywrightModules(pwDir)
	if err != nil || len(mods) == 0 {
		t.Fatalf("pw: mods=%v err=%v", mods, err)
	}

	blocked := t.TempDir()
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	if _, err := collectRequiredPlaywrightModules(blocked); err == nil {
		t.Fatal("expected permission error from discover")
	}
}

func withInjectedScenarioHooks(t *testing.T) {
	t.Helper()
	oldInstall := installForE2EHook
	oldApply := applyScenarioFixturesHook
	oldSeed := seedModuleIndexHook
	oldStart := startServerHook
	oldStop := stopServerHook
	oldWait := waitForHTTP200Hook
	t.Cleanup(func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
	})
	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error { return nil }
	applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
		return nil
	}
	seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
		return nil
	}
	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
	}
	stopServerHook = func(cmd *exec.Cmd) {}
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
}

func TestRunOneScenarioNoSpecsAfterFilter(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)
	withInjectedScenarioHooks(t)
	oldRunPW := runPlaywrightHook
	oldRunHost := runE2EHostHook
	t.Cleanup(func() {
		runPlaywrightHook = oldRunPW
		runE2EHostHook = oldRunHost
	})
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		t.Fatal("pw should not run")
		return nil
	}
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		t.Fatal("qjs should not run")
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "a.spec.ts"), []byte("import { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runOneScenario(context.Background(), RunOptions{
		Module:         "auth",
		ModulesPath:    modulesPath,
		WorkDir:        t.TempDir(),
		TmpPath:        t.TempDir(),
		Stdout:         io.Discard,
		Stderr:         io.Discard,
		PlaywrightArgs: []string{"no-match-at-all"},
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "no e2e specs found") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOneScenarioQJSHostError(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)
	withInjectedScenarioHooks(t)
	oldRunHost := runE2EHostHook
	oldRunPW := runPlaywrightHook
	t.Cleanup(func() {
		runE2EHostHook = oldRunHost
		runPlaywrightHook = oldRunPW
	})
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		return errors.New("host boom")
	}
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		t.Fatal("pw should not run after host error")
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "q.spec.ts"), []byte("import { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runOneScenario(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		TmpPath:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "host boom") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOneScenarioPartitionError(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)
	withInjectedScenarioHooks(t)

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(specsDir, "unreadable.spec.ts")
	if err := os.WriteFile(spec, []byte("import { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Discover finds the file by name; partition fails when reading it.
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error {
		return os.Chmod(spec, 0o000)
	}
	t.Cleanup(func() { _ = os.Chmod(spec, 0o644) })

	err := runOneScenario(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		TmpPath:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "read ") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOneScenarioDiscoverSpecsError(t *testing.T) {
	setE2ETestGlobalPlaywrightRoot(t)
	withInjectedScenarioHooks(t)

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Block WalkDir after prepare so discoverPlaywrightSpecFiles fails.
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error {
		if err := os.Chmod(specsDir, 0o000); err != nil {
			return err
		}
		return nil
	}
	t.Cleanup(func() { _ = os.Chmod(specsDir, 0o755) })

	err := runOneScenario(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		TmpPath:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "discover e2e specs") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOneScenarioPreflightNodeModulesError(t *testing.T) {
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(t.TempDir(), "missing-global"))
	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "q.spec.ts"), []byte("import { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runOneScenario(context.Background(), RunOptions{
		Module:                "auth",
		ModulesPath:           modulesPath,
		WorkDir:               t.TempDir(),
		TmpPath:               t.TempDir(),
		Stdout:                io.Discard,
		Stderr:                io.Discard,
		staticRequiredModules: []string{"@playwright/test"},
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "@playwright/test") {
		t.Fatalf("got %v", err)
	}
}

func TestRunModuleDiscoverSpecsPermissionError(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(specsDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(specsDir, 0o755) })

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil {
		t.Fatal("expected discover permission error")
	}
}

func TestRunModuleQJSOnlySkipsPlaywrightResolve(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "q.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('q', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldRunOne := runOneScenarioHook
	called := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { runOneScenarioHook = oldRunOne })

	t.Setenv("PATH", "")
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(t.TempDir(), "missing-global"))
	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err != nil {
		t.Fatalf("qjs-only should skip playwright preflight: %v", err)
	}
	if !called {
		t.Fatal("expected runOneScenario")
	}
}

func TestRunPlaywrightDiscoverAndScanErrors(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "runtime.json")

	blocked := t.TempDir()
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	err := runPlaywright(context.Background(), RunOptions{WorkDir: t.TempDir()}, blocked, "http://127.0.0.1:9", runtimePath, nil)
	if err == nil || !strings.Contains(err.Error(), "discover playwright specs") {
		t.Fatalf("discover: %v", err)
	}

	unreadable := filepath.Join(t.TempDir(), "x.spec.ts")
	if err := os.WriteFile(unreadable, []byte("import { test } from '@playwright/test';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o644) })
	err = runPlaywright(context.Background(), RunOptions{WorkDir: t.TempDir()}, t.TempDir(), "http://127.0.0.1:9", runtimePath, []string{unreadable})
	if err == nil || !strings.Contains(err.Error(), "scan playwright imports") {
		t.Fatalf("scan: %v", err)
	}
}

func TestRequiredPlaywrightModulesHookErrors(t *testing.T) {
	old := requiredPlaywrightModulesFromSpecFilesHook
	t.Cleanup(func() { requiredPlaywrightModulesFromSpecFilesHook = old })
	requiredPlaywrightModulesFromSpecFilesHook = func(specFiles []string) ([]string, error) {
		return nil, errors.New("scan boom")
	}

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "a.spec.ts"), []byte("import { test } from '@playwright/test';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "scan boom") {
		t.Fatalf("RunModule: %v", err)
	}

	mods, err := collectRequiredPlaywrightModules(specsDir)
	if err == nil || !strings.Contains(err.Error(), "scan boom") {
		t.Fatalf("collect: mods=%v err=%v", mods, err)
	}

	requiredPlaywrightModulesFromSpecFilesHook = func(specFiles []string) ([]string, error) {
		return []string{"", "  ", "@choysum/e2e", "@playwright/test"}, nil
	}
	mods, err = collectRequiredPlaywrightModules(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0] != "@playwright/test" {
		t.Fatalf("filter continue: %v", mods)
	}
}

func TestRunModuleResolvePlaywrightCommandError(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "a.spec.ts"), []byte("import { test } from '@playwright/test';\ntest('a', async () => {});\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	globalRoot := filepath.Join(t.TempDir(), "global-node-modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "@playwright", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Package present for preflight, but no playwright binary under roots / PATH.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)

	oldRunOne := runOneScenarioHook
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		t.Fatal("should not run scenario")
		return nil
	}
	t.Cleanup(func() { runOneScenarioHook = oldRunOne })

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		NpmPath:     globalRoot,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "playwright") {
		t.Fatalf("got %v", err)
	}
}
