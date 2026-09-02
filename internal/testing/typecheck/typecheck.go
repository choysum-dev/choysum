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
	"slices"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/policy"
	moddeps "github.com/choysum-dev/choysum/internal/testing/moddeps"
	noderuntime "github.com/choysum-dev/choysum/internal/testing/noderuntime"
	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/tailscale/hujson"
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

const (
	typecheckVueTSCVersion     = "3.3.7"
	typecheckTypeScriptVersion = "6.0.3"
)

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

	tmpRoot := testingpathing.EffectiveCLITestTmpRoot(ctx, opts.TmpPath)
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

	hasWebSources := false
	if st, err := os.Stat(filepath.Join(modulesRoot, app, "web")); err == nil && st.IsDir() {
		hasWebSources = true
	}

	requiredModules := typecheckRequiredModules(hasWebSources)
	moduleRoots := []string{
		filepath.Join(repoRoot, "node_modules"),
		filepath.Join(modulesRoot, "node_modules"),
		strings.TrimSpace(opts.NpmPath),
		noderuntime.ResolveGlobalNpmRootBestEffort(),
	}
	warnedMissingTypeAssets, err := warnMissingTypeAssetsPrecheck(opts.Stderr, modulesRoot, app)
	if err != nil {
		return err
	}
	if err := noderuntime.PreflightRequiredNodeModules("typecheck", app, requiredModules, moduleRoots...); err != nil {
		return formatTypecheckPreflightError(app, err, warnedMissingTypeAssets)
	}
	if err := validateTypecheckToolchainVersions(moduleRoots...); err != nil {
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

	npxPath, err := resolveNpxPath(opts.NpmPath)
	if err != nil {
		return err
	}

	var viteClientTypesPath string
	if hasWebSources {
		// Reuse the same module roots as preflight so the selected
		// vite/client.d.ts path always matches the locations that satisfied vite.
		viteClientTypesPath = resolveViteClientDTS(repoRoot, moduleRoots...)
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

	// Write ambient stub for subpath imports (locales, plugins, chart
	// sub-modules, etc.) that don't have individual type definitions.
	ambientStubPath := filepath.Join(tmpTsconfigDir, "subpath-stubs.d.ts")
	if err := writeSubpathStubs(ambientStubPath); err != nil {
		return xfmt.Errorf("typecheck: write subpath stubs: %w", err)
	}

	cleanupTmpArtifacts := func() {
		_ = os.Remove(tmpTsconfigPath)
		_ = os.Remove(ambientStubPath)
		_ = os.Remove(tmpTsconfigDir)
	}
	if !opts.Keep {
		defer cleanupTmpArtifacts()
	} else {
		fmt.Fprintf(opts.Stderr, "choysum test typecheck: kept artifacts dir: %s\n", tmpTsconfigDir)
	}

	include := []string{
		filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "*.d.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "service", "**", "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.ts")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.tsx")),
		filepath.ToSlash(filepath.Join(modulesRoot, app, "web", "**", "*.vue")),
	}
	coreAmbientTypes := filepath.Join(modulesRoot, "core", "types", "$choysum.d.ts")
	if _, err := os.Stat(coreAmbientTypes); err == nil {
		include = append(include, filepath.ToSlash(coreAmbientTypes))
	}
	if hasWebSources {
		include = append(include, filepath.ToSlash(viteClientTypesPath))
	}
	include = append(include, filepath.ToSlash(ambientStubPath))

	typeRoots := resolveTypeRoots(repoRoot)
	types := resolveCompilerTypes(typeRoots)
	paths := resolveModulePaths(modulesRoot)
	paths["@/*"] = []string{filepath.ToSlash(filepath.Join(modulesRoot, "*"))}
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
			"types":                        types,
			"typeRoots":                    typeRoots,
			"paths":                        paths,
			"noEmit":                       true,
		},
		"include": include,
		"exclude": []string{
			filepath.ToSlash(filepath.Join(repoRoot, "node_modules")),
			filepath.ToSlash(filepath.Join(repoRoot, "dist")),
			filepath.ToSlash(filepath.Join(repoRoot, "coverage")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "*.test.ts")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "*.spec.ts")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "tests", "**", "*.ts")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "tests", "**", "*.tsx")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "__tests__", "**", "*.ts")),
			filepath.ToSlash(filepath.Join(modulesRoot, app, "**", "__tests__", "**", "*.tsx")),
			"**/*.test.ts",
			"**/*.spec.ts",
			"**/tests/**/*.ts",
			"**/tests/**/*.tsx",
			"**/__tests__/**/*.ts",
			"**/__tests__/**/*.tsx",
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
	c.Env = noderuntime.SanitizeNpmChildEnv(os.Environ())
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		_, _ = opts.Stderr.Write(out)
		if out[len(out)-1] != '\n' {
			fmt.Fprintln(opts.Stderr)
		}
	}
	if err != nil {
		return formatTypecheckFailureWithGuidance(app, err, string(out), warnedMissingTypeAssets)
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

func typecheckRequiredModules(hasWebSources bool) []string {
	if hasWebSources {
		return []string{"vite", "vue-tsc"}
	}
	return []string{"vue-tsc"}
}

func formatTypecheckFailureWithGuidance(app string, runErr error, output string, warnedMissingTypeAssets bool) error {
	baseErr := xfmt.Errorf("typecheck failed for %s: %w", app, runErr)
	if !warnedMissingTypeAssets && !shouldSuggestTypeFetchFromOutput(output) {
		return baseErr
	}

	app = strings.TrimSpace(app)
	if app == "" {
		app = "<app>"
	}
	return xfmt.Errorf(
		"%w\nrecommended action:\n  go run . type-fetch %s",
		baseErr,
		app,
	)
}

func shouldSuggestTypeFetchFromOutput(output string) bool {
	output = strings.ToLower(strings.TrimSpace(output))
	if output == "" {
		return false
	}
	if strings.Contains(output, "ts2307") || strings.Contains(output, "ts7016") {
		return true
	}
	if strings.Contains(output, "cannot find module") && strings.Contains(output, "type declaration") {
		return true
	}
	return false
}

func warnMissingTypeAssetsPrecheck(stderr io.Writer, modulesRoot string, app string) (bool, error) {
	if stderr == nil {
		return false, nil
	}

	externalModules, err := moddeps.CollectExternalModuleDependencies(modulesRoot, []string{app}, true)
	if err != nil {
		return false, xfmt.Errorf("typecheck: collect module dependencies: %w", err)
	}
	if len(externalModules) == 0 {
		return false, nil
	}

	missingModules := missingTypeAssetModules(modulesRoot, externalModules)
	if len(missingModules) == 0 {
		return false, nil
	}

	preview := strings.Join(missingModules, ", ")
	if len(missingModules) > 3 {
		preview = strings.Join(missingModules[:3], ", ") + ", ..."
	}

	_, _ = fmt.Fprintf(
		stderr,
		"Warning: type declarations may be incomplete for %s.\nmissing %d module(s) (sample: %s)\nrecommended action:\n  go run . type-fetch %s\n\n",
		app,
		len(missingModules),
		preview,
		app,
	)
	return true, nil
}

func formatTypecheckPreflightError(app string, preflightErr error, warnedMissingTypeAssets bool) error {
	var missingErr *noderuntime.MissingNodeModulesPreflightError
	if !errors.As(preflightErr, &missingErr) {
		return preflightErr
	}

	app = strings.TrimSpace(app)
	if app == "" {
		app = "<app>"
	}

	missingModules := noderuntime.NormalizeStringList(missingErr.MissingModules)

	var b strings.Builder
	fmt.Fprintf(&b, "typecheck preflight failed for %s. tests were not started.\n", app)
	fmt.Fprintf(&b, "%s\n", noderuntime.FormatMissingModulesSummary(missingModules, 3))
	fmt.Fprintf(&b, "install command:\n  %s\n", typecheckInstallCommand(missingModules))
	if warnedMissingTypeAssets {
		fmt.Fprintf(&b, "recommended before retry:\n  go run . type-fetch %s\n", app)
	}
	fmt.Fprintf(&b, "retry:\n  go run . test typecheck %s", app)
	b.WriteString("\n")
	return errors.New(b.String())
}

func typecheckInstallCommand(missingModules []string) string {
	packages := make([]string, 0, len(missingModules)+2)
	for _, moduleName := range noderuntime.NormalizeStringList(missingModules) {
		switch moduleName {
		case "vue-tsc":
			packages = append(packages, "vue-tsc@"+typecheckVueTSCVersion)
		case "typescript":
			packages = append(packages, "typescript@"+typecheckTypeScriptVersion)
		default:
			packages = append(packages, moduleName)
		}
	}

	// vue-tsc currently requires the TypeScript 5/6 JavaScript API. Installing
	// vue-tsc without an explicit TypeScript version can resolve TypeScript 7,
	// which fails before project diagnostics with ERR_PACKAGE_PATH_NOT_EXPORTED.
	if slices.Contains(packages, "vue-tsc@"+typecheckVueTSCVersion) &&
		!slices.Contains(packages, "typescript@"+typecheckTypeScriptVersion) {
		packages = append(packages, "typescript@"+typecheckTypeScriptVersion)
	}
	if slices.Contains(packages, "vue-tsc@"+typecheckVueTSCVersion) &&
		!slices.Contains(packages, "@types/node") {
		packages = append(packages, "@types/node")
	}

	return "npm install -g " + strings.Join(packages, " ")
}

func validateTypecheckToolchainVersions(moduleRoots ...string) error {
	roots := noderuntime.NormalizeModuleRoots(moduleRoots...)
	vueTSCVersion, vueTSCRoot := nodePackageVersion("vue-tsc", roots...)

	typescriptRoots := make([]string, 0, len(roots)+1)
	if vueTSCRoot != "" {
		typescriptRoots = append(typescriptRoots, filepath.Join(vueTSCRoot, "vue-tsc", "node_modules"))
	}
	typescriptRoots = append(typescriptRoots, roots...)
	typescriptVersion, _ := nodePackageVersion("typescript", typescriptRoots...)

	var mismatches []string
	if vueTSCVersion != "" && vueTSCVersion != typecheckVueTSCVersion {
		mismatches = append(mismatches, fmt.Sprintf("vue-tsc=%s (required %s)", vueTSCVersion, typecheckVueTSCVersion))
	}
	// Pin typescript only when vue-tsc is present; the 5/6 API constraint is vue-tsc-specific.
	if vueTSCVersion != "" && typescriptVersion != "" && typescriptVersion != typecheckTypeScriptVersion {
		mismatches = append(mismatches, fmt.Sprintf("typescript=%s (required %s)", typescriptVersion, typecheckTypeScriptVersion))
	}
	if len(mismatches) == 0 {
		return nil
	}

	return xfmt.Errorf(
		"typecheck toolchain version mismatch: %s\ninstall command:\n  %s",
		strings.Join(mismatches, ", "),
		typecheckInstallCommand([]string{"vue-tsc"}),
	)
}

func nodePackageVersion(moduleName string, moduleRoots ...string) (string, string) {
	for _, root := range noderuntime.NormalizeModuleRoots(moduleRoots...) {
		packageJSONPath := filepath.Join(root, filepath.FromSlash(moduleName), "package.json")
		data, err := os.ReadFile(packageJSONPath)
		if err != nil {
			continue
		}
		var manifest struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		if version := strings.TrimSpace(manifest.Version); version != "" {
			return version, root
		}
	}
	return "", ""
}

func missingTypeAssetModules(modulesRoot string, expectedModules []string) []string {
	pathsByModule := readModuleTSConfigPaths(modulesRoot)
	if len(pathsByModule) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for _, moduleName := range noderuntime.NormalizeStringList(expectedModules) {
		pathEntries, hasPathMapping := resolveTypePathEntries(pathsByModule, moduleName)
		if !hasPathMapping {
			continue
		}
		if hasAnyExistingTypeAsset(pathEntries, modulesRoot) {
			continue
		}
		missing = append(missing, moduleName)
	}
	return missing
}

func resolveTypePathEntries(pathsByModule map[string][]string, moduleName string) ([]string, bool) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, false
	}

	entries, ok := pathsByModule[moduleName]
	if ok {
		return entries, true
	}

	resolved := make([]string, 0, 4)
	hasMapping := false
	for key, valueEntries := range pathsByModule {
		replacements, matched := matchTSConfigPathPattern(strings.TrimSpace(key), moduleName)
		if !matched {
			continue
		}
		hasMapping = true
		for _, entry := range valueEntries {
			resolved = append(resolved, applyPathPatternReplacements(entry, replacements))
		}
	}

	return resolved, hasMapping
}

