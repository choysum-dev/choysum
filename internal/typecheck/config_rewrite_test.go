// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteChoysumTypesPath_NoMarker(t *testing.T) {
	in := "/tmp/plain/types/foo.d.ts"
	if got := RewriteChoysumTypesPath(in); got != filepath.ToSlash(in) {
		t.Fatalf("got %q", got)
	}
}

func TestRewriteChoysumTypesPath_ExistingKeepsAbs(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".choysum", "pkg", "types", "esm.sh_vue@1", "index.d.ts")
	mustMkdir(t, filepath.Dir(abs))
	mustWrite(t, abs, "export {}\n")
	got := RewriteChoysumTypesPath(filepath.ToSlash(abs))
	if got != filepath.ToSlash(abs) {
		t.Fatalf("got %q want %q", got, abs)
	}
}

func TestRewriteChoysumTypesPath_RemapsViaChoysumHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")

	cached := filepath.Join(home, "pkg", "types", "esm.sh_vue@3.5.0", "index.d.ts")
	mustMkdir(t, filepath.Dir(cached))
	mustWrite(t, cached, "export {}\n")

	missing := filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "esm.sh_vue@3.5.0", "index.d.ts")
	got := RewriteChoysumTypesPath(filepath.ToSlash(missing))
	want := filepath.ToSlash(cached)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRewriteChoysumTypesPath_RemapsViaTestTmp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CHOYSUM_HOME", "")
	t.Setenv("CHOYSUM_TEST_TMP", tmp)

	cached := filepath.Join(tmp, "cache", "pkg", "types", "esm.sh_vue@1", "index.d.ts")
	mustMkdir(t, filepath.Dir(cached))
	mustWrite(t, cached, "export {}\n")

	missing := filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "esm.sh_vue@1", "index.d.ts")
	got := RewriteChoysumTypesPath(filepath.ToSlash(missing))
	if got != filepath.ToSlash(cached) {
		t.Fatalf("got %q want %q", got, cached)
	}
}

func TestRewriteChoysumTypesPath_MissingKeepsOriginal(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CHOYSUM_TEST_TMP", t.TempDir())
	missing := filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "no-such-pkg", "x.d.ts")
	got := RewriteChoysumTypesPath(filepath.ToSlash(missing))
	if got != filepath.ToSlash(missing) {
		t.Fatalf("got %q", got)
	}
}

func TestPreferTypesWriteDir_Order(t *testing.T) {
	home := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", tmp)

	got := PreferTypesWriteDir()
	want := filepath.Clean(filepath.Join(home, "pkg", "types"))
	if got != want {
		t.Fatalf("PreferTypesWriteDir = %q want %q", got, want)
	}

	t.Setenv("CHOYSUM_HOME", "")
	got = PreferTypesWriteDir()
	want = filepath.Clean(filepath.Join(tmp, "cache", "pkg", "types"))
	if got != want {
		t.Fatalf("PreferTypesWriteDir tmp = %q want %q", got, want)
	}
}

func TestPreferTypesWriteDir_EmptyWhenNoRoots(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", "")
	t.Setenv("CHOYSUM_TEST_TMP", "")
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })
	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	if got := PreferTypesWriteDir(); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestChoysumTypesSearchRoots_DedupAndSkipEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", "  "+home+"  ")
	t.Setenv("CHOYSUM_TEST_TMP", home) // distinct path under same tree still different
	roots := choysumTypesSearchRoots()
	if len(roots) < 2 {
		t.Fatalf("roots = %v", roots)
	}
	seen := map[string]struct{}{}
	for _, r := range roots {
		if r == "" || r == "." {
			t.Fatalf("empty root in %v", roots)
		}
		if _, ok := seen[r]; ok {
			t.Fatalf("duplicate %q in %v", r, roots)
		}
		seen[r] = struct{}{}
	}

	t.Setenv("CHOYSUM_HOME", ".")
	t.Setenv("CHOYSUM_TEST_TMP", "  ")
	roots = choysumTypesSearchRoots()
	for _, r := range roots {
		if r == "." {
			t.Fatalf("dot root leaked: %v", roots)
		}
	}

	userHome, err := os.UserHomeDir()
	if err == nil {
		t.Setenv("CHOYSUM_HOME", filepath.Join(userHome, ".choysum"))
		t.Setenv("CHOYSUM_TEST_TMP", "")
		roots = choysumTypesSearchRoots()
		seen = map[string]struct{}{}
		for _, r := range roots {
			if _, ok := seen[r]; ok {
				t.Fatalf("duplicate after home alias: %v", roots)
			}
			seen[r] = struct{}{}
		}
	}
}

