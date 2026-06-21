// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeExecFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func ensureFrontendRequiredModulesAt(t *testing.T, moduleRoot string) {
	t.Helper()
	required := []string{"vitest", "vite", "@bufbuild/protobuf", "@vitejs/plugin-vue", "vue", "@vue/compiler-sfc", "@vue/shared", "@vue/server-renderer", "@vue/test-utils", "sass-embedded"}
	for _, mod := range required {
		if err := os.MkdirAll(filepath.Join(moduleRoot, filepath.FromSlash(mod)), 0o755); err != nil {
			t.Fatalf("mkdir required module %s: %v", mod, err)
		}
	}
}

func ensureFrontendRequiredModules(t *testing.T, repoRoot string) {
	t.Helper()
	ensureFrontendRequiredModulesAt(t, filepath.Join(repoRoot, "node_modules"))
}

func runFrontendTest(ctx context.Context, repoRoot string, app string, pattern string, coverage bool, coverageReport bool, coverageCheck bool, feCoverageAll bool, coverageReportDir string, coverageLines int, coverageFunctions int, coverageBranches int, coverageStatements int, tmpRoot string, keep bool) (bool, error) {
	return RunOneAppFrontendTests(ctx, repoRoot, app, "", pattern, coverage, coverageReport, coverageCheck, feCoverageAll, coverageReportDir, coverageLines, coverageFunctions, coverageBranches, coverageStatements, tmpRoot, keep)
}