func matchTSConfigPathPattern(pattern string, moduleName string) ([]string, bool) {
	if pattern == "" {
		return nil, false
	}
	if !strings.Contains(pattern, "*") {
		return nil, pattern == moduleName
	}

	parts := strings.Split(pattern, "*")
	starMatches := make([]string, 0, len(parts)-1)
	remain := moduleName

	if !strings.HasPrefix(remain, parts[0]) {
		return nil, false
	}
	remain = strings.TrimPrefix(remain, parts[0])

	for i := 1; i < len(parts); i++ {
		segment := parts[i]
		if i == len(parts)-1 {
			if segment == "" {
				starMatches = append(starMatches, remain)
				return starMatches, true
			}
			if !strings.HasSuffix(remain, segment) {
				return nil, false
			}
			starMatches = append(starMatches, remain[:len(remain)-len(segment)])
			return starMatches, true
		}

		if segment == "" {
			starMatches = append(starMatches, "")
			continue
		}

		index := strings.Index(remain, segment)
		if index < 0 {
			return nil, false
		}
		starMatches = append(starMatches, remain[:index])
		remain = remain[index+len(segment):]
	}

	return starMatches, true
}

func applyPathPatternReplacements(pathPattern string, replacements []string) string {
	if len(replacements) == 0 || !strings.Contains(pathPattern, "*") {
		return pathPattern
	}

	var b strings.Builder
	replacementIndex := 0
	for _, ch := range pathPattern {
		if ch != '*' {
			b.WriteRune(ch)
			continue
		}
		if replacementIndex < len(replacements) {
			b.WriteString(replacements[replacementIndex])
			replacementIndex++
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func readModuleTSConfigPaths(modulesRoot string) map[string][]string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return nil
	}

	var tsconfig struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		// tsconfig.json is often JSONC (comments/trailing commas).
		// Use hujson to standardize into strict JSON before unmarshaling.
		jsoncValue, parseErr := hujson.Parse(data)
		if parseErr != nil {
			return nil
		}
		jsoncValue.Standardize()
		if err := json.Unmarshal(jsoncValue.Pack(), &tsconfig); err != nil {
			return nil
		}
	}
	return tsconfig.CompilerOptions.Paths
}

