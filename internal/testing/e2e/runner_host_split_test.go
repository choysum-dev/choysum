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
	filtered2, pass3 := filterE2ESpecsByArgs(files, []string{"", "  ", "/m/auth/e2e/register.spec.ts", "--headed"})
	if len(filtered2) != 1 || !strings.HasSuffix(filtered2[0], "register.spec.ts") {
		t.Fatalf("path filter=%v", filtered2)
	}
	if len(pass3) != 1 || pass3[0] != "--headed" {
		t.Fatalf("pass3=%v", pass3)
	}
	none, _ := filterE2ESpecsByArgs(files, []string{"no-match-at-all"})
	if len(none) != 0 {
		t.Fatalf("expected empty, got %v", none)
	}
	_, _ = filterE2ESpecsByArgs(files, []string{"[invalid"})
}

func TestUniqueScenarioFixtureModules(t *testing.T) {
	got := uniqueScenarioFixtureModules([]string{"base", "meta", "auth", "meta", "", "base"}, "partner")
	want := []string{"base", "auth"}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}

	gotMeta := uniqueScenarioFixtureModules([]string{"meta", "task", "meta"}, "meta")
	wantMeta := []string{"meta", "task"}
	if len(gotMeta) != len(wantMeta) {
		t.Fatalf("meta got=%v want=%v", gotMeta, wantMeta)
	}
	for i := range wantMeta {
		if gotMeta[i] != wantMeta[i] {
			t.Fatalf("meta got=%v want=%v", gotMeta, wantMeta)
		}
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
	withInjectedScenarioHooks(t)
	oldRunHost := runE2EHostHook
	t.Cleanup(func() { runE2EHostHook = oldRunHost })
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		t.Fatal("host should not run")
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
		SpecFilterArgs: []string{"no-match-at-all"},
	}, map[string]*sourceModulePackage{
		"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
	}, "default")
	if err == nil || !strings.Contains(err.Error(), "no e2e specs found") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOneScenarioQJSHostError(t *testing.T) {
	withInjectedScenarioHooks(t)
	oldRunHost := runE2EHostHook
	t.Cleanup(func() { runE2EHostHook = oldRunHost })
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
		return errors.New("host boom")
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

func TestRunOneScenarioDiscoverSpecsError(t *testing.T) {
	withInjectedScenarioHooks(t)

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
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

func TestRunModuleDiscoverSpecsPermissionError(t *testing.T) {
	setE2ETestChromiumPath(t)
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

func TestRunModuleHostPathSkipsScenarioWhenIllegal(t *testing.T) {
	// Covered more specifically in TestRunModuleFastFailsWhenIllegalMarks; keep a
	// split-file smoke that QJS-only modules reach the scenario hook.
	setE2ETestChromiumPath(t)
	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	writeQJSSpec(t, filepath.Join(modulesPath, "auth", "e2e", "q.spec.ts"))

	oldRunOne := runOneScenarioHook
	called := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { runOneScenarioHook = oldRunOne })

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		WorkDir:     t.TempDir(),
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	})
	if err != nil {
		t.Fatalf("qjs-only RunModule: %v", err)
	}
	if !called {
		t.Fatal("expected runOneScenario")
	}
}
