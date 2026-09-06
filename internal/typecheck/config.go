// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/buke/typescript-go-internal/v7/pkg/collections"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/tailscale/hujson"
)

// userHomeDir is os.UserHomeDir; tests may override when covering empty-root paths.
var userHomeDir = os.UserHomeDir

// BuildCompilerOptions builds compiler options aligned with the historical
// typecheck temporary tsconfig (service + web TS/TSX / strict / bundler).
//
// Relative path aliases are resolved against tsconfig compilerOptions.baseUrl
// when set (default: modules root). CompilerOptions.BaseUrl itself is not set:
// typescript-go rejects that option (TS5102).
func BuildCompilerOptions(modulesPath, repoRoot string) (*core.CompilerOptions, error) {
	modulesPath = filepath.Clean(modulesPath)
	repoRoot = filepath.Clean(repoRoot)

	paths, pathsBase, err := resolveModulePaths(modulesPath, repoRoot)
	if err != nil {
		return nil, err
	}
	pathsMap := collections.NewOrderedMapWithSizeHint[string, []string](len(paths) + 1)
	aliases := make([]string, 0, len(paths))
	for alias := range paths {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	for _, alias := range aliases {
		pathsMap.Set(alias, paths[alias])
	}
	if _, ok := pathsMap.Get("@/*"); !ok {
		pathsMap.Set("@/*", []string{filepath.ToSlash(filepath.Join(pathsBase, "*"))})
	}

	typeRoots := resolveTypeRoots(modulesPath, repoRoot)
	types := resolveCompilerTypes(modulesPath, typeRoots)

	opts := &core.CompilerOptions{
		Target:                           core.ScriptTargetES2020,
		Module:                           core.ModuleKindESNext,
		ModuleResolution:                 core.ModuleResolutionKindBundler,
		Jsx:                              core.JsxEmitPreserve,
		Strict:                           core.TSTrue,
		StrictNullChecks:                 core.TSTrue,
		StrictPropertyInitialization:     core.TSFalse,
		ExperimentalDecorators:           core.TSTrue,
		AllowJs:                          core.TSTrue,
		AllowArbitraryExtensions:         core.TSTrue,
		SkipLibCheck:                     core.TSTrue,
		NoEmit:                           core.TSTrue,
		Lib:                              []string{"es2020", "dom", "dom.iterable"},
		Paths:                            pathsMap,
		PathsBasePath:                    filepath.ToSlash(pathsBase),
		TypeRoots:                        typeRoots,
		Types:                            types,
		ForceConsistentCasingInFileNames: core.TSTrue,
	}
	return opts, nil
}

func resolveModulePaths(modulesRoot, repoRoot string) (map[string][]string, string, error) {
	_ = repoRoot // reserved; typecheck must not read node_modules (use type-fetch paths only).
	out := make(map[string][]string)
	pathsBase := modulesRoot
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, pathsBase, nil
		}
		return nil, "", fmt.Errorf("typecheck: read %s: %w", tsconfigPath, err)
	}
	var tsconfig struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	// tsconfig.json is JSONC (comments / trailing commas); standardize first.
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil, "", fmt.Errorf("typecheck: parse %s: %w", tsconfigPath, err)
	}
	hv.Standardize()
	data = hv.Pack()
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return nil, "", fmt.Errorf("typecheck: parse %s: %w", tsconfigPath, err)
	}
	if base := strings.TrimSpace(tsconfig.CompilerOptions.BaseURL); base != "" {
		if filepath.IsAbs(base) {
			pathsBase = filepath.Clean(base)
		} else {
			pathsBase = filepath.Clean(filepath.Join(modulesRoot, base))
		}
	}
	for alias, targets := range tsconfig.CompilerOptions.Paths {
		var absTargets []string
		for _, t := range targets {
			var abs string
			if !filepath.IsAbs(t) {
				abs = filepath.ToSlash(filepath.Join(pathsBase, t))
			} else {
				abs = filepath.ToSlash(t)
			}
			abs = rewriteChoysumTypesPath(abs)
			if !typePathExists(abs) {
				continue
			}
			absTargets = append(absTargets, abs)
		}
		if len(absTargets) == 0 {
			continue
		}
		out[alias] = absTargets
	}
	return out, pathsBase, nil
}

