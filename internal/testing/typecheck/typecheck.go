// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	xfmt "golang.org/x/exp/errors/fmt"
)

type RunOptions struct {
	ModulesPath string
	NpmPath     string
	RepoRoot    string
	TmpPath     string
	Target      string // app name or "all"
	Keep        bool

	Stdout io.Writer
	Stderr io.Writer
}

var errNoTypecheckInputs = errors.New("typecheck: no checkable ts inputs")

var errTypecheckInputFound = errors.New("typecheck: input found")

func Run(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(testingpathing.TestingRunIDFromContext(ctx)) == "" {
		ctx = testingpathing.ContextWithTestingRunID(ctx, testingpathing.NewTestingRunID())
	}
	if strings.TrimSpace(opts.ModulesPath) == "" {
		return xfmt.Errorf("typecheck: modules_path is required")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		wd, _ := os.Getwd()
		opts.RepoRoot = wd
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return xfmt.Errorf("typecheck: cannot determine repo root")
	}
	if strings.TrimSpace(opts.Target) == "" {
		opts.Target = "all"
	}

	apps, err := ResolveApps(opts.ModulesPath, opts.Target)
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		fmt.Fprintln(opts.Stdout, "no tests found")
		return nil
	}
	sort.Strings(apps)

	ranAny := false
	for _, app := range apps {
		if err := TypecheckApp(ctx, opts, app); err != nil {
			if errors.Is(err, errNoTypecheckInputs) {
				continue
			}
			return err
		}
		ranAny = true
	}
	if !ranAny {
		fmt.Fprintln(opts.Stdout, "no tests found")
	}
	return nil
}

func ResolveApps(modulesPath string, target string) ([]string, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil, xfmt.Errorf("typecheck: modules_path is required")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		target = "all"
	}

	if target != "all" {
		appDir := filepath.Join(modulesPath, target)
		st, err := os.Stat(appDir)
		if err != nil || !st.IsDir() {
			return nil, xfmt.Errorf("typecheck: unknown app %q", target)
		}
		hasTargets, err := HasTargets(modulesPath, target)
		if err != nil {
			return nil, err
		}
		if !hasTargets {
			return nil, nil
		}
		return []string{target}, nil
	}

	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, xfmt.Errorf("typecheck: read modules dir: %w", err)
	}

	apps := make([]string, 0)
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if name == ".choysum" || name == "tmp" {
			continue
		}
		hasTargets, err := HasTargets(modulesPath, name)
		if err != nil {
			return nil, err
		}
		if hasTargets {
			apps = append(apps, name)
		}
	}
	return apps, nil
}

func HasTargets(modulesPath string, app string) (bool, error) {
	serviceDir := filepath.Join(modulesPath, app, "service")
	webDir := filepath.Join(modulesPath, app, "web")
	if st, err := os.Stat(serviceDir); err == nil && st.IsDir() {
		return true, nil
	}
	if st, err := os.Stat(webDir); err == nil && st.IsDir() {
		return true, nil
	}
	return false, nil
}

