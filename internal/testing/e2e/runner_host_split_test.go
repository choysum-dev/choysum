// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