func hasAnyExistingTypeAsset(pathEntries []string, modulesRoot string) bool {
	for _, entry := range pathEntries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		resolvedPath := entry
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(modulesRoot, filepath.FromSlash(resolvedPath))
		}

		if strings.ContainsAny(resolvedPath, "*?[]") {
			matches, err := filepath.Glob(resolvedPath)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if st, err := os.Stat(match); err == nil && !st.IsDir() {
					return true
				}
			}
			continue
		}

		if st, err := os.Stat(resolvedPath); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func resolveNpxPath(npmPath string) (string, error) {
	hints := make([]string, 0, 2)
	if strings.TrimSpace(npmPath) != "" {
		hints = append(hints, npmPath, filepath.Dir(npmPath))
	}
	if npxPath, _, found := noderuntime.FindExecutableInRoots("npx", hints...); found {
		return npxPath, nil
	}
	if npxPath, _, found := noderuntime.FindExecutable("npx"); found {
		return npxPath, nil
	}
	return "", xfmt.Errorf("typecheck: npx not found. Ensure Node.js/npm is installed and npx is in PATH")
}

// resolveViteClientDTS returns the first existing vite/client.d.ts path from the
// provided module roots and falls back to repoRoot/node_modules for clear errors.
func resolveViteClientDTS(repoRoot string, moduleRoots ...string) string {
	for _, moduleRoot := range noderuntime.NormalizeModuleRoots(moduleRoots...) {
		viteClientPath := filepath.Join(moduleRoot, "vite", "client.d.ts")
		if _, err := os.Stat(viteClientPath); err == nil {
			return viteClientPath
		}
	}
	return filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts")
}