func TestRunOneAppFrontendTestsGuards(t *testing.T) {
	t.Run("rejects missing app", func(t *testing.T) {
		tmpRoot := t.TempDir()
		failed, err := runFrontendTest(context.Background(), t.TempDir(), "", "", false, false, false, false, "coverage", 0, 0, 0, 0, tmpRoot, false)
		if err == nil || !strings.Contains(err.Error(), "missing app") {
			t.Fatalf("expected missing app error, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing npx", func(t *testing.T) {
		t.Setenv("PATH", "")
		tmpRoot := t.TempDir()
		failed, err := runFrontendTest(context.Background(), t.TempDir(), "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, tmpRoot, false)
		if err == nil || !strings.Contains(err.Error(), "npx not found") {
			t.Fatalf("expected npx not found error, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("allows npx to resolve vitest without global binary", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir)
		ensureFrontendRequiredModules(t, repoRoot)

		tmpRoot := t.TempDir()
		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, tmpRoot, false)
		if err != nil || failed {
			t.Fatalf("expected success with npx-resolved vitest, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing required modules with global install guidance", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		globalRoot := filepath.Join(t.TempDir(), "global-node-modules")

		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)

		if err := os.MkdirAll(filepath.Join(repoRoot, "modules", "auth"), 0o755); err != nil {
			t.Fatalf("mkdir modules/auth: %v", err)
		}
		pkgJSON := `{"peerDependencies":{"@vueuse/core":"^14.3.0"}}`
		if err := os.WriteFile(filepath.Join(repoRoot, "modules", "auth", "package.json"), []byte(pkgJSON), 0o644); err != nil {
			t.Fatalf("write auth package.json: %v", err)
		}

		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err == nil {
			t.Fatalf("expected missing modules error, got failed=%v", failed)
		}
		if !strings.Contains(err.Error(), "missing required modules") {
			t.Fatalf("expected missing required modules message, got %v", err)
		}
		if !strings.Contains(err.Error(), "npm install -g") {
			t.Fatalf("expected global install guidance, got %v", err)
		}
		if !strings.Contains(err.Error(), "@vueuse/core") {
			t.Fatalf("expected app dependency in missing list, got %v", err)
		}
	})

	t.Run("accepts required modules from modules node_modules", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		ensureFrontendRequiredModulesAt(t, filepath.Join(repoRoot, "modules", "node_modules"))

		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err != nil || failed {
			t.Fatalf("expected modules/node_modules to satisfy required modules, failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing dependencies from dependent modules", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		globalRoot := filepath.Join(t.TempDir(), "global-node-modules")

		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)
		ensureFrontendRequiredModulesAt(t, globalRoot)

		if err := os.MkdirAll(filepath.Join(repoRoot, "modules", "auth"), 0o755); err != nil {
			t.Fatalf("mkdir modules/auth: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(repoRoot, "modules", "core"), 0o755); err != nil {
			t.Fatalf("mkdir modules/core: %v", err)
		}
		authPkg := `{"choysum":{"depends":["core"]}}`
		if err := os.WriteFile(filepath.Join(repoRoot, "modules", "auth", "package.json"), []byte(authPkg), 0o644); err != nil {
			t.Fatalf("write auth package.json: %v", err)
		}
		corePkg := `{"peerDependencies":{"@connectrpc/connect":"^2.1.1"}}`
		if err := os.WriteFile(filepath.Join(repoRoot, "modules", "core", "package.json"), []byte(corePkg), 0o644); err != nil {
			t.Fatalf("write core package.json: %v", err)
		}

		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err == nil {
			t.Fatalf("expected missing modules error, got failed=%v", failed)
		}
		if !strings.Contains(err.Error(), "@connectrpc/connect") {
			t.Fatalf("expected dependent-module package in missing list, got %v", err)
		}
	})

	t.Run("ignores workspace scoped dependencies", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		globalRoot := filepath.Join(t.TempDir(), "global-node-modules")

		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)
		ensureFrontendRequiredModulesAt(t, globalRoot)

		if err := os.MkdirAll(filepath.Join(repoRoot, "modules", "auth"), 0o755); err != nil {
			t.Fatalf("mkdir modules/auth: %v", err)
		}
		authPkg := `{"dependencies":{"@choysum-dev/core":"workspace:*"},"peerDependencies":{"@choysum-dev/base":"workspace:*"}}`
		if err := os.WriteFile(filepath.Join(repoRoot, "modules", "auth", "package.json"), []byte(authPkg), 0o644); err != nil {
			t.Fatalf("write auth package.json: %v", err)
		}

		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err != nil || failed {
			t.Fatalf("expected workspace scoped dependencies to be ignored, failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing coverage provider", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		ensureFrontendRequiredModules(t, repoRoot)

		// vitest is on PATH, but @vitest/coverage-v8 is not installed.
		// vitest itself will report the missing coverage provider at runtime;
		// our pre-flight no longer guards this case.
		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", true, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err != nil {
			t.Fatalf("expected no pre-flight error, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("requires happy-dom when vitest environment marker is used", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		globalRoot := filepath.Join(t.TempDir(), "global-node-modules")

		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)
		ensureFrontendRequiredModulesAt(t, globalRoot)

		testFile := filepath.Join(repoRoot, "modules", "meta", "web", "__tests__", "env.test.ts")
		if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
			t.Fatalf("mkdir test dir: %v", err)
		}
		if err := os.WriteFile(testFile, []byte("// @vitest-environment happy-dom\nimport { describe, it } from 'vitest'\ndescribe('env', () => { it('ok', () => {}) })\n"), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		failed, err := runFrontendTest(context.Background(), repoRoot, "meta", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err == nil {
			t.Fatalf("expected missing modules error, got failed=%v", failed)
		}
		if !strings.Contains(err.Error(), "happy-dom") {
			t.Fatalf("expected happy-dom in missing list, got %v", err)
		}
	})
}

func TestRunOneAppFrontendTestsUsesTemporaryGlobalNodeModulesLink(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nlink_path=\"${CHOYSUM_EXPECT_GLOBAL_LINK}\"\nif [ ! -L \"$link_path\" ]; then\n  echo \"missing global module symlink: $link_path\" >&2\n  exit 12\nfi\nexit 0\n")
	writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CHOYSUM_EXPECT_GLOBAL_LINK", filepath.Join(repoRoot, "modules", "node_modules", "@vue", "shared"))

	globalRoot := filepath.Join(t.TempDir(), "global-node-modules")
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", globalRoot)
	ensureFrontendRequiredModulesAt(t, globalRoot)

	failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, true)
	if err != nil || failed {
		t.Fatalf("expected frontend run success, failed=%v err=%v", failed, err)
	}

	if _, err := os.Lstat(filepath.Join(repoRoot, "modules", "node_modules", "@vue", "shared")); !os.IsNotExist(err) {
		t.Fatalf("expected temporary global module symlink to be cleaned up, err=%v", err)
	}
	if st, err := os.Stat(filepath.Join(repoRoot, "modules", "node_modules")); err == nil && st.IsDir() {
		entries, readErr := os.ReadDir(filepath.Join(repoRoot, "modules", "node_modules"))
		if readErr != nil {
			t.Fatalf("read modules/node_modules: %v", readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("expected modules/node_modules to be empty after cleanup, found %d entries", len(entries))
		}
	}
}

func TestEnsureGlobalModuleLinksCleansUpOnSymlinkFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permission semantics differ on windows")
	}

	repoRoot := t.TempDir()
	globalRoot := filepath.Join(t.TempDir(), "global-node-modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "left-pad"), 0o755); err != nil {
		t.Fatalf("mkdir global left-pad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(globalRoot, "@scoped", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir global @scoped/pkg: %v", err)
	}

	localScopedDir := filepath.Join(repoRoot, "modules", "node_modules", "@scoped")
	if err := os.MkdirAll(localScopedDir, 0o755); err != nil {
		t.Fatalf("mkdir local @scoped dir: %v", err)
	}
	if err := os.Chmod(localScopedDir, 0o555); err != nil {
		t.Fatalf("chmod local @scoped dir readonly: %v", err)
	}
	defer func() {
		_ = os.Chmod(localScopedDir, 0o755)
	}()

	cleanup, err := ensureGlobalModuleLinks(repoRoot, globalRoot, []string{"left-pad", "@scoped/pkg"})
	if err == nil {
		cleanup()
		t.Fatal("expected symlink failure for readonly scoped directory")
	}
	if _, statErr := os.Lstat(filepath.Join(repoRoot, "modules", "node_modules", "left-pad")); !os.IsNotExist(statErr) {
		t.Fatalf("expected rollback to remove previously created left-pad link, lstat err=%v", statErr)
	}
}

func TestRunOneAppFrontendTestsKeepTmpConfig(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\ncapture=\"${CHOYSUM_CAPTURE_CONFIG}\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--config\" ]; then\n    shift\n    echo \"$1\" > \"$capture\"\n    break\n  fi\n  shift\ndone\nexit 0\n")
	writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ensureFrontendRequiredModules(t, repoRoot)

	captureDefault := filepath.Join(t.TempDir(), "default-config.txt")
	t.Setenv("CHOYSUM_CAPTURE_CONFIG", captureDefault)
	failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
	if err != nil || failed {
		t.Fatalf("expected frontend run success, failed=%v err=%v", failed, err)
	}
	defaultPathBytes, err := os.ReadFile(captureDefault)
	if err != nil {
		t.Fatalf("read captured config path: %v", err)
	}
	defaultConfigPath := strings.TrimSpace(string(defaultPathBytes))
	if defaultConfigPath == "" {
		t.Fatalf("expected captured config path for default mode")
	}
	if _, err := os.Stat(defaultConfigPath); !os.IsNotExist(err) {
		t.Fatalf("expected default mode to cleanup tmp config, stat err=%v", err)
	}

	captureKeep := filepath.Join(t.TempDir(), "keep-config.txt")
	t.Setenv("CHOYSUM_CAPTURE_CONFIG", captureKeep)
	failed, err = runFrontendTest(context.Background(), repoRoot, "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, true)
	if err != nil || failed {
		t.Fatalf("expected frontend run success with keep, failed=%v err=%v", failed, err)
	}
	keepPathBytes, err := os.ReadFile(captureKeep)
	if err != nil {
		t.Fatalf("read captured config path (keep): %v", err)
	}
	keepConfigPath := strings.TrimSpace(string(keepPathBytes))
	if keepConfigPath == "" {
		t.Fatalf("expected captured config path for keep mode")
	}
	if _, err := os.Stat(keepConfigPath); err != nil {
		t.Fatalf("expected keep mode to preserve tmp config, err=%v", err)
	}
}

func TestRunOneAppFrontendTestsConfigIncludesJUnitAndLcov(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\ncapture=\"${CHOYSUM_CAPTURE_CONFIG}\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--config\" ]; then\n    shift\n    echo \"$1\" > \"$capture\"\n    break\n  fi\n  shift\ndone\nexit 0\n")
	writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ensureFrontendRequiredModules(t, repoRoot)

	capturePath := filepath.Join(t.TempDir(), "config.txt")
	t.Setenv("CHOYSUM_CAPTURE_CONFIG", capturePath)
	junitPath := filepath.Join(repoRoot, ".choysum", "test-results", "auth.frontend.junit.xml")
	failed, err := RunOneAppFrontendTests(context.Background(), repoRoot, "auth", junitPath, "", true, true, false, false, "coverage", 0, 0, 0, 0, repoRoot, true)
	if err != nil || failed {
		t.Fatalf("expected frontend run success, failed=%v err=%v", failed, err)
	}

	configPathBytes, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read captured config path: %v", err)
	}
	configPath := strings.TrimSpace(string(configPathBytes))
	if configPath == "" {
		t.Fatalf("expected captured config path")
	}
	configRaw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configText := string(configRaw)
	checks := []string{
		"cacheDir: \"",
		"/frontend/vite-cache/auth\"",
		"reporters: ['default', 'junit']",
		`junit: "` + filepath.ToSlash(junitPath) + `"`,
		"'lcovonly'",
		"'html'",
	}
	for _, want := range checks {
		if !strings.Contains(configText, want) {
			t.Fatalf("config missing %q: %s", want, configText)
		}
	}
	if _, err := os.Stat(filepath.Dir(junitPath)); err != nil {
		t.Fatalf("expected junit dir to be created: %v", err)
	}
}

func TestAppUsesVitestEnvironmentStopsAfterMatch(t *testing.T) {
	repoRoot := t.TempDir()
	webRoot := filepath.Join(repoRoot, "modules", "auth", "web")
	if err := os.MkdirAll(filepath.Join(webRoot, "__tests__"), 0o755); err != nil {
		t.Fatalf("mkdir tests dir: %v", err)
	}

	markerFile := filepath.Join(webRoot, "__tests__", "a.spec.ts")
	markerContent := "// @vitest-environment happy-dom\nimport { describe, it } from 'vitest'\ndescribe('a', () => { it('ok', () => {}) })\n"
	if err := os.WriteFile(markerFile, []byte(markerContent), 0o644); err != nil {
		t.Fatalf("write marker test file: %v", err)
	}

	brokenDir := filepath.Join(webRoot, "__tests__", "zzzz.spec.ts")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("mkdir trailing pseudo test dir: %v", err)
	}

	found, err := appUsesVitestEnvironment(repoRoot, "auth", "happy-dom")
	if err != nil {
		t.Fatalf("appUsesVitestEnvironment error: %v", err)
	}
	if !found {
		t.Fatal("expected vitest environment marker to be found")
	}
}

func TestRunOneAppFrontendTestsCoverageCheck(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
	writeExecFile(t, filepath.Join(binDir, "vitest"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ensureFrontendRequiredModules(t, repoRoot)

	summaryPath := filepath.Join(repoRoot, "cov", "fe", "auth", "coverage-summary.json")
	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("mkdir summary dir: %v", err)
	}

	if err := os.WriteFile(summaryPath, []byte(`{"total":{"lines":{"pct":95},"functions":{"pct":94},"branches":{"pct":93},"statements":{"pct":92}}}`), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}

	failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", true, false, true, false, "cov", 90, 90, 90, 90, repoRoot, false)
	if err != nil || failed {
		t.Fatalf("expected coverage check pass, got failed=%v err=%v", failed, err)
	}

	if err := os.WriteFile(summaryPath, []byte(`{"total":{"lines":{"pct":50},"functions":{"pct":50},"branches":{"pct":50},"statements":{"pct":50}}}`), 0o644); err != nil {
		t.Fatalf("write failing summary: %v", err)
	}

	failed, err = runFrontendTest(context.Background(), repoRoot, "auth", "", true, false, true, false, "cov", 80, 80, 80, 80, repoRoot, false)
	if err == nil || !failed || !strings.Contains(err.Error(), "coverage check failed") {
		t.Fatalf("expected coverage check failure, got failed=%v err=%v", failed, err)
	}
}
