// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchTSConfigPathPattern(t *testing.T) {
	if _, ok := matchTSConfigPathPattern("", "vue"); ok {
		t.Fatal("empty pattern")
	}
	if reps, ok := matchTSConfigPathPattern("vue", "vue"); !ok || len(reps) != 0 {
		t.Fatalf("exact match %#v %v", reps, ok)
	}
	if _, ok := matchTSConfigPathPattern("vue", "react"); ok {
		t.Fatal("exact mismatch")
	}

	reps, ok := matchTSConfigPathPattern("@/*", "@/foo/bar")
	if !ok || len(reps) != 1 || reps[0] != "foo/bar" {
		t.Fatalf("@/* %#v %v", reps, ok)
	}
	if _, ok := matchTSConfigPathPattern("pre/*", "other/x"); ok {
		t.Fatal("prefix miss")
	}

	reps, ok = matchTSConfigPathPattern("a/*/c", "a/b/c")
	if !ok || len(reps) != 1 || reps[0] != "b" {
		t.Fatalf("mid star %#v %v", reps, ok)
	}
	if _, ok := matchTSConfigPathPattern("a/*/c", "a/b/x"); ok {
		t.Fatal("suffix miss")
	}
	if _, ok := matchTSConfigPathPattern("a/*/c", "a/nope"); ok {
		t.Fatal("segment miss")
	}

	// Empty middle segment between stars (pattern "a**b" → parts a,"",b).
	reps, ok = matchTSConfigPathPattern("a**b", "axb")
	if !ok || len(reps) != 2 || reps[0] != "" || reps[1] != "x" {
		t.Fatalf("empty mid segment %#v %v", reps, ok)
	}

	reps, ok = matchTSConfigPathPattern("*/*", "foo/bar")
	if !ok || len(reps) != 2 || reps[0] != "foo" || reps[1] != "bar" {
		t.Fatalf("two stars %#v %v", reps, ok)
	}

	reps, ok = matchTSConfigPathPattern("pre*mid*suf", "preXXmidYYsuf")
	if !ok || len(reps) != 2 || reps[0] != "XX" || reps[1] != "YY" {
		t.Fatalf("multi mid %#v %v", reps, ok)
	}
	if _, ok := matchTSConfigPathPattern("pre*mid*suf", "preXXmissing"); ok {
		t.Fatal("middle segment miss")
	}
}

func TestApplyPathPatternReplacements(t *testing.T) {
	if got := applyPathPatternReplacements("plain", []string{"x"}); got != "plain" {
		t.Fatal(got)
	}
	if got := applyPathPatternReplacements("a/*/c", nil); got != "a/*/c" {
		t.Fatal(got)
	}
	if got := applyPathPatternReplacements("a/*/c", []string{"b"}); got != "a/b/c" {
		t.Fatal(got)
	}
	if got := applyPathPatternReplacements("*/*", []string{"x"}); got != "x/*" {
		t.Fatalf("leftover star: %q", got)
	}
	if got := applyPathPatternReplacements("*/*", []string{"x", "y"}); got != "x/y" {
		t.Fatal(got)
	}
}

func TestHasAnyExistingTypeAsset(t *testing.T) {
	modules := t.TempDir()
	if hasAnyExistingTypeAsset([]string{"", "  "}, modules) {
		t.Fatal("empty entries")
	}

	rel := "types/vue.d.ts"
	makeDir(t, filepath.Join(modules, "types"))
	writeFile(t, filepath.Join(modules, filepath.FromSlash(rel)), "export {}\n")
	if !hasAnyExistingTypeAsset([]string{rel}, modules) {
		t.Fatal("relative existing")
	}

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	cached := filepath.Join(home, "pkg", "types", "esm.sh_vue@1", "index.d.ts")
	makeDir(t, filepath.Dir(cached))
	writeFile(t, cached, "export {}\n")
	rewritable := "../../.choysum/pkg/types/esm.sh_vue@1/index.d.ts"
	if !hasAnyExistingTypeAsset([]string{rewritable}, modules) {
		t.Fatal("rewrite should find CHOYSUM_HOME asset")
	}

	globDir := filepath.Join(modules, "glob")
	makeDir(t, globDir)
	writeFile(t, filepath.Join(globDir, "a.d.ts"), "export {}\n")
	if !hasAnyExistingTypeAsset([]string{filepath.ToSlash(filepath.Join(globDir, "*.d.ts"))}, modules) {
		t.Fatal("glob file match")
	}

	onlyDir := filepath.Join(modules, "onlydir")
	makeDir(t, onlyDir)
	if hasAnyExistingTypeAsset([]string{filepath.ToSlash(filepath.Join(onlyDir, "*"))}, modules) {
		t.Fatal("glob dirs only should not count")
	}

	if hasAnyExistingTypeAsset([]string{filepath.Join(modules, "missing.d.ts")}, modules) {
		t.Fatal("missing")
	}

	abs := filepath.Join(modules, "abs.d.ts")
	writeFile(t, abs, "export {}\n")
	if !hasAnyExistingTypeAsset([]string{abs}, modules) {
		t.Fatal("abs file")
	}

	// Unclosed '[' makes filepath.Glob return an error on many platforms.
	if hasAnyExistingTypeAsset([]string{filepath.Join(modules, "bad[")}, modules) {
		t.Fatal("glob error should not count as existing")
	}
}

