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
	oldRunE2EHost := runE2EHostHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runE2EHostHook = oldRunE2EHost
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
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		if _, err := os.Stat(runtimePath); err != nil {
			t.Fatalf("expected runtime file created before host, err=%v", err)
		}
		return nil
	}

	modulesPath := t.TempDir()
	specsDir := filepath.Join(modulesPath, "auth", "e2e")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("mkdir specs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
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
	if installCalls != 1 {
		t.Fatalf("expected single target install (auth), got %d", installCalls)
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
	oldRunE2EHost := runE2EHostHook
	defer func() {
		installForE2EHook = oldInstall
		applyScenarioFixturesHook = oldApply
		seedModuleIndexHook = oldSeed
		startServerHook = oldStart
		stopServerHook = oldStop
		waitForHTTP200Hook = oldWait
		runE2EHostHook = oldRunE2EHost
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
	if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
		t.Fatalf("write spec file: %v", err)
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
	hostErr := errors.New("host failed")
	runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
		return hostErr
	}
	err = runOneScenario(context.Background(), RunOptions{Module: "auth", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
	if !errors.Is(err, hostErr) {
		t.Fatalf("expected host error, got %v", err)
	}
}

func TestRunModuleUsesScenarioHook(t *testing.T) {
	setE2ETestChromiumPath(t)
	oldRunOne := runOneScenarioHook
	defer func() { runOneScenarioHook = oldRunOne }()

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	writeQJSSpec(t, filepath.Join(modulesPath, "auth", "e2e", "ok.spec.ts"))

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
	setE2ETestChromiumPath(t)
	oldRunOne := runOneScenarioHook
	defer func() { runOneScenarioHook = oldRunOne }()

	modulesPath := t.TempDir()
	writePackageFile(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth","e2e":{"specs":"e2e"}}}`)
	writeQJSSpec(t, filepath.Join(modulesPath, "auth", "e2e", "ok.spec.ts"))

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
	t.Run("closure without auth does not enable or install auth", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
		}()

		installed := []string{}
		var seenConfig string
		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
			if seenConfig == "" {
				raw, err := os.ReadFile(configPath)
				if err != nil {
					return err
				}
				seenConfig = string(raw)
			}
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "meta", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		manifests := map[string]*sourceModulePackage{
			"meta": {DirName: "meta", Depends: []string{"task"}, E2E: &packageE2E{Specs: "e2e"}},
			"task": {DirName: "task", E2E: &packageE2E{Specs: "e2e"}},
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		err := runOneScenario(context.Background(), RunOptions{Module: "meta", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario(meta) error: %v", err)
		}
		if !reflect.DeepEqual(installed, []string{"meta"}) {
			t.Fatalf("unexpected install order/modules: %#v", installed)
		}
		if !strings.Contains(seenConfig, "auth:\n  enabled: false\n") {
			t.Fatalf("expected auth.enabled false when auth is absent from depends closure, got %q", seenConfig)
		}
		if !strings.Contains(seenConfig, "CHOYSUM_E2E_SKIP_INDEX_STALE_SYNC: \"true\"") {
			t.Fatalf("expected global skip index stale sync, got %q", seenConfig)
		}
	})

	t.Run("closure with auth enables auth without extra installs", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
		}()

		installed := []string{}
		var seenConfig string
		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
			if seenConfig == "" {
				raw, err := os.ReadFile(configPath)
				if err != nil {
					return err
				}
				seenConfig = string(raw)
			}
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "web", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		manifests := map[string]*sourceModulePackage{
			"web":  {DirName: "web", Depends: []string{"auth"}, E2E: &packageE2E{Specs: "e2e"}},
			"auth": {DirName: "auth", E2E: &packageE2E{Specs: "e2e"}},
		}

		err := runOneScenario(context.Background(), RunOptions{Module: "web", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "default")
		if err != nil {
			t.Fatalf("runOneScenario(web) error: %v", err)
		}
		if !reflect.DeepEqual(installed, []string{"web"}) {
			t.Fatalf("unexpected install order/modules: %#v", installed)
		}
		if !strings.Contains(seenConfig, "auth:\n  enabled: true\n") {
			t.Fatalf("expected auth.enabled true when auth is in depends closure, got %q", seenConfig)
		}
	})

	t.Run("scenario backendEnv comes from package.json not scenario name", func(t *testing.T) {
		oldInstall := installForE2EHook
		oldApply := applyScenarioFixturesHook
		oldSeed := seedModuleIndexHook
		oldStart := startServerHook
		oldStop := stopServerHook
		oldWait := waitForHTTP200Hook
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "web", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		manifests := map[string]*sourceModulePackage{
			"web": {
				DirName: "web",
				Depends: []string{"meta"},
				E2E: &packageE2E{
					Specs: "e2e",
					Scenarios: map[string]packageScene{
						"lock-conflict": {Extends: "default"},
						"default":       {},
					},
				},
			},
			"meta": {
				DirName: "meta",
				E2E: &packageE2E{
					Scenarios: map[string]packageScene{
						"lock-conflict": {
							BackendEnv: map[string]string{"CHOYSUM_E2E_FORCE_LOCK_CONFLICT": "true"},
						},
					},
				},
			},
		}

		err := runOneScenario(context.Background(), RunOptions{Module: "web", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifests, "lock-conflict")
		if err != nil {
			t.Fatalf("runOneScenario(web lock-conflict) error: %v", err)
		}
		if !strings.Contains(seenConfig, "CHOYSUM_E2E_FORCE_LOCK_CONFLICT: \"true\"") {
			t.Fatalf("expected FORCE_LOCK from meta scenario backendEnv, got %q", seenConfig)
		}
		if strings.Contains(seenConfig, "CHOYSUM_E2E_FORCE_RELOAD_FAILED") || strings.Contains(seenConfig, "CHOYSUM_E2E_FORCE_RESULT_STATUS") {
			t.Fatalf("unexpected other FORCE_* keys from hard-coded scenario switch, got %q", seenConfig)
		}

		seenConfig = ""
		manifestsNoEnv := map[string]*sourceModulePackage{
			"web": {
				DirName: "web",
				E2E: &packageE2E{
					Specs: "e2e",
					Scenarios: map[string]packageScene{
						"lock-conflict": {},
					},
				},
			},
		}
		err = runOneScenario(context.Background(), RunOptions{Module: "web", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifestsNoEnv, "lock-conflict")
		if err != nil {
			t.Fatalf("runOneScenario(web lock-conflict no env) error: %v", err)
		}
		if strings.Contains(seenConfig, "CHOYSUM_E2E_FORCE_LOCK_CONFLICT") {
			t.Fatalf("scenario name alone must not inject FORCE_LOCK, got %q", seenConfig)
		}

		seenConfig = ""
		manifestsOverride := map[string]*sourceModulePackage{
			"web": {
				DirName: "web",
				Depends: []string{"meta"},
				E2E: &packageE2E{
					Specs: "e2e",
					Scenarios: map[string]packageScene{
						"lock-conflict": {
							BackendEnv: map[string]string{"CHOYSUM_E2E_FORCE_LOCK_CONFLICT": "from-web"},
						},
					},
				},
			},
			"meta": {
				DirName: "meta",
				E2E: &packageE2E{
					Scenarios: map[string]packageScene{
						"lock-conflict": {
							BackendEnv: map[string]string{"CHOYSUM_E2E_FORCE_LOCK_CONFLICT": "from-meta"},
						},
					},
				},
			},
		}
		err = runOneScenario(context.Background(), RunOptions{Module: "web", ModulesPath: modulesPath, WorkDir: t.TempDir(), TmpPath: t.TempDir(), StartupTimeout: time.Second, Stderr: io.Discard}, manifestsOverride, "lock-conflict")
		if err != nil {
			t.Fatalf("runOneScenario(web lock-conflict override) error: %v", err)
		}
		if !strings.Contains(seenConfig, "CHOYSUM_E2E_FORCE_LOCK_CONFLICT: \"from-web\"") {
			t.Fatalf("expected target backendEnv to override dependency, got %q", seenConfig)
		}
		if strings.Contains(seenConfig, "from-meta") {
			t.Fatalf("dependency backendEnv must not win over target, got %q", seenConfig)
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
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
		}()

		installForE2EHook = func(ctx context.Context, configPath string, moduleName string, withDemo bool) error { return nil }
		startServerHook = func(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
			return &exec.Cmd{Process: &os.Process{Pid: 12345}}, nil
		}
		stopServerHook = func(cmd *exec.Cmd) {}
		waitForHTTP200Hook = func(ctx context.Context, url string, timeout time.Duration) error { return nil }
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
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
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
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
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
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
		oldRunE2EHost := runE2EHostHook
		defer func() {
			installForE2EHook = oldInstall
			applyScenarioFixturesHook = oldApply
			seedModuleIndexHook = oldSeed
			startServerHook = oldStart
			stopServerHook = oldStop
			waitForHTTP200Hook = oldWait
			runE2EHostHook = oldRunE2EHost
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
		runE2EHostHook = func(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, onlyFiles []string) error {
			return nil
		}

		modulesPath := t.TempDir()
		specsDir := filepath.Join(modulesPath, "auth", "e2e")
		if err := os.MkdirAll(specsDir, 0o755); err != nil {
			t.Fatalf("mkdir specs dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(specsDir, "sample.spec.ts"), []byte("import { test } from '@choysum/e2e';\ntest('sample', async () => {});\n"), 0o644); err != nil {
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
		if !strings.Contains(out, "# e2e-qjs auth (") {
			t.Fatalf("expected e2e-qjs progress line, got %q", out)
		}
		if strings.Contains(out, "# e2e-playwright") {
			t.Fatalf("unexpected playwright progress line, got %q", out)
		}
	})
}