// rewriteChoysumTypesPath remaps repo-relative ../../.choysum/pkg/types/... entries
// that point at a missing tree onto type-fetch caches. modules/tsconfig paths are
// relative to modules/ (`../../.choysum/...`), so they only resolve to ~/.choysum
// when the repo itself lives at $HOME/<name>. CI checkouts under
// $GITHUB_WORKSPACE do not; search CHOYSUM_HOME, ~/.choysum, and CHOYSUM_TEST_TMP
// cache instead.
func rewriteChoysumTypesPath(abs string) string {
	abs = filepath.ToSlash(abs)
	const marker = "/.choysum/pkg/types/"
	idx := strings.Index(abs, marker)
	if idx < 0 {
		return abs
	}
	suffix := abs[idx+len(marker):]
	pick := func(candidate string) (string, bool) {
		if !typePathExists(candidate) {
			return "", false
		}
		// Prefer a complete vue type-fetch graph over an incomplete first hit
		// (empty sibling stubs pass Stat but export almost nothing).
		if isVueTypeFetchEntryPath(candidate) && !vueTypeEntryComplete(candidate) {
			return candidate, false
		}
		return candidate, true
	}
	var incompleteFallback string
	if got, ok := pick(abs); ok {
		return got
	} else if got != "" {
		// Incomplete local hit — keep searching for a complete copy.
		incompleteFallback = got
	}
	for _, root := range choysumTypesSearchRoots() {
		alt := filepath.ToSlash(filepath.Join(root, filepath.FromSlash(suffix)))
		if got, ok := pick(alt); ok {
			return got
		} else if got != "" && incompleteFallback == "" {
			incompleteFallback = got
		}
	}
	if incompleteFallback != "" {
		return incompleteFallback
	}
	return abs
}

// isVueTypeFetchEntryPath reports whether path looks like an esm.sh vue package
// entry produced by type-fetch (needs runtime-* siblings to be usable).
func isVueTypeFetchEntryPath(path string) bool {
	base := filepath.Base(filepath.FromSlash(path))
	return vueTypeFetchEntryRE.MatchString(base)
}

// RewriteChoysumTypesPath is the exported form of rewriteChoysumTypesPath for
// typecheck orchestration (missing-asset prechecks).
func RewriteChoysumTypesPath(abs string) string {
	return rewriteChoysumTypesPath(abs)
}

// HasResolvableVueTypes reports whether modules/tsconfig can resolve the `vue`
// package via type-fetch path assets (not node_modules).
func HasResolvableVueTypes(modulesPath, repoRoot string) bool {
	return hasResolvableVueTypes(modulesPath, repoRoot)
}

// VueTypeEntryComplete reports whether a resolved `vue` paths target includes a
// usable type-fetch graph (entry + @vue/runtime-* siblings with real exports).
func VueTypeEntryComplete(entry string) bool {
	return vueTypeEntryComplete(entry)
}

// choysumTypesSearchRoots returns candidate directories that hold type-fetch .d.ts
// files (each entry is the `…/pkg/types` directory itself).
func choysumTypesSearchRoots() []string {
	var roots []string
	seen := map[string]struct{}{}
	add := func(p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		roots = append(roots, p)
	}
	if v := strings.TrimSpace(os.Getenv("CHOYSUM_HOME")); v != "" {
		add(filepath.Join(v, "pkg", "types"))
	}
	// CLI test harness durable cache (unit/e2e/typecheck).
	if tmp := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_TMP")); tmp != "" {
		add(filepath.Join(tmp, "cache", "pkg", "types"))
	}
	if home, err := userHomeDir(); err == nil {
		add(filepath.Join(home, ".choysum", "pkg", "types"))
	}
	return roots
}

// PreferTypesWriteDir picks where typecheck should download missing type-fetch
// assets: CHOYSUM_HOME, then CHOYSUM_TEST_TMP cache, then ~/.choysum.
func PreferTypesWriteDir() string {
	roots := choysumTypesSearchRoots()
	if len(roots) == 0 {
		return ""
	}
	// Search order already prefers writable CLI/cache homes over ~/.choysum.
	return roots[0]
}

func typePathExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	// Glob entries (type-fetch wildcards) are treated as present; TS resolves them.
	if strings.ContainsAny(path, "*?[") {
		return true
	}
	if _, err := os.Stat(filepath.FromSlash(path)); err == nil {
		return true // file or directory (TS may resolve index.*)
	}
	// Extensionless / directory-index targets used by some path mappings.
	for _, suffix := range []string{".d.ts", ".d.mts", ".ts", ".tsx", ".vue", "/index.d.ts", "/index.d.mts", "/index.ts", "/index.tsx"} {
		if st, err := os.Stat(filepath.FromSlash(path + suffix)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// hasResolvableVueTypes reports whether Vue package types are available via
// modules/tsconfig paths (type-fetch under ~/.choysum/pkg/types). Does not
// consult node_modules. A type-fetch vue entry alone is not enough — the entry
// only re-exports @vue/runtime-dom; without those siblings the module resolves
// but exports almost nothing (compileToFunction only).
func hasResolvableVueTypes(modulesPath, repoRoot string) bool {
	paths, _, err := resolveModulePaths(modulesPath, repoRoot)
	if err != nil {
		return false
	}
	targets, ok := paths["vue"]
	if !ok {
		return false
	}
	for _, t := range targets {
		if vueTypeEntryComplete(t) {
			return true
		}
	}
	return false
}

// vueTypeEntryComplete reports whether a resolved `vue` paths target is usable.
// Type-fetch entries named esm.sh_vue@<ver>_… must also have runtime-dom /
// runtime-core / reactivity siblings in the same directory with real exports
// (empty `export {}` placeholders are not enough — they pass Stat but leave
// `import { h, PropType, toRef } from 'vue'` unresolved).
func vueTypeEntryComplete(entry string) bool {
	entry = strings.TrimSpace(entry)
	if entry == "" || !typePathExists(entry) {
		return false
	}
	base := filepath.Base(filepath.FromSlash(entry))
	m := vueTypeEntryVerRE.FindStringSubmatch(base)
	if len(m) != 2 {
		// Fixture / non-type-fetch targets: presence is enough.
		return true
	}
	ver := m[1]
	dir := filepath.Dir(filepath.FromSlash(entry))
	domName := fmt.Sprintf("esm.sh_@vue_runtime-dom@%s_dist_runtime-dom.d.ts.d.ts", ver)
	coreName := fmt.Sprintf("esm.sh_@vue_runtime-core@%s_dist_runtime-core.d.ts.d.ts", ver)
	reactName := fmt.Sprintf("esm.sh_@vue_reactivity@%s_dist_reactivity.d.ts.d.ts", ver)
	for _, name := range []string{domName, coreName, reactName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	// Prefer runtime-core markers: runtime-dom often only mentions ExtractPropTypes
	// (substring of PropType) while re-exporting the real PropType/h from core.
	coreData, err := os.ReadFile(filepath.Join(dir, coreName))
	if err != nil {
		return false
	}
	if !vueCoreExportRE.Match(coreData) {
		return false
	}
	reactData, err := os.ReadFile(filepath.Join(dir, reactName))
	if err != nil || !vueToRefExportRE.Match(reactData) {
		return false
	}
	return true
}

// Word-boundary markers so ExtractPropTypes does not count as PropType.
var (
	vueTypeFetchEntryRE = regexp.MustCompile(`(?i)^esm\.sh_vue@`)
	vueTypeEntryVerRE   = regexp.MustCompile(`(?i)^esm\.sh_vue@([0-9][^/_]*?)(?:_|\.d\.|$)`)
	vueCoreExportRE     = regexp.MustCompile(`\bPropType\b|declare function h\b|function h<`)
	vueToRefExportRE    = regexp.MustCompile(`\btoRef\b`)
)

func resolveTypeRoots(modulesPath, repoRoot string) []string {
	var roots []string
	seen := map[string]struct{}{}
	add := func(p string) {
		p = filepath.ToSlash(filepath.Clean(p))
		if _, ok := seen[p]; ok {
			return
		}
		if st, err := os.Stat(filepath.FromSlash(p)); err != nil || !st.IsDir() {
			return
		}
		seen[p] = struct{}{}
		roots = append(roots, p)
	}

	// Prefer modules/tsconfig.json typeRoots (type-fetch under ~/.choysum).
	for _, raw := range readTsconfigTypeRoots(modulesPath) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			// Empty entries Join to modulesPath and would wrongly register it
			// as a custom type root.
			continue
		}
		abs := raw
		if !filepath.IsAbs(raw) {
			abs = filepath.Join(modulesPath, raw)
		}
		abs = rewriteChoysumTypesPath(filepath.ToSlash(abs))
		// typeRoots entries are directories; rewriteChoysumTypesPath may leave a
		// missing file path — also try directory existence via choysum home.
		if !typePathExists(abs) && !dirExists(abs) {
			if alt := rewriteChoysumTypesDir(abs); alt != "" {
				abs = alt
			}
		}
		add(abs)
	}

	// Opportunistic only — typecheck must not require node_modules.
	add(filepath.Join(repoRoot, "node_modules", "@types"))
	if globalRoot := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); globalRoot != "" {
		add(filepath.Join(globalRoot, "@types"))
	}
	return roots
}