func TypecheckApp(ctx context.Context, opts RunOptions, app string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(opts.ModulesPath) == "" {
		return xfmt.Errorf("typecheck: modules_path is required")
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		wd, _ := os.Getwd()
		opts.RepoRoot = wd
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return xfmt.Errorf("typecheck: cannot determine repo root")
	}
	repoRoot := opts.RepoRoot
	if !filepath.IsAbs(repoRoot) {
		repoRoot, _ = filepath.Abs(repoRoot)
	}
	repoRoot = filepath.Clean(repoRoot)

	modulesRoot := opts.ModulesPath
	if !filepath.IsAbs(modulesRoot) {
		modulesRoot, _ = filepath.Abs(modulesRoot)
	}
	modulesRoot = filepath.Clean(modulesRoot)

	tmpRoot := strings.TrimSpace(opts.TmpPath)
	if tmpRoot == "" {
		tmpRoot = os.TempDir()
	}
	if !filepath.IsAbs(tmpRoot) {
		tmpRoot, _ = filepath.Abs(tmpRoot)
	}
	tmpRoot = filepath.Clean(tmpRoot)
	if strings.TrimSpace(tmpRoot) == "" {
		return xfmt.Errorf("typecheck: cannot determine tmp path")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	app = strings.TrimSpace(app)
	if app == "" {
		return xfmt.Errorf("typecheck: missing app name")
	}
	hasInputs, err := hasTypecheckInputs(modulesRoot, app)
	if err != nil {
		return xfmt.Errorf("typecheck: scan inputs for %s: %w", app, err)
	}
	if !hasInputs {
		return errNoTypecheckInputs
	}

	npxPath, err := resolveNpxPath(opts.NpmPath)
	if err != nil {
		return err
	}

	vueTscBin := filepath.Join(repoRoot, "node_modules", ".bin", "vue-tsc")
	vueTscPkg := filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json")
	if _, err := os.Stat(vueTscPkg); err != nil {
		if _, err2 := os.Stat(vueTscBin); err2 != nil {
			return xfmt.Errorf("typecheck: vue-tsc is not installed. Run `npm install` in repo root (%s)", repoRoot)
		}
	}

	hasWebSources := false
	if st, err := os.Stat(filepath.Join(modulesRoot, app, "web")); err == nil && st.IsDir() {
		hasWebSources = true
	}
	viteClientTypesPath := filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts")
	if hasWebSources {
		if _, err := os.Stat(viteClientTypesPath); err != nil {
			return xfmt.Errorf("typecheck: vite is not installed. Run `npm install` in repo root (%s)", repoRoot)
		}
	}

	tmpTsconfigRoot, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, repoRoot, tmpRoot, "typecheck")
	if err != nil {
		return xfmt.Errorf("typecheck: resolve tmp dir: %w", err)
	}
	tmpTsconfigDir := filepath.Join(tmpTsconfigRoot, sanitizeAppToken(app))
	if err := os.MkdirAll(tmpTsconfigDir, 0o755); err != nil {
		return xfmt.Errorf("typecheck: ensure tmp dir: %w", err)
	}
	tmpTsconfigFile, err := os.CreateTemp(tmpTsconfigDir, fmt.Sprintf("%s-*.tsconfig.json", sanitizeAppToken(app)))
	if err != nil {
		return xfmt.Errorf("typecheck: create tmp tsconfig: %w", err)
	}
	tmpTsconfigPath := tmpTsconfigFile.Name()
	cleanupTmpArtifacts := func() {
		_ = os.Remove(tmpTsconfigPath)
		_ = os.Remove(tmpTsconfigDir)
	}
	if !opts.Keep {
		defer cleanupTmpArtifacts()
	}

	include := []string{
		filepath.ToSlash(filepath.Join(modulesRoot, "**", "*.d.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "service", "**", "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.tsx")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.vue")),
	}
	if hasWebSources {
		include = append(include, filepath.ToSlash(viteClientTypesPath))
	}
	cfg := map[string]any{
		"compilerOptions": map[string]any{
			"target":                       "ES2020",
			"module":                       "ESNext",
			"moduleResolution":             "bundler",
			"lib":                          []string{"ES2020", "DOM", "DOM.Iterable"},
			"strict":                       true,
			"strictPropertyInitialization": false,
			"experimentalDecorators":       true,
			"allowJs":                      true,
			"allowArbitraryExtensions":     true,
			"skipLibCheck":                 true,
			"types":                        []string{"node"},
			"typeRoots":                    []string{filepath.ToSlash(filepath.Join(repoRoot, "node_modules", "@types"))},
			"paths": map[string]any{
				"@/*": []string{filepath.ToSlash(filepath.Join(modulesRoot, "*"))},
			},
			"noEmit": true,
		},
		"include": include,
		"exclude": []string{
			filepath.ToSlash(filepath.Join(repoRoot, "node_modules")),
			filepath.ToSlash(filepath.Join(repoRoot, "dist")),
			filepath.ToSlash(filepath.Join(repoRoot, "coverage")),
			"**/*.bak",
			"**/*.gen.*",
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		_ = tmpTsconfigFile.Close()
		return xfmt.Errorf("typecheck: build tsconfig: %w", err)
	}
	if _, err := tmpTsconfigFile.Write(append(data, '\n')); err != nil {
		_ = tmpTsconfigFile.Close()
		return xfmt.Errorf("typecheck: write tsconfig: %w", err)
	}
	if err := tmpTsconfigFile.Close(); err != nil {
		return xfmt.Errorf("typecheck: close tsconfig: %w", err)
	}

	fmt.Fprintf(opts.Stderr, "# typecheck %s\n", app)
	c := exec.CommandContext(ctx, npxPath, "--no-install", "vue-tsc", "-p", tmpTsconfigPath, "--noEmit")
	c.Dir = repoRoot
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		_, _ = opts.Stderr.Write(out)
		if out[len(out)-1] != '\n' {
			fmt.Fprintln(opts.Stderr)
		}
	}
	if err != nil {
		return xfmt.Errorf("typecheck failed for %s: %w", app, err)
	}
	fmt.Fprintf(opts.Stderr, "# typecheck %s ok\n", app)
	return nil
}

func sanitizeAppToken(app string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token := strings.TrimSpace(replacer.Replace(app))
	if token == "" {
		return "app"
	}
	return token
}

func resolveNpxPath(npmPath string) (string, error) {
	if strings.TrimSpace(npmPath) != "" {
		dir := filepath.Dir(npmPath)
		candidate := filepath.Join(dir, "npx")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	if _, err := exec.LookPath("npx"); err != nil {
		return "", xfmt.Errorf("typecheck: missing npx (Node.js). Install Node/npm, then run `npm install` in repo root")
	}
	return "npx", nil
}

func hasTypecheckInputs(modulesPath string, app string) (bool, error) {
	root := filepath.Join(modulesPath, app)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		_ = path
		if d.IsDir() {
			if shouldSkipTypecheckInputScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx") || strings.HasSuffix(name, ".vue") {
			return errTypecheckInputFound
		}
		return nil
	})
	if err == nil {
		return false, nil
	}
	if errors.Is(err, errTypecheckInputFound) {
		return true, nil
	}
	return false, err
}

func shouldSkipTypecheckInputScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".choysum", "tmp":
		return true
	default:
		return false
	}
}
