// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/policy"
	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	gonative "github.com/choysum-dev/choysum/internal/typecheck"
	xfmt "golang.org/x/exp/errors/fmt"
)

// RunOptions configures choysum test typecheck.
type RunOptions struct {
	ModulesPath string
	// NpmPath is retained for call-site compatibility; Go-native typecheck does not use Node.
	NpmPath  string
	RepoRoot string
	TmpPath  string
	Target   string // app name or "all"
	Keep     bool

	Stdout io.Writer
	Stderr io.Writer
}

var errNoTypecheckInputs = errors.New("typecheck: no checkable ts inputs")

var errTypecheckInputFound = errors.New("typecheck: input found")

// walkTypecheckInputsDir is filepath.WalkDir; tests may override for scan errors.
var walkTypecheckInputsDir = filepath.WalkDir

// Test hooks for otherwise unreachable OS edge cases.
var (
	osGetwd              = os.Getwd
	osTempDir            = os.TempDir
	nativeCheck          = gonative.Check
	resolveTestingTmpDir = testingpathing.ResolveTestingTmpDirFromContext
)

// Run typechecks one app or all apps under ModulesPath.
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
		wd, _ := osGetwd()
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
		fmt.Fprintln(opts.Stdout, testsemantics.NoTestsFoundMessage)
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
		fmt.Fprintln(opts.Stdout, testsemantics.NoTestsFoundMessage)
	}
	return nil
}

// ResolveApps returns apps to typecheck for target ("all" or a single app name).
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
			return nil, xfmt.Errorf("%s", testsemantics.PrefixForCommand("typecheck", testsemantics.UnknownAppMessage(target)))
		}
		if !HasTargets(modulesPath, target) {
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
		if HasTargets(modulesPath, name) {
			apps = append(apps, name)
		}
	}
	return apps, nil
}

// HasTargets reports whether app has a service/ or web/ directory.
func HasTargets(modulesPath string, app string) bool {
	serviceDir := filepath.Join(modulesPath, app, "service")
	webDir := filepath.Join(modulesPath, app, "web")
	if st, err := os.Stat(serviceDir); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(webDir); err == nil && st.IsDir() {
		return true
	}
	return false
}

// TypecheckApp runs Go-native typecheck for one application (no Node / vue-tsc).
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
		wd, _ := osGetwd()
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

	tmpRoot := testingpathing.EffectiveCLITestTmpRoot(ctx, opts.TmpPath)
	if tmpRoot == "" {
		tmpRoot = osTempDir()
	}
	if !filepath.IsAbs(tmpRoot) {
		tmpRoot, _ = filepath.Abs(tmpRoot)
	}
	tmpRoot = filepath.Clean(tmpRoot)
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

	warnedMissingTypeAssets, err := warnMissingTypeAssetsPrecheck(opts.Stderr, modulesRoot, app)
	if err != nil {
		return err
	}
	if err := ensureTypeAssets(ctx, opts.Stderr, modulesRoot, app); err != nil {
		return err
	}

	serviceDir := filepath.Join(modulesRoot, app, "service")
	if st, err := os.Stat(serviceDir); err == nil && st.IsDir() {
		if err := policy.CheckServiceImportBoundaryOnDisk(
			modulesRoot,
			app,
			policy.ModulePathAliasForBoundary(modulesRoot),
		); err != nil {
			return xfmt.Errorf("typecheck: %w", err)
		}
	}

	var keepDir string
	if opts.Keep {
		tmpTsconfigRoot, err := resolveTestingTmpDir(ctx, repoRoot, tmpRoot, "typecheck")
		if err != nil {
			return xfmt.Errorf("typecheck: resolve tmp dir: %w", err)
		}
		keepDir = filepath.Join(tmpTsconfigRoot, sanitizeAppToken(app))
		if err := os.MkdirAll(keepDir, 0o755); err != nil {
			return xfmt.Errorf("typecheck: ensure keep dir: %w", err)
		}
		fmt.Fprintf(opts.Stderr, "choysum test typecheck: kept artifacts dir: %s\n", keepDir)
	}

	fmt.Fprintf(opts.Stderr, "# typecheck %s\n", app)
	res, err := nativeCheck(ctx, gonative.Options{
		ModulesPath: modulesRoot,
		RepoRoot:    repoRoot,
		App:         app,
		Scope:       gonative.ScopeAll,
		KeepDir:     keepDir,
	})
	if errors.Is(err, gonative.ErrNoRootFiles) {
		return errNoTypecheckInputs
	}
	if err != nil {
		return formatTypecheckFailureWithGuidance(app, err, err.Error(), warnedMissingTypeAssets)
	}

	gonative.FormatStderr(opts.Stderr, res.Diagnostics)
	var dump strings.Builder
	gonative.FormatStderr(&dump, res.Diagnostics)
	if keepDir != "" {
		diagPath := filepath.Join(keepDir, "diagnostics.txt")
		if writeErr := os.WriteFile(diagPath, []byte(dump.String()), 0o644); writeErr != nil {
			fmt.Fprintf(opts.Stderr, "choysum test typecheck: write %s: %v\n", diagPath, writeErr)
		}
	}

	if err := res.Err(); err != nil {
		return formatTypecheckFailureWithGuidance(app, err, dump.String(), warnedMissingTypeAssets)
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

func hasTypecheckInputs(modulesPath string, app string) (bool, error) {
	root := filepath.Join(modulesPath, app)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	err = walkTypecheckInputsDir(root, func(path string, d fs.DirEntry, err error) error {
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
