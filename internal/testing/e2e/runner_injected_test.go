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
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunOneScenarioWithHooksSuccess(t *testing.T) {
	oldInstall := installForE2EHook
	oldApply := applyScenarioFixturesHook
	oldSeed := seedModuleIndexHook
	oldStart := startServerHook
	oldStop := stopServerHook
	oldWait := waitForHTTP200Hook
	oldRunPlaywright := runPlaywrightHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runPlaywrightHook = oldRunPlaywright
	}()

	installCalls := 0
	stopCalled := false
	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
		installCalls++
		return nil
	}
	applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
		*loadedFixtures = append(*loadedFixtures, "auth/fixtures/default.json")
		return nil
	}
	seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
		return nil
	}
	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
	}
	stopServerHook = func(cmd *exec.Cmd) {
		stopCalled = true
	}
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error {
		return nil
	}
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
		if _, err := os.Stat(runtimePath); err != nil {
			t.Fatalf("expected runtime file created before playwright, err=%v", err)
		}
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
	}

	manifests := map[string]*sourceModulePackage{
		"auth": {
			DirName: "auth",
			E2E:     &packageE2E{Specs: "e2e"},
		},
	}

	err := runOneScenario(context.Background(), RunOptions{
		Module:         "auth",
		ModulesPath:    modulesPath,
		WorkDir:        t.TempDir(),
		TmpPath:        t.TempDir(),
		StartupTimeout: time.Second,
		Stderr:         io.Discard,
	}, manifests, "default")
	if err != nil {
		t.Fatalf("runOneScenario error: %v", err)
	}
	if installCalls < 2 {
		t.Fatalf("expected at least auth+meta install calls, got %d", installCalls)
	}
	if !stopCalled {
		t.Fatalf("expected stopServer hook called")
	}
}

func TestRunOneScenarioWithHooksErrorPaths(t *testing.T) {
	oldInstall := installForE2EHook
	oldApply := applyScenarioFixturesHook
	oldSeed := seedModuleIndexHook
	oldStart := startServerHook
	oldStop := stopServerHook
	oldWait := waitForHTTP200Hook
	oldRunPlaywright := runPlaywrightHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runPlaywrightHook = oldRunPlaywright
	}()

	installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error { return nil }
	applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
		return nil
	}
	seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
		return nil
	}
	stopServerHook = func(cmd *exec.Cmd) {}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}

	manifests := map[string]*sourceModulePackage{
		"auth": {
			DirName: "auth",
			E2E:     &packageE2E{Specs: "e2e"},
		},
	}

	startErr := errors.New("start failed")
	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return nil, startErr
	}
	err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
	if !errors.Is(err, startErr) {
		t.Fatalf("expected start server error, got %v", err)
	}

	startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
		return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
	}
	waitErr := errors.New("not ready")
	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return waitErr }
	err = runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
	if !errors.Is(err, waitErr) {
		t.Fatalf("expected wait error, got %v", err)
	}

	waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
	playErr := errors.New("playwright failed")
	runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
		return playErr
	}
	err = runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
	if !errors.Is(err, playErr) {
		t.Fatalf("expected playwright error, got %v", err)
	}
}

