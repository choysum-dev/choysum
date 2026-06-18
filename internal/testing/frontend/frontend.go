// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	xfmt "golang.org/x/exp/errors/fmt"
)

type vitestCoverageSummary struct {
	Total struct {
		Lines struct {
			Pct float64 `json:"pct"`
		} `json:"lines"`
		Functions struct {
			Pct float64 `json:"pct"`
		} `json:"functions"`
		Branches struct {
			Pct float64 `json:"pct"`
		} `json:"branches"`
		Statements struct {
			Pct float64 `json:"pct"`
		} `json:"statements"`
	} `json:"total"`
}

func RunOneAppFrontendTests(
	ctx context.Context,
	repoRoot string,
	app string,
	junitPath string,
	pattern string,
	coverage bool,
	coverageReport bool,
	coverageCheck bool,
	feCoverageAll bool,
	coverageReportDir string,
	coverageLines int,
	coverageFunctions int,
	coverageBranches int,
	coverageStatements int,
	tmpRoot string,
	keep bool,
) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}

	app = strings.TrimSpace(app)
	if app == "" {
		return true, xfmt.Errorf("vitest: missing app")
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		wd, _ := os.Getwd()
		repoRoot = wd
	}
	if strings.TrimSpace(repoRoot) == "" {
		return true, xfmt.Errorf("vitest: cannot determine repo root")
	}

	if _, err := exec.LookPath("npx"); err != nil {
		return true, xfmt.Errorf("vitest: npx not found. Install Node.js from https://nodejs.org")
	}
	if _, err := exec.LookPath("vitest"); err != nil {
		return true, xfmt.Errorf("vitest: vitest is not installed. Run: npm install -g vitest")
	}
	junitPath = strings.TrimSpace(junitPath)
	if junitPath != "" {
		junitDir := filepath.Dir(junitPath)
		if junitDir != "" && junitDir != "." {
			if err := os.MkdirAll(junitDir, 0o755); err != nil {
				return true, xfmt.Errorf("vitest: create junit dir: %w", err)
			}
		}
		junitPath = filepath.ToSlash(junitPath)
	}

	workspaceTmpDir, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, repoRoot, tmpRoot, "frontend")
	if err != nil {
		return true, xfmt.Errorf("vitest: resolve tmp dir: %w", err)
	}
	vitestTmpDir := filepath.Join(workspaceTmpDir, "vitest", sanitizeFrontendAppToken(app))
	if err := os.MkdirAll(vitestTmpDir, 0o755); err != nil {
		return true, xfmt.Errorf("vitest: create tmp dir: %w", err)
	}

	configPath := filepath.Join(vitestTmpDir, fmt.Sprintf("%s.%d.vitest.config.ts", app, time.Now().UnixNano()))
	cleanup := func() { _ = os.Remove(configPath) }
	if !keep {
		defer cleanup()
	}

	reportsDir := filepath.ToSlash(filepath.Join(coverageReportDir, "fe", app))
	includeGlob := filepath.ToSlash(filepath.Join("modules", app, "web", "**", "*.{test,spec}.{ts,tsx}"))
	coverageIncludeGlob := filepath.ToSlash(filepath.Join("modules", app, "web", "**", "*.{ts,tsx,vue}"))

	var b strings.Builder
	b.WriteString("import { defineConfig } from 'vitest/config'\n")
	b.WriteString("import vue from '@vitejs/plugin-vue'\n\n")
	b.WriteString("import path from 'node:path'\n")
	b.WriteString("export default defineConfig({\n")
	b.WriteString("  plugins: [vue()],\n")
	b.WriteString("  resolve: {\n")
	b.WriteString("    alias: {\n")
	b.WriteString("      '@': path.resolve(process.cwd(), 'modules'),\n")
	b.WriteString("    },\n")
	b.WriteString("  },\n")
	b.WriteString("  test: {\n")
	b.WriteString("    include: ['" + includeGlob + "'],\n")
	b.WriteString("    environment: 'node',\n")
	b.WriteString("    passWithNoTests: true,\n")
	if junitPath != "" {
		b.WriteString("    reporters: ['default', 'junit'],\n")
		b.WriteString("    outputFile: {\n")
		b.WriteString("      junit: " + strconv.Quote(junitPath) + ",\n")
		b.WriteString("    },\n")
	}
	if coverage {
		b.WriteString("    coverage: {\n")
		b.WriteString("      provider: 'v8',\n")
		b.WriteString(fmt.Sprintf("      all: %t,\n", feCoverageAll))
		b.WriteString("      include: ['" + coverageIncludeGlob + "'],\n")
		b.WriteString("      exclude: [\n")
		b.WriteString("        '**/node_modules/**',\n")
		b.WriteString("        '**/dist/**',\n")
		b.WriteString("        '**/pb/**',\n")
		b.WriteString("        '**/*.d.ts',\n")
		b.WriteString("        '**/*.{test,spec}.{ts,tsx}',\n")
		b.WriteString("      ],\n")
		b.WriteString("      excludeAfterRemap: true,\n")
		b.WriteString("      reportsDirectory: '" + reportsDir + "',\n")
		b.WriteString("      reporter: [\n")
		b.WriteString("        'json-summary',\n")
		b.WriteString("        'text',\n")
		if coverageReport {
			b.WriteString("        'lcovonly',\n")
			b.WriteString("        'html',\n")
		}
		b.WriteString("      ],\n")
		b.WriteString("    },\n")
	}
	b.WriteString("  },\n")
	b.WriteString("})\n")

	if err := os.WriteFile(configPath, []byte(b.String()), 0o644); err != nil {
		return true, xfmt.Errorf("vitest: write tmp config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "# vitest %s\n", app)
	args := []string{"--no-install", "vitest", "run", "--config", configPath}
	if coverage {
		args = append(args, "--coverage")
	}
	if strings.TrimSpace(pattern) != "" {
		args = append(args, "-t", pattern)
	}

	c := exec.CommandContext(ctx, "npx", args...)
	c.Dir = repoRoot
	c.Env = append(os.Environ(), "NODE_PATH="+filepath.Join(repoRoot, "node_modules"))
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return true, xfmt.Errorf("vitest failed for %s: %w", app, err)
	}

	if coverage && coverageCheck {
		summaryPath := filepath.Join(repoRoot, coverageReportDir, "fe", app, "coverage-summary.json")
		raw, err := os.ReadFile(summaryPath)
		if err != nil {
			return true, xfmt.Errorf("vitest: read coverage summary (%s): %w", summaryPath, err)
		}
		var sum vitestCoverageSummary
		if err := json.Unmarshal(raw, &sum); err != nil {
			return true, xfmt.Errorf("vitest: parse coverage summary: %w", err)
		}
		below := func(pct float64, threshold int) bool {
			if threshold <= 0 {
				return false
			}
			return pct+1e-9 < float64(threshold)
		}
		if below(sum.Total.Lines.Pct, coverageLines) ||
			below(sum.Total.Functions.Pct, coverageFunctions) ||
			below(sum.Total.Branches.Pct, coverageBranches) ||
			below(sum.Total.Statements.Pct, coverageStatements) {
			return true, xfmt.Errorf(
				"vitest coverage check failed for %s (lines=%.2f%% functions=%.2f%% branches=%.2f%% statements=%.2f%%)",
				app,
				sum.Total.Lines.Pct,
				sum.Total.Functions.Pct,
				sum.Total.Branches.Pct,
				sum.Total.Statements.Pct,
			)
		}
	}

	fmt.Fprintf(os.Stderr, "# vitest %s ok\n", app)
	return false, nil
}

func sanitizeFrontendAppToken(app string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token := strings.TrimSpace(replacer.Replace(app))
	if token == "" {
		return "app"
	}
	return token
}