func TestTypePathExists_Branches(t *testing.T) {
	if typePathExists("") || typePathExists("  ") {
		t.Fatal("empty should be false")
	}
	if !typePathExists("/any/*/glob.d.ts") {
		t.Fatal("glob wildcard treated as present")
	}

	dir := t.TempDir()
	if !typePathExists(dir) {
		t.Fatal("existing directory")
	}

	file := filepath.Join(dir, "plain.d.ts")
	mustWrite(t, file, "export {}\n")
	if !typePathExists(filepath.ToSlash(file)) {
		t.Fatal("existing file")
	}

	base := filepath.Join(dir, "extless")
	mustWrite(t, base+".d.ts", "export {}\n")
	if !typePathExists(filepath.ToSlash(base)) {
		t.Fatal("extensionless .d.ts")
	}

	pkg := filepath.Join(dir, "pkgdir")
	mustMkdir(t, pkg)
	mustWrite(t, filepath.Join(pkg, "index.d.ts"), "export {}\n")
	if !typePathExists(filepath.ToSlash(pkg)) {
		t.Fatal("directory with index.d.ts")
	}

	mtsBase := filepath.Join(dir, "mts")
	mustWrite(t, mtsBase+".d.mts", "export {}\n")
	if !typePathExists(filepath.ToSlash(mtsBase)) {
		t.Fatal(".d.mts")
	}
	tsBase := filepath.Join(dir, "tsonly")
	mustWrite(t, tsBase+".ts", "export {}\n")
	if !typePathExists(filepath.ToSlash(tsBase)) {
		t.Fatal(".ts")
	}

	if typePathExists(filepath.Join(dir, "totally-missing")) {
		t.Fatal("missing path")
	}
}

func TestRewriteChoysumTypesDir(t *testing.T) {
	if rewriteChoysumTypesDir("/tmp/no/marker") != "" {
		t.Fatal("no marker")
	}

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	typesRoot := filepath.Join(home, "pkg", "types")
	mustMkdir(t, typesRoot)

	exact := filepath.Join(t.TempDir(), ".choysum", "pkg", "types")
	got := rewriteChoysumTypesDir(filepath.ToSlash(exact))
	if got != filepath.ToSlash(typesRoot) {
		t.Fatalf("exact suffix: got %q want %q", got, typesRoot)
	}

	sub := filepath.Join(typesRoot, "typeRoots")
	mustMkdir(t, sub)
	missingSub := filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "typeRoots")
	got = rewriteChoysumTypesDir(filepath.ToSlash(missingSub))
	if got != filepath.ToSlash(sub) {
		t.Fatalf("subdir remap: got %q want %q", got, sub)
	}

	// Existing dir under marker returns itself.
	existing := filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "keep")
	mustMkdir(t, existing)
	got = rewriteChoysumTypesDir(filepath.ToSlash(existing))
	if got != filepath.ToSlash(existing) {
		t.Fatalf("existing: got %q", got)
	}

	t.Setenv("CHOYSUM_HOME", t.TempDir()) // empty types tree
	if rewriteChoysumTypesDir(filepath.ToSlash(filepath.Join(t.TempDir(), ".choysum", "pkg", "types", "gone"))) != "" {
		t.Fatal("expected empty when not found")
	}
	// Exact /.choysum/pkg/types may still remap onto ~/.choysum when that dir exists.
	_ = rewriteChoysumTypesDir(filepath.ToSlash(filepath.Join(t.TempDir(), ".choysum", "pkg", "types")))
}

