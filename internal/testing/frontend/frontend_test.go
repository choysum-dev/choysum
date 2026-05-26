// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
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
		if err == nil || !strings.Contains(err.Error(), "missing npx") {
			t.Fatalf("expected missing npx error, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing vitest binary", func(t *testing.T) {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir)

		tmpRoot := t.TempDir()
		failed, err := runFrontendTest(context.Background(), t.TempDir(), "auth", "", false, false, false, false, "coverage", 0, 0, 0, 0, tmpRoot, false)
		if err == nil || !strings.Contains(err.Error(), "vitest is not installed") {
			t.Fatalf("expected missing vitest error, got failed=%v err=%v", failed, err)
		}
	})

	t.Run("rejects missing coverage provider", func(t *testing.T) {
		repoRoot := t.TempDir()
		binDir := filepath.Join(t.TempDir(), "bin")
		writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir)
		writeExecFile(t, filepath.Join(repoRoot, "node_modules", ".bin", "vitest"), "#!/bin/sh\nexit 0\n")

		failed, err := runFrontendTest(context.Background(), repoRoot, "auth", "", true, false, false, false, "coverage", 0, 0, 0, 0, repoRoot, false)
		if err == nil || !strings.Contains(err.Error(), "coverage-v8") {
			t.Fatalf("expected missing coverage provider error, got failed=%v err=%v", failed, err)
		}
	})
}

func TestRunOneAppFrontendTestsKeepTmpConfig(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\ncapture=\"${CHOYSUM_CAPTURE_CONFIG}\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--config\" ]; then\n    shift\n    echo \"$1\" > \"$capture\"\n    break\n  fi\n  shift\ndone\nexit 0\n")
	t.Setenv("PATH", binDir)
	writeExecFile(t, filepath.Join(repoRoot, "node_modules", ".bin", "vitest"), "#!/bin/sh\nexit 0\n")

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
	t.Setenv("PATH", binDir)
	writeExecFile(t, filepath.Join(repoRoot, "node_modules", ".bin", "vitest"), "#!/bin/sh\nexit 0\n")
	writeExecFile(t, filepath.Join(repoRoot, "node_modules", "@vitest", "coverage-v8", "package.json"), "{}\n")

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

func TestRunOneAppFrontendTestsCoverageCheck(t *testing.T) {
	repoRoot := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	writeExecFile(t, filepath.Join(binDir, "npx"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	writeExecFile(t, filepath.Join(repoRoot, "node_modules", ".bin", "vitest"), "#!/bin/sh\nexit 0\n")
	writeExecFile(t, filepath.Join(repoRoot, "node_modules", "@vitest", "coverage-v8", "package.json"), "{}\n")

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