func readTsconfigTypeRoots(modulesRoot string) []string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		return nil
	}
	var tsconfig struct {
		CompilerOptions struct {
			TypeRoots []string `json:"typeRoots"`
			Types     []string `json:"types"`
		} `json:"compilerOptions"`
	}
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil
	}
	hv.Standardize()
	if err := json.Unmarshal(hv.Pack(), &tsconfig); err != nil {
		return nil
	}
	return tsconfig.CompilerOptions.TypeRoots
}

func readTsconfigTypes(modulesRoot string) []string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		return nil
	}
	var tsconfig struct {
		CompilerOptions struct {
			Types []string `json:"types"`
		} `json:"compilerOptions"`
	}
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil
	}
	hv.Standardize()
	if err := json.Unmarshal(hv.Pack(), &tsconfig); err != nil {
		return nil
	}
	return tsconfig.CompilerOptions.Types
}

func dirExists(path string) bool {
	st, err := os.Stat(filepath.FromSlash(path))
	return err == nil && st.IsDir()
}

// rewriteChoysumTypesDir remaps missing /.choysum/pkg/types/... directories onto
// type-fetch caches (same search roots as rewriteChoysumTypesPath).
func rewriteChoysumTypesDir(abs string) string {
	abs = filepath.ToSlash(abs)
	const marker = "/.choysum/pkg/types/"
	idx := strings.Index(abs, marker)
	if idx < 0 {
		const markerExact = "/.choysum/pkg/types"
		if !strings.HasSuffix(abs, markerExact) {
			return ""
		}
		for _, root := range choysumTypesSearchRoots() {
			if dirExists(root) {
				return filepath.ToSlash(root)
			}
		}
		return ""
	}
	if dirExists(abs) {
		return abs
	}
	suffix := abs[idx+len(marker):]
	for _, root := range choysumTypesSearchRoots() {
		alt := filepath.ToSlash(filepath.Join(root, filepath.FromSlash(suffix)))
		if dirExists(alt) {
			return alt
		}
	}
	return ""
}

func resolveCompilerTypes(modulesPath string, typeRoots []string) []string {
	if configured := readTsconfigTypes(modulesPath); len(configured) > 0 {
		var out []string
		for _, name := range configured {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			for _, root := range typeRoots {
				if dirExists(filepath.Join(root, name)) {
					out = append(out, name)
					break
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	for _, root := range typeRoots {
		if dirExists(filepath.Join(root, "node")) {
			return []string{"node"}
		}
	}
	return nil
}

// ResolveModulePathsForTest exposes path resolution for unit tests.
func ResolveModulePathsForTest(modulesRoot, repoRoot string) (map[string][]string, string, error) {
	return resolveModulePaths(modulesRoot, repoRoot)
}

// Test hooks for hard-to-trigger filesystem failures.
var readFile = os.ReadFile