func TestRunModuleUsesScenarioHook(t *testing.T) {
	oldRunOne := runOneScenarioHook
	defer func() { runOneScenarioHook = oldRunOne }()

	binDir := t.TempDir()
	writeExecFile(t, filepath.Join(binDir, "playwright"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	calls := 0
	sawDeadline := false
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, manifests map[string]*sourceModulePackage, scenario string) error {
		calls++
		if _, ok := ctx.Deadline(); ok {
			sawDeadline = true
		}
		if strings.TrimSpace(opts.WorkDir) == "" {
			t.Fatalf("expected RunModule to provide non-empty workdir")
		}
		if manifests["auth"] == nil {
			t.Fatalf("expected manifests to include target module")
		}
		return nil
	}

	err := RunModule(context.Background(), RunOptions{
		Module:      "auth",
		ModulesPath: modulesPath,
		Scenarios:   []string{"default", "smoke"},
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("RunModule returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 scenario runs, got %d", calls)
	}
	if !sawDeadline {
		t.Fatalf("expected per-scenario timeout context with deadline")
	}
}

func TestRunModulePropagatesScenarioHookError(t *testing.T) {
	oldRunOne := runOneScenarioHook
	defer func() { runOneScenarioHook = oldRunOne }()

	binDir := t.TempDir()
	writeExecFile(t, filepath.Join(binDir, "playwright"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)

	wantErr := errors.New("scenario failed")
	runOneScenarioHook = func(ctx context.Context, opts RunOptions, manifests map[string]*sourceModulePackage, scenario string) error {
		return wantErr
	}

	err := RunModule(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected scenario error, got %v", err)
	}
}

func TestRunOneScenarioAdditionalBranches(t *testing.T) {
	t.Run("meta module installs task and auth", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunPlaywright := runPlaywrightHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runPlaywrightHook = oldRunPlaywright
		}()

		installed := []string{}
		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
			installed = append(installed, moduleName)
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
		waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
		runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "meta", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		manifests := map[string]*sourceModulePackage{
			"meta": {DirName: "meta", E2E: &packageE2E{Specs: "e2e"}},
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		err := runOneScenario(context.Background(), RunOptions{Module: "meta", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario(meta) error: %v", err)
		}
		if !reflect.DeepEqual(installed, []string{"meta", "task", "auth", "auth"}) {
			t.Fatalf("unexpected install order/modules: %#v", installed)
		}
	})

	t.Run("invalid specs rel is rejected", func(t *testing.T) {
		modulesPath := t.TempDir()
		manifests := map[string]*sourceModulePackage{
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "../outside"}},
		}
		err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if err == nil || !strings.Contains(err.Error(), "invalid package.json choysum.e2e.specs") {
			t.Fatalf("expected invalid specs error, got %v", err)
		}
	})

	t.Run("fixture and seed errors are propagated", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunPlaywright := runPlaywrightHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runPlaywrightHook = oldRunPlaywright
		}()

		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error { return nil }
		startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
			return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
		}
		stopServerHook = func(cmd *exec.Cmd) {}
		waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
		runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}
		manifests := map[string]*sourceModulePackage{
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		fixtureErr := errors.New("fixture failed")
		applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
			return fixtureErr
		}
		seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
			return nil
		}
		err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if !errors.Is(err, fixtureErr) {
			t.Fatalf("expected fixture error, got %v", err)
		}

		applyScenarioFixturesHook = func(ctx context.Context, configPath string, closure []string, manifests map[string]*sourceModulePackage, scenario string, targetModule string, verbose bool, stderr io.Writer, loadedFixtures *[]string) error {
			return nil
		}
		seedErr := errors.New("seed failed")
		seedModuleIndexHook = func(ctx context.Context, configPath string, manifests map[string]*sourceModulePackage) error {
			return seedErr
		}
		err = runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if !errors.Is(err, seedErr) {
			t.Fatalf("expected seed error, got %v", err)
		}
	})

	t.Run("keep mode prints run dir", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunPlaywright := runPlaywrightHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runPlaywrightHook = oldRunPlaywright
		}()

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
		runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}
		manifests := map[string]*sourceModulePackage{
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		var stderr strings.Builder
		err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Keep: true, Stderr: &stderr}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario keep error: %v", err)
		}
		out := stderr.String()
		if !strings.Contains(out, "kept run dir") {
			t.Fatalf("expected keep output, got %q", out)
		}
		if strings.Contains(out, string(filepath.Separator)+"by-module"+string(filepath.Separator)) {
			t.Fatalf("expected run dir without by-module segment, got %q", out)
		}
		if !strings.Contains(out, string(filepath.Separator)+"e2e"+string(filepath.Separator)+"auth"+string(filepath.Separator)) {
			t.Fatalf("expected module segment directly under e2e root, got %q", out)
		}
	})

	t.Run("runtime log level is written to temp config", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunPlaywright := runPlaywrightHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runPlaywrightHook = oldRunPlaywright
		}()

		var seenConfig string
		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
			if seenConfig == "" {
				raw, err := os.ReadFile(configPath)
				if err != nil {
					return err
				}
				seenConfig = string(raw)
			}
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
		waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
		runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}
		manifests := map[string]*sourceModulePackage{
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, RuntimeLogLevel: "info", Stderr: io.Discard}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario error: %v", err)
		}
		if !strings.Contains(seenConfig, "log:\n  level: \"info\"") {
			t.Fatalf("expected temp config to contain info runtime log level, got %q", seenConfig)
		}
	})

	t.Run("prints runtime preparation progress lines", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunPlaywright := runPlaywrightHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runPlaywrightHook = oldRunPlaywright
		}()

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
		waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
		runPlaywrightHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("test"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}
		manifests := map[string]*sourceModulePackage{
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		var stderr strings.Builder
		err := runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: &stderr}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario error: %v", err)
		}
		out := stderr.String()
		if !strings.Contains(out, "# prepare runtime auth\n") {
			t.Fatalf("expected prepare start line, got %q", out)
		}
		if !strings.Contains(out, "# prepare runtime auth ok (") {
			t.Fatalf("expected prepare completion line, got %q", out)
		}
	})
}