// resolveTypeRoots returns typeRoots entries for the generated tsconfig,
// including both local node_modules/@types and the global npm prefix.
func resolveTypeRoots(repoRoot string) []string {
	var roots []string
	localTypes := filepath.Join(repoRoot, "node_modules", "@types")
	if _, err := os.Stat(localTypes); err == nil {
		roots = append(roots, filepath.ToSlash(localTypes))
	}
	if globalRoot, err := globalNpmRoot(); err == nil {
		globalTypes := filepath.Join(globalRoot, "@types")
		if _, err := os.Stat(globalTypes); err == nil {
			roots = append(roots, filepath.ToSlash(globalTypes))
		}
	}
	if len(roots) == 0 {
		// Keep a sensible default so the error message is clear.
		roots = append(roots, filepath.ToSlash(localTypes))
	}
	return roots
}

// globalNpmRoot returns the global node_modules path (npm root -g).
func globalNpmRoot() (string, error) {
	return noderuntime.ResolveGlobalNpmRoot()
}

// resolveCompilerTypes returns the "types" compiler option, including
// only type libraries that are actually installed in a typeRoot.
func resolveCompilerTypes(typeRoots []string) []string {
	for _, root := range typeRoots {
		if _, err := os.Stat(filepath.Join(root, "node")); err == nil {
			return []string{"node"}
		}
	}
	return nil
}