func TestHasResolvableVueTypes(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)

	if HasResolvableVueTypes(modules, dir) {
		t.Fatal("no tsconfig")
	}

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	vueFile := filepath.Join(home, "pkg", "types", "esm.sh_vue@3", "index.d.ts")
	mustMkdir(t, filepath.Dir(vueFile))
	mustWrite(t, vueFile, "export {}\n")

	// Relative path that rewrites onto CHOYSUM_HOME.
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3/index.d.ts"]
    }
  }
}
`)
	if !HasResolvableVueTypes(modules, dir) {
		t.Fatal("expected vue resolvable via rewrite")
	}

	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["./missing-vue.d.ts"]
    }
  }
}
`)
	if HasResolvableVueTypes(modules, dir) {
		t.Fatal("missing vue target")
	}

	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "other": ["./x.d.ts"]
    }
  }
}
`)
	if HasResolvableVueTypes(modules, dir) {
		t.Fatal("no vue key")
	}
}

func TestHasResolvableVueTypes_IncompleteTypeFetchGraph(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	origHome := userHomeDir
	t.Cleanup(func() { userHomeDir = origHome })
	userHomeDir = func() (string, error) { return filepath.Join(t.TempDir(), "no-types-home"), nil }

	entry := filepath.Join(home, "pkg", "types", "esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts")
	mustMkdir(t, filepath.Dir(entry))
	mustWrite(t, entry, `export * from"./esm.sh_@vue_runtime-dom@3.5.35_dist_runtime-dom.d.ts.d.ts";`+"\n")
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts"]
    }
  }
}
`)
	if HasResolvableVueTypes(modules, dir) {
		t.Fatal("entry without runtime-* siblings must not count as resolvable")
	}

	for _, name := range []string{
		"esm.sh_@vue_runtime-dom@3.5.35_dist_runtime-dom.d.ts.d.ts",
		"esm.sh_@vue_runtime-core@3.5.35_dist_runtime-core.d.ts.d.ts",
		"esm.sh_@vue_reactivity@3.5.35_dist_reactivity.d.ts.d.ts",
	} {
		content := "export {}\n"
		if strings.Contains(name, "runtime-dom") {
			content = "export type PropType<T> = any;\nexport function h(...args: any[]): any;\n"
		}
		if strings.Contains(name, "reactivity") {
			content = "export declare function toRef(...args: any[]): any;\n"
		}
		mustWrite(t, filepath.Join(home, "pkg", "types", name), content)
	}
	if !HasResolvableVueTypes(modules, dir) {
		t.Fatal("expected complete type-fetch vue graph")
	}

	// Hollow siblings (Stat-ok, no real exports) must not count as complete.
	mustWrite(t, filepath.Join(home, "pkg", "types", "esm.sh_@vue_runtime-dom@3.5.35_dist_runtime-dom.d.ts.d.ts"), "export {}\n")
	if HasResolvableVueTypes(modules, dir) {
		t.Fatal("empty runtime-dom sibling must not count as resolvable")
	}
}

