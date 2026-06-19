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
	"sort"
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

// ValidateFrontendTestDependencies checks whether required frontend tooling and
// modules are available in local module roots or global npm root.
func ValidateFrontendTestDependencies(repoRoot string, app string) error {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		wd, _ := os.Getwd()
		repoRoot = wd
	}
	if strings.TrimSpace(repoRoot) == "" {
		return xfmt.Errorf("vitest: cannot determine repo root")
	}

	app = strings.TrimSpace(app)
	if app == "" {
		return xfmt.Errorf("vitest: missing app")
	}

	if _, err := exec.LookPath("npx"); err != nil {
		return xfmt.Errorf("vitest: npx not found. Install Node.js from https://nodejs.org")
	}
	if _, err := exec.LookPath("vitest"); err != nil {
		return xfmt.Errorf("vitest: vitest is not installed. Run: npm install -g vitest")
	}

	globalNodeModulesRoot := resolveGlobalNpmRoot()
	requiredModules, err := collectRequiredFrontendModules(repoRoot, app)
	if err != nil {
		return err
	}
	moduleRoots := append(localFrontendModuleRoots(repoRoot), globalNodeModulesRoot)
	missingModules := missingRequiredNodeModules(requiredModules, moduleRoots...)
	if len(missingModules) > 0 {
		return xfmt.Errorf(
			"vitest: missing required modules for %s: %s. Install globally: npm install -g %s",
			app,
			strings.Join(missingModules, ", "),
			strings.Join(missingModules, " "),
		)
	}

	return nil
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

	if err := ValidateFrontendTestDependencies(repoRoot, app); err != nil {
		return true, err
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		wd, _ := os.Getwd()
		repoRoot = wd
	}
	app = strings.TrimSpace(app)
	globalNodeModulesRoot := resolveGlobalNpmRoot()
	requiredModules, err := collectRequiredFrontendModules(repoRoot, app)
	if err != nil {
		return true, err
	}
	localModuleRoots := localFrontendModuleRoots(repoRoot)
	missingLocalModules := missingRequiredNodeModules(requiredModules, localModuleRoots...)
	if len(missingLocalModules) > 0 {
		cleanupGlobalLinks, err := ensureGlobalModuleLinks(repoRoot, globalNodeModulesRoot, missingLocalModules)
		if err != nil {
			return true, err
		}
		defer cleanupGlobalLinks()
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
	nodePathValue := buildNodePath(repoRoot, globalNodeModulesRoot)
	c.Env = replaceOrAppendEnv(os.Environ(), "NODE_PATH", nodePathValue)
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

func collectRequiredFrontendModules(repoRoot string, app string) ([]string, error) {
	required := map[string]struct{}{
		"vitest":               {},
		"vite":                 {},
		"@bufbuild/protobuf":   {},
		"@vitejs/plugin-vue":   {},
		"vue":                  {},
		"@vue/compiler-sfc":    {},
		"@vue/server-renderer": {},
		"@vue/test-utils":      {},
	}

	visitedModules := map[string]struct{}{}
	var collectModuleDeps func(moduleName string) error
	collectModuleDeps = func(moduleName string) error {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			return nil
		}
		if _, seen := visitedModules[moduleName]; seen {
			return nil
		}
		visitedModules[moduleName] = struct{}{}

		modulePkgPath := filepath.Join(repoRoot, "modules", moduleName, "package.json")
		pkg, err := readFrontendModulePackage(modulePkgPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return xfmt.Errorf("vitest: read module package.json for %s: %w", moduleName, err)
		}

		for name := range pkg.Dependencies {
			required[name] = struct{}{}
		}
		for name := range pkg.PeerDependencies {
			required[name] = struct{}{}
		}
		for _, depModule := range pkg.Choysum.Depends {
			if err := collectModuleDeps(depModule); err != nil {
				return err
			}
		}

		return nil
	}

	if err := collectModuleDeps(app); err != nil {
		return nil, err
	}

	usesHappyDOM, err := appUsesVitestEnvironment(repoRoot, app, "happy-dom")
	if err != nil {
		return nil, err
	}
	if usesHappyDOM {
		required["happy-dom"] = struct{}{}
	}

	modules := make([]string, 0, len(required))
	for name := range required {
		if strings.TrimSpace(name) == "" {
			continue
		}
		modules = append(modules, name)
	}
	sort.Strings(modules)
	return modules, nil
}

type frontendModulePackage struct {
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	Choysum          struct {
		Depends []string `json:"depends"`
	} `json:"choysum"`
}

func readFrontendModulePackage(path string) (frontendModulePackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return frontendModulePackage{}, err
	}
	var pkg frontendModulePackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return frontendModulePackage{}, xfmt.Errorf("parse package.json: %w", err)
	}
	if pkg.Dependencies == nil {
		pkg.Dependencies = map[string]string{}
	}
	if pkg.PeerDependencies == nil {
		pkg.PeerDependencies = map[string]string{}
	}
	return pkg, nil
}