// resolveModulePaths reads modules/tsconfig.json and returns its "paths"
// entries resolved to absolute paths (relative to the modules directory).
func resolveModulePaths(modulesRoot string) map[string]any {
	paths := make(map[string]any)
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return paths
	}
	var tsconfig struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return paths
	}
	for alias, targets := range tsconfig.CompilerOptions.Paths {
		var absTargets []string
		for _, t := range targets {
			if filepath.IsAbs(t) {
				absTargets = append(absTargets, filepath.ToSlash(t))
			} else {
				absTargets = append(absTargets, filepath.ToSlash(filepath.Join(modulesRoot, t)))
			}
		}
		paths[alias] = absTargets
	}
	return paths
}

// writeSubpathStubs writes a .d.ts file that declares ambient modules for
// subpath imports (locales, plugins, etc.) that don't have individual
// type definitions on esm.sh.  The open-ended declare module syntax
// (without {}) tells TypeScript the module exists without checking
// specific exports.
func writeSubpathStubs(dst string) error {
	stubs := []string{
		// dayjs locales and plugins (pure JS, no .d.ts).
		"dayjs/locale/*",
		"dayjs/plugin/*",
		// Style and static-asset side-effect imports used by web entrypoints.
		"*.css",
		"*.scss",
		"*.sass",
		"*.svg",
		// element-plus locale lang modules (pure JS).
		"element-plus/es/locale/lang/*",
		// External modules/subpaths without stable d.ts coverage.
		"@element-plus/icons-vue",
		"nprogress",
		// Test-only imports.
		"vitest",
		"@vue/test-utils",
	}
	var b strings.Builder
	b.WriteString("// Ambient declarations for subpath imports without individual types.\n")
	for _, mod := range stubs {
		fmt.Fprintf(&b, "declare module %q;\n", mod)
	}
	b.WriteString(`
interface ImportMetaEnv {
	readonly [key: string]: string | undefined;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}

declare module "@bufbuild/protobuf/codegenv2" {
  export type Message = any;
  export type GenFile = any;
  export type GenMessage<T = any> = any;
  export type GenService<T = any> = any;
  export const fileDesc: any;
  export const messageDesc: any;
  export const enumDesc: any;
  export const serviceDesc: any;
}

declare module "@bufbuild/protobuf/wkt" {
  export type Value = any;
  export const EmptySchema: any;
  export const ListValueSchema: any;
  export const NullValue: any;
  export const StructSchema: any;
  export const ValueSchema: any;
}

declare module "kysely/helpers/postgres" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/mysql" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/sqlite" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/mssql" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "echarts/core" {
  export const use: (...args: any[]) => void;
}

declare module "echarts/charts" {
  export const BarChart: any;
  export const LineChart: any;
  export const PieChart: any;
}

declare module "echarts/components" {
  export const TitleComponent: any;
  export const TooltipComponent: any;
  export const LegendComponent: any;
  export const GridComponent: any;
}

declare module "echarts/renderers" {
  export const SVGRenderer: any;
}

declare module "element-plus/es/components/table-v2/src/row" {
	export type RowEventHandlerParams = any;
}

declare module "element-plus/es/components/table-v2/src/types" {
	export type RowEventHandlerParams = import("element-plus/es/components/table-v2/src/row").RowEventHandlerParams;
	export type KeyType = string | number;
}

declare module "fast-deep-equal" {
	export default function equal(a: any, b: any): boolean;
}

declare module "node:fs" {
	export function readFileSync(path: string, encoding?: string): string;
}

declare module "node:path" {
	export function resolve(...paths: string[]): string;
}
`)
	return os.WriteFile(dst, []byte(b.String()), 0o644)
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