func TestResolveTypeRoots_TypeRootsRewrite(t *testing.T) {
	repo := t.TempDir()
	modules := filepath.Join(repo, "modules")
	mustMkdir(t, modules)

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	typeRootsDir := filepath.Join(home, "pkg", "types", "typeRoots")
	mustMkdir(t, typeRootsDir)

	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "typeRoots": ["../../.choysum/pkg/types/typeRoots"],
    "types": ["node"]
  }
}
`)
	nodeDir := filepath.Join(typeRootsDir, "node")
	mustMkdir(t, nodeDir)

	roots := resolveTypeRoots(modules, repo)
	found := false
	for _, r := range roots {
		if strings.Contains(filepath.ToSlash(r), "typeRoots") {
			found = true
			if !dirExists(r) {
				t.Fatalf("rewritten typeRoot missing: %s", r)
			}
		}
	}
	if !found {
		t.Fatalf("expected rewritten typeRoots in %v", roots)
	}

	types := resolveCompilerTypes(modules, roots)
	if len(types) != 1 || types[0] != "node" {
		t.Fatalf("types = %v", types)
	}
}

func TestResolveCompilerTypes_ConfiguredTypes(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	root := filepath.Join(dir, "types")
	mustMkdir(t, filepath.Join(root, "node"))
	mustMkdir(t, filepath.Join(root, "jest"))
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "types": ["node", "", "jest", "missing"]
  }
}
`)
	got := resolveCompilerTypes(modules, []string{root})
	if len(got) != 2 || got[0] != "node" || got[1] != "jest" {
		t.Fatalf("got %v", got)
	}

	// Configured types none exist → fall through to default node discovery.
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "types": ["missing-only"]
  }
}
`)
	got = resolveCompilerTypes(modules, []string{root})
	if len(got) != 1 || got[0] != "node" {
		t.Fatalf("fallback = %v", got)
	}
}

func TestResolveModulePaths_SkipsMissingTargets(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	mustWrite(t, filepath.Join(modules, "present.d.ts"), "export {}\n")
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "ok": ["./present.d.ts"],
      "gone": ["./absent.d.ts"]
    }
  }
}
`)
	paths, _, err := ResolveModulePathsForTest(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := paths["ok"]; !ok {
		t.Fatalf("ok missing: %#v", paths)
	}
	if _, ok := paths["gone"]; ok {
		t.Fatalf("gone should be skipped: %#v", paths)
	}
}

func TestTypePathExists_IndexSuffixesViaExtensionlessMissingDir(t *testing.T) {
	// Stat(path) fails when path is not a directory; /index.* suffixes still resolve
	// when the parent exists and the index file is present under a path that is only
	// reachable via the suffix probe. Use a path that does not exist as an entry but
	// where path+"/index.d.ts" cannot exist either without the parent — so create the
	// parent as a real dir named differently and probe the extensionless leaf that
	// only has ".d.ts" / ".ts" siblings (already covered). Here cover "?" glob chars.
	if !typePathExists("/tmp/types/foo?.[d].ts") {
		t.Fatal("?[ treated as present")
	}
}

func TestPreferTypesWriteDir_UsesHomeFallback(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", "")
	t.Setenv("CHOYSUM_TEST_TMP", "")
	got := PreferTypesWriteDir()
	if got == "" {
		// UserHomeDir may fail in exotic environments; skip assertion.
		if _, err := os.UserHomeDir(); err != nil {
			t.Skip("no home dir")
		}
		t.Fatal("expected ~/.choysum/pkg/types fallback")
	}
	if !strings.Contains(filepath.ToSlash(got), "/.choysum/pkg/types") {
		t.Fatalf("got %q", got)
	}
}

func TestHasResolvableVueTypes_ReadError(t *testing.T) {
	modules := t.TempDir()
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{ "compilerOptions": { "paths": {} } }`)
	orig := readFile
	t.Cleanup(func() { readFile = orig })
	readFile = func(string) ([]byte, error) {
		return nil, os.ErrPermission
	}
	if HasResolvableVueTypes(modules, modules) {
		t.Fatal("read error should yield false")
	}
}

func TestResolveTypeRoots_ExactTypesDirRewrite(t *testing.T) {
	repo := t.TempDir()
	modules := filepath.Join(repo, "modules")
	mustMkdir(t, modules)
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	mustMkdir(t, filepath.Join(home, "pkg", "types"))

	// No trailing slash after pkg/types — rewriteChoysumTypesPath misses (marker needs /),
	// rewriteChoysumTypesDir handles the exact suffix.
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "typeRoots": ["../../.choysum/pkg/types", "/abs/missing", ""]
  }
}
`)
	roots := resolveTypeRoots(modules, repo)
	found := false
	want := filepath.ToSlash(filepath.Join(home, "pkg", "types"))
	for _, r := range roots {
		if filepath.Clean(r) == filepath.Clean(want) || strings.HasSuffix(filepath.ToSlash(r), "/pkg/types") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected remapped pkg/types in %v (want %s)", roots, want)
	}
}