func appUsesVitestEnvironment(repoRoot string, app string, environment string) (bool, error) {
	environment = strings.TrimSpace(environment)
	if environment == "" {
		return false, nil
	}

	webRoot := filepath.Join(repoRoot, "modules", app, "web")
	st, err := os.Stat(webRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, xfmt.Errorf("vitest: stat web root for %s: %w", app, err)
	}
	if !st.IsDir() {
		return false, nil
	}

	marker := "@vitest-environment " + environment
	found := false
	err = filepath.Walk(webRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || found {
			return nil
		}

		name := info.Name()
		if !strings.Contains(name, ".test.") && !strings.Contains(name, ".spec.") {
			return nil
		}
		if !strings.HasSuffix(name, ".ts") &&
			!strings.HasSuffix(name, ".tsx") &&
			!strings.HasSuffix(name, ".js") &&
			!strings.HasSuffix(name, ".jsx") &&
			!strings.HasSuffix(name, ".mjs") &&
			!strings.HasSuffix(name, ".cjs") {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), marker) {
			found = true
		}
		return nil
	})
	if err != nil {
		return false, xfmt.Errorf("vitest: scan web tests for %s: %w", app, err)
	}

	return found, nil
}

func localFrontendModuleRoots(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, "node_modules"),
		filepath.Join(repoRoot, "modules", "node_modules"),
	}
}

func ensureGlobalModuleLinks(repoRoot string, globalNodeModulesRoot string, moduleNames []string) (func(), error) {
	noop := func() {}
	globalNodeModulesRoot = strings.TrimSpace(globalNodeModulesRoot)
	if globalNodeModulesRoot == "" {
		return noop, nil
	}
	if st, err := os.Stat(globalNodeModulesRoot); err != nil || !st.IsDir() {
		return noop, nil
	}

	localNodeModulesRoot := filepath.Join(repoRoot, "node_modules")
	if err := os.MkdirAll(localNodeModulesRoot, 0o755); err != nil {
		return nil, xfmt.Errorf("vitest: create local node_modules: %w", err)
	}

	createdLinks := make([]string, 0, len(moduleNames))
	for _, moduleName := range moduleNames {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			continue
		}

		globalModuleDir := filepath.Join(globalNodeModulesRoot, filepath.FromSlash(moduleName))
		if st, err := os.Stat(globalModuleDir); err != nil || !st.IsDir() {
			continue
		}

		localModuleDir := filepath.Join(localNodeModulesRoot, filepath.FromSlash(moduleName))
		if _, err := os.Lstat(localModuleDir); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, xfmt.Errorf("vitest: stat %s: %w", localModuleDir, err)
		}

		if err := os.MkdirAll(filepath.Dir(localModuleDir), 0o755); err != nil {
			return nil, xfmt.Errorf("vitest: prepare %s: %w", localModuleDir, err)
		}
		if err := os.Symlink(globalModuleDir, localModuleDir); err != nil {
			return nil, xfmt.Errorf("vitest: link %s -> %s: %w", localModuleDir, globalModuleDir, err)
		}
		createdLinks = append(createdLinks, localModuleDir)
	}

	cleanup := func() {
		for _, localModuleDir := range createdLinks {
			st, err := os.Lstat(localModuleDir)
			if err != nil || st.Mode()&os.ModeSymlink == 0 {
				continue
			}
			_ = os.Remove(localModuleDir)
		}
	}

	return cleanup, nil
}

func missingRequiredNodeModules(required []string, moduleRoots ...string) []string {
	missing := make([]string, 0)
	for _, moduleName := range required {
		if moduleInstalledInRoots(moduleName, moduleRoots...) {
			continue
		}
		missing = append(missing, moduleName)
	}
	return missing
}

func moduleInstalledInRoots(moduleName string, moduleRoots ...string) bool {
	for _, root := range moduleRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		moduleDir := filepath.Join(root, filepath.FromSlash(moduleName))
		if st, err := os.Stat(moduleDir); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func resolveGlobalNpmRoot() string {
	if override := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); override != "" {
		return override
	}
	out, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func buildNodePath(repoRoot string, globalNodeModulesRoot string) string {
	values := make([]string, 0, 4)
	for _, localNodeModules := range localFrontendModuleRoots(repoRoot) {
		if st, err := os.Stat(localNodeModules); err == nil && st.IsDir() {
			values = append(values, localNodeModules)
		}
	}
	if st, err := os.Stat(globalNodeModulesRoot); err == nil && st.IsDir() {
		values = append(values, globalNodeModulesRoot)
	}
	if current := strings.TrimSpace(os.Getenv("NODE_PATH")); current != "" {
		values = append(values, current)
	}
	return strings.Join(values, string(os.PathListSeparator))
}

func replaceOrAppendEnv(env []string, key string, value string) []string {
	if strings.TrimSpace(key) == "" {
		return env
	}
	needle := key + "="
	replaced := false
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, needle) {
			if !replaced {
				out = append(out, needle+value)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, needle+value)
	}
	return out
}