func TestShouldSuggestTypeFetchFromOutput_MoreBranches(t *testing.T) {
	if shouldSuggestTypeFetchFromOutput("") || shouldSuggestTypeFetchFromOutput("  ") {
		t.Fatal("empty")
	}
	if !shouldSuggestTypeFetchFromOutput("error TS7016: Could not find a declaration file") {
		t.Fatal("TS7016")
	}
	if !shouldSuggestTypeFetchFromOutput("Cannot find module 'x' or its corresponding type declaration.") {
		t.Fatal("cannot find + type declaration")
	}
	if shouldSuggestTypeFetchFromOutput("Cannot find module 'x'") {
		t.Fatal("module alone")
	}
}

func TestFormatTypecheckFailureWithGuidance(t *testing.T) {
	err := formatTypecheckFailureWithGuidance("auth", errors.New("boom"), "TS2307 fail", false)
	if !strings.Contains(err.Error(), "type-fetch auth") {
		t.Fatalf("%v", err)
	}
	err = formatTypecheckFailureWithGuidance("  ", errors.New("boom"), "TS2307", false)
	if !strings.Contains(err.Error(), "type-fetch <app>") {
		t.Fatalf("%v", err)
	}
	err = formatTypecheckFailureWithGuidance("auth", errors.New("boom"), "unrelated", false)
	if strings.Contains(err.Error(), "recommended action") {
		t.Fatalf("no guidance: %v", err)
	}
	err = formatTypecheckFailureWithGuidance("auth", errors.New("boom"), "unrelated", true)
	if !strings.Contains(err.Error(), "recommended action") {
		t.Fatalf("warned: %v", err)
	}
}

func TestResolveTypePathEntries_Pattern(t *testing.T) {
	paths := map[string][]string{
		"@/*": {"./@/*"},
		"vue": {"./vue.d.ts"},
	}
	entries, ok := resolveTypePathEntries(paths, "vue")
	if !ok || len(entries) != 1 {
		t.Fatalf("%v %v", entries, ok)
	}
	entries, ok = resolveTypePathEntries(paths, "@/components/X")
	if !ok || len(entries) != 1 || entries[0] != "./@/components/X" {
		t.Fatalf("%v %v", entries, ok)
	}
	if _, ok := resolveTypePathEntries(paths, ""); ok {
		t.Fatal("empty name")
	}
	if _, ok := resolveTypePathEntries(paths, "missing"); ok {
		t.Fatal("no mapping")
	}
}

func TestWarnMissingTypeAssetsPrecheck_NilStderr(t *testing.T) {
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{"compilerOptions":{"paths":{}}}`)
	warned, err := warnMissingTypeAssetsPrecheck(nil, modules, "auth")
	if err != nil || warned {
		t.Fatalf("warned=%v err=%v", warned, err)
	}
}

func TestWarnMissingTypeAssetsPrecheck_TruncatesPreview(t *testing.T) {
	modules := t.TempDir()
	makeDir(t, filepath.Join(modules, "auth", "service"))
	writeFile(t, filepath.Join(modules, "auth", "service", "index.ts"), "export {}\n")
	writeFile(t, filepath.Join(modules, "auth", "package.json"), `{
  "dependencies": {
    "pkg-a": "1.0.0",
    "pkg-b": "1.0.0",
    "pkg-c": "1.0.0",
    "pkg-d": "1.0.0"
  }
}`)
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "pkg-a": ["./types/a.d.ts"],
      "pkg-b": ["./types/b.d.ts"],
      "pkg-c": ["./types/c.d.ts"],
      "pkg-d": ["./types/d.d.ts"]
    }
  }
}
`)
	var stderr strings.Builder
	warned, err := warnMissingTypeAssetsPrecheck(&stderr, modules, "auth")
	if err != nil || !warned {
		t.Fatalf("warned=%v err=%v", warned, err)
	}
	got := stderr.String()
	if !strings.Contains(got, ", ...") {
		t.Fatalf("expected truncated preview, got %q", got)
	}
}

func TestMissingTypeAssetModules_NoTsconfig(t *testing.T) {
	modules := t.TempDir()
	if got := missingTypeAssetModules(modules, []string{"lodash"}); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestReadModuleTSConfigPaths_ParseFailures(t *testing.T) {
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{ unterminated`)
	if got := readModuleTSConfigPaths(modules); got != nil {
		t.Fatalf("bad hujson should return nil, got %#v", got)
	}

	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": "not-an-object"
  }
}`)
	if got := readModuleTSConfigPaths(modules); got != nil {
		t.Fatalf("bad paths shape should return nil, got %#v", got)
	}
}

func TestMatchTSConfigPathPattern_TrailingStarOnly(t *testing.T) {
	reps, ok := matchTSConfigPathPattern("prefix*", "prefixTail")
	if !ok || len(reps) != 1 || reps[0] != "Tail" {
		t.Fatalf("prefix* %#v %v", reps, ok)
	}
}

func TestMissingTypeAssetModules_PatternAndExisting(t *testing.T) {
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "@scope/*": ["./types/@scope/*.d.ts"],
      "present": ["./types/present.d.ts"],
      "absent": ["./types/absent.d.ts"]
    }
  }
}
`)
	makeDir(t, filepath.Join(modules, "types", "@scope"))
	writeFile(t, filepath.Join(modules, "types", "@scope", "pkg.d.ts"), "export {}\n")
	writeFile(t, filepath.Join(modules, "types", "present.d.ts"), "export {}\n")

	missing := missingTypeAssetModules(modules, []string{"present", "absent", "@scope/pkg", "unmapped"})
	if len(missing) != 1 || missing[0] != "absent" {
		t.Fatalf("missing = %v", missing)
	}
}