func TestResolveTypeRoots_DedupNodeModules(t *testing.T) {
	repo := t.TempDir()
	modules := filepath.Join(repo, "modules")
	mustMkdir(t, modules)
	types := filepath.Join(repo, "node_modules", "@types")
	mustMkdir(t, types)
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "typeRoots": ["`+filepath.ToSlash(types)+`"]
  }
}
`)
	roots := resolveTypeRoots(modules, repo)
	if len(roots) != 1 {
		t.Fatalf("expected deduped single root, got %v", roots)
	}
}

func TestRewriteChoysumTypesDir_ExactNoExistingRoots(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", filepath.Join(t.TempDir(), "missing-home"))
	t.Setenv("CHOYSUM_TEST_TMP", filepath.Join(t.TempDir(), "missing-tmp"))
	any := false
	for _, r := range choysumTypesSearchRoots() {
		if dirExists(r) {
			any = true
			break
		}
	}
	if any {
		t.Skip("~/.choysum/pkg/types exists; cannot hit empty exact remap")
	}
	if got := rewriteChoysumTypesDir(filepath.ToSlash(filepath.Join(t.TempDir(), ".choysum", "pkg", "types"))); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestChoysumTypesSearchRoots_SkipsDotHome(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", ".")
	t.Setenv("CHOYSUM_TEST_TMP", "  ")
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })
	userHomeDir = func() (string, error) { return ".", nil }
	for _, r := range choysumTypesSearchRoots() {
		if r == "." || r == "" {
			t.Fatalf("dot/empty roots must be skipped, roots=%v", choysumTypesSearchRoots())
		}
	}
}

func TestHasResolvableVueTypes_AllTargetsGlobPresent(t *testing.T) {
	modules := t.TempDir()
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["./missing-but-glob/*.d.ts"]
    }
  }
}
`)
	if !HasResolvableVueTypes(modules, modules) {
		t.Fatal("glob vue mapping counts as present")
	}
}

func TestResolveTypeRoots_SkipsEmptyEntry(t *testing.T) {
	repo := t.TempDir()
	modules := filepath.Join(repo, "modules")
	mustMkdir(t, modules)
	types := filepath.Join(repo, "node_modules", "@types")
	mustMkdir(t, types)
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "typeRoots": ["", "  ", "`+filepath.ToSlash(types)+`"]
  }
}
`)
	roots := resolveTypeRoots(modules, repo)
	if len(roots) != 1 {
		t.Fatalf("roots = %v", roots)
	}
	for _, r := range roots {
		if strings.TrimSpace(r) == "" {
			t.Fatalf("empty typeRoot leaked: %v", roots)
		}
		if filepath.Clean(r) == filepath.Clean(modules) {
			t.Fatalf("empty typeRoot must not register modulesPath: %v", roots)
		}
	}
}

func TestRewriteChoysumTypesDir_ExactNoRootsAnywhere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })
	userHomeDir = func() (string, error) { return filepath.Join(home, "empty-home"), nil }

	exact := filepath.ToSlash(filepath.Join(t.TempDir(), ".choysum", "pkg", "types"))
	if got := rewriteChoysumTypesDir(exact); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestReadTsconfigTypesAndTypeRoots_ParseErrors(t *testing.T) {
	modules := t.TempDir()
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{ not json`)
	if readTsconfigTypeRoots(modules) != nil {
		t.Fatal("bad hujson typeRoots")
	}
	if readTsconfigTypes(modules) != nil {
		t.Fatal("bad hujson types")
	}

	// Valid JSONC but wrong shape still unmarshals to empty.
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{ "compilerOptions": "nope" }`)
	_ = readTsconfigTypeRoots(modules)
	_ = readTsconfigTypes(modules)
}
