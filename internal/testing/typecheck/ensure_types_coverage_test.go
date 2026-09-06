// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esmresolver"
)

func TestNormalizeNPMVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"*", ""},
		{"workspace:foo", ""},
		{"file:../vue", ""},
		{"^3.5.35", "3.5.35"},
		{"~3.5.35", "3.5.35"},
		{"=3.5.35", "3.5.35"},
		{"v3.5.35", "3.5.35"},
		{">=3.5.35", "3.5.35"},
		{"<=3.5.35", "3.5.35"},
		{">3.5.35", "3.5.35"},
		{"npm:3.5.35", "3.5.35"},
		{"3.5.35 || 4.0.0", "3.5.35"},
		{"3.5.35-beta.1", "3.5.35-beta.1"},
		{"latest", ""},
		{"  3.5.35  ", "3.5.35"},
	}
	for _, tt := range cases {
		if got := normalizeNPMVersion(tt.in); got != tt.want {
			t.Fatalf("normalizeNPMVersion(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestReadPackageJSONVueVersion(t *testing.T) {
	if got := readPackageJSONVueVersion(filepath.Join(t.TempDir(), "missing.json")); got != "" {
		t.Fatalf("missing: %q", got)
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	writeFile(t, bad, "{not json")
	if got := readPackageJSONVueVersion(bad); got != "" {
		t.Fatalf("invalid json: %q", got)
	}

	deps := filepath.Join(t.TempDir(), "deps.json")
	writeFile(t, deps, `{"dependencies":{"vue":"^3.5.1"}}`)
	if got := readPackageJSONVueVersion(deps); got != "3.5.1" {
		t.Fatalf("deps: %q", got)
	}
	dev := filepath.Join(t.TempDir(), "dev.json")
	writeFile(t, dev, `{"devDependencies":{"vue":"~3.5.2"}}`)
	if got := readPackageJSONVueVersion(dev); got != "3.5.2" {
		t.Fatalf("dev: %q", got)
	}
	peer := filepath.Join(t.TempDir(), "peer.json")
	writeFile(t, peer, `{"peerDependencies":{"vue":"3.5.3"}}`)
	if got := readPackageJSONVueVersion(peer); got != "3.5.3" {
		t.Fatalf("peer: %q", got)
	}
	empty := filepath.Join(t.TempDir(), "empty.json")
	writeFile(t, empty, `{"dependencies":{"lodash":"1.0.0"}}`)
	if got := readPackageJSONVueVersion(empty); got != "" {
		t.Fatalf("no vue: %q", got)
	}
}

func TestResolveVueVersionFromPackageJSON(t *testing.T) {
	if got := resolveVueVersionFromPackageJSON(""); got != "" {
		t.Fatalf("empty root: %q", got)
	}
	if got := resolveVueVersionFromPackageJSON(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("missing root: %q", got)
	}

	modules := t.TempDir()
	makeDir(t, filepath.Join(modules, "web"))
	writeFile(t, filepath.Join(modules, "web", "package.json"), `{"peerDependencies":{"vue":"^3.5.35"}}`)
	if got := resolveVueVersionFromPackageJSON(modules); got != "3.5.35" {
		t.Fatalf("preferred web: %q", got)
	}

	modules2 := t.TempDir()
	makeDir(t, filepath.Join(modules2, "partner_bank"))
	writeFile(t, filepath.Join(modules2, "partner_bank", "package.json"), `{"dependencies":{"vue":"3.4.0"}}`)
	if got := resolveVueVersionFromPackageJSON(modules2); got != "3.4.0" {
		t.Fatalf("fallback module: %q", got)
	}

	// Prefer listed candidates; skip already-seen when scanning the rest.
	modules3 := t.TempDir()
	makeDir(t, filepath.Join(modules3, "web"))
	makeDir(t, filepath.Join(modules3, "auth"))
	makeDir(t, filepath.Join(modules3, "other"))
	// Preferred candidates have no vue; scan skips web/auth (seen) then hits other.
	writeFile(t, filepath.Join(modules3, "web", "package.json"), `{"dependencies":{}}`)
	writeFile(t, filepath.Join(modules3, "auth", "package.json"), `{"dependencies":{}}`)
	writeFile(t, filepath.Join(modules3, "other", "package.json"), `{"dependencies":{"vue":"3.5.10"}}`)
	if got := resolveVueVersionFromPackageJSON(modules3); got != "3.5.10" {
		t.Fatalf("scan skip seen: %q", got)
	}
}

func TestResolveVueVersionFromTypesDir(t *testing.T) {
	if got := resolveVueVersionFromTypesDir(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := resolveVueVersionFromTypesDir(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("missing: %q", got)
	}

	typesDir := t.TempDir()
	makeDir(t, filepath.Join(typesDir, "subdir"))
	writeFile(t, filepath.Join(typesDir, "esm.sh_vue@3.5.1_dist_vue.d.mts.d.ts"), "export {}\n")
	writeFile(t, filepath.Join(typesDir, "other.txt"), "x\n")
	if got := resolveVueVersionFromTypesDir(typesDir); got != "3.5.1" {
		t.Fatalf("incomplete anyVer: %q", got)
	}

	typesDir2 := t.TempDir()
	writeCompleteVueGraph(t, typesDir2, "3.5.35")
	if got := resolveVueVersionFromTypesDir(typesDir2); got != "3.5.35" {
		t.Fatalf("complete: %q", got)
	}
}

func TestFindCompleteVueEntry(t *testing.T) {
	if got := findCompleteVueEntry("", "3.5.35"); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := findCompleteVueEntry(t.TempDir(), ""); got != "" {
		t.Fatalf("empty ver: %q", got)
	}
	if got := findCompleteVueEntry(filepath.Join(t.TempDir(), "missing"), "3.5.35"); got != "" {
		t.Fatalf("missing dir: %q", got)
	}

	typesDir := t.TempDir()
	writeCompleteVueGraph(t, typesDir, "3.5.35")
	want := filepath.Join(typesDir, "esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts")
	if got := findCompleteVueEntry(typesDir, "3.5.35"); got != want {
		t.Fatalf("preferred: %q", got)
	}

	// Alternate name via directory scan (not in the preferred filename list).
	typesDir2 := t.TempDir()
	makeDir(t, filepath.Join(typesDir2, "0dir"))                                           // sorts before esm.*; covers IsDir continue
	writeFile(t, filepath.Join(typesDir2, "esm.sh_vue@3.5.350_extra.d.ts"), "export {}\n") // version boundary
	writeFile(t, filepath.Join(typesDir2, "nope.txt"), "x\n")
	for _, name := range []string{
		"esm.sh_@vue_runtime-dom@3.5.35_dist_runtime-dom.d.ts.d.ts",
		"esm.sh_@vue_runtime-core@3.5.35_dist_runtime-core.d.ts.d.ts",
		"esm.sh_@vue_reactivity@3.5.35_dist_reactivity.d.ts.d.ts",
	} {
		body := "export {}\n"
		if strings.Contains(name, "runtime-core") {
			body = "export type PropType<T> = any;\ndeclare function h(...args: any[]): any;\n"
		}
		if strings.Contains(name, "reactivity") {
			body = "export declare function toRef(...args: any[]): any;\n"
		}
		writeFile(t, filepath.Join(typesDir2, name), body)
	}
	alt := filepath.Join(typesDir2, "esm.sh_vue@3.5.35_index.d.ts")
	writeFile(t, alt, "export {}\n")
	if got := findCompleteVueEntry(typesDir2, "3.5.35"); got != alt {
		t.Fatalf("scan alt: %q", got)
	}
}

func TestEnsureVueTsconfigPath(t *testing.T) {
	if err := ensureVueTsconfigPath(t.TempDir(), ""); err != nil {
		t.Fatal(err)
	}
	modules := t.TempDir()
	entry := filepath.Join(t.TempDir(), "esm.sh_vue@1.d.ts")
	writeFile(t, entry, "export {}\n")
	if err := ensureVueTsconfigPath(modules, entry); err != nil {
		t.Fatal(err)
	}
	if _, ok := readModuleTSConfigPaths(modules)["vue"]; !ok {
		t.Fatal("expected vue path")
	}

	// Corrupt tsconfig → remove + retry success.
	modules2 := t.TempDir()
	writeFile(t, filepath.Join(modules2, "tsconfig.json"), "{ not json")
	if err := ensureVueTsconfigPath(modules2, entry); err != nil {
		t.Fatal(err)
	}

	// Retry failure: updateTsconfigPaths always fails.
	orig := updateTsconfigPaths
	t.Cleanup(func() { updateTsconfigPaths = orig })
	updateTsconfigPaths = func(string, []esmresolver.TypeFetchResult) error {
		return errors.New("write boom")
	}
	modules3 := t.TempDir()
	err := ensureVueTsconfigPath(modules3, entry)
	if err == nil || !strings.Contains(err.Error(), "write boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveVueAdjacentNodeVersion(t *testing.T) {
	if got := resolveVueAdjacentNodeVersion(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := resolveVueAdjacentNodeVersion(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("missing: %q", got)
	}
	typesDir := t.TempDir()
	makeDir(t, filepath.Join(typesDir, "subdir"))
	writeFile(t, filepath.Join(typesDir, "other.txt"), "x\n")
	if got := resolveVueAdjacentNodeVersion(typesDir); got != "" {
		t.Fatalf("dirs/files without node: %q", got)
	}
	writeFile(t, filepath.Join(typesDir, "esm.sh_@types_node@22.20.1_index.d.ts.d.ts"), "export {}\n")
	if got := resolveVueAdjacentNodeVersion(typesDir); got != "22.20.1" {
		t.Fatalf("got %q", got)
	}
}

func TestPurgeIncompleteVueTypeFetch(t *testing.T) {
	purgeIncompleteVueTypeFetch("", "3.5.35")
	purgeIncompleteVueTypeFetch(t.TempDir(), "")
	purgeIncompleteVueTypeFetch(filepath.Join(t.TempDir(), "missing"), "3.5.35")

	typesDir := t.TempDir()
	makeDir(t, filepath.Join(typesDir, "subdir"))
	writeFile(t, filepath.Join(typesDir, "vue@3.5.1.d.ts"), "export {}\n")
	// Version boundary: 3.5.10 must not be purged when targeting 3.5.1.
	keep := filepath.Join(typesDir, "esm.sh_vue@3.5.10_dist_vue.d.mts.d.ts")
	writeFile(t, keep, "export {}\n")
	hollow := filepath.Join(typesDir, "esm.sh_vue@3.5.1_dist_vue.d.mts.d.ts")
	writeFile(t, hollow, "export {}\n")
	sib := filepath.Join(typesDir, "esm.sh_@vue_runtime-core@3.5.1_dist_runtime-core.d.ts.d.ts")
	writeFile(t, sib, "export {}\n")
	purgeIncompleteVueTypeFetch(typesDir, "3.5.1")
	if _, err := os.Stat(hollow); !os.IsNotExist(err) {
		t.Fatal("hollow entry should be removed")
	}
	if _, err := os.Stat(sib); !os.IsNotExist(err) {
		t.Fatal("siblings should be removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("3.5.10 must survive")
	}

	// Complete graph is left in place (including vue@ver.d.ts package cache).
	typesDir2 := t.TempDir()
	writeCompleteVueGraph(t, typesDir2, "3.5.35")
	entry := filepath.Join(typesDir2, "esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts")
	pkgCache := filepath.Join(typesDir2, "vue@3.5.35.d.ts")
	writeFile(t, pkgCache, "export {}\n")
	purgeIncompleteVueTypeFetch(typesDir2, "3.5.35")
	if _, err := os.Stat(entry); err != nil {
		t.Fatal("complete entry must remain")
	}
	if _, err := os.Stat(pkgCache); err != nil {
		t.Fatal("complete graph must not delete vue@ver.d.ts package cache")
	}
}

func TestEnsureTypeAssets_TsconfigWriteAndStillMissing(t *testing.T) {
	origFetch := fetchTypeDefinition
	origUpdate := updateTsconfigPaths
	t.Cleanup(func() {
		fetchTypeDefinition = origFetch
		updateTsconfigPaths = origUpdate
	})

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts"]
    }
  }
}
`)

	t.Run("tsconfig write error", func(t *testing.T) {
		fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, typesDir, pkg, ver string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			p := filepath.Join(typesDir, "esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts")
			writeCompleteVueGraph(t, typesDir, "3.5.0")
			return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
		}
		updateTsconfigPaths = func(string, []esmresolver.TypeFetchResult) error {
			return errors.New("tsconfig boom")
		}
		err := ensureTypeAssets(context.Background(), io.Discard, modules, "demo")
		if err == nil || !strings.Contains(err.Error(), "tsconfig boom") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("still missing after successful write", func(t *testing.T) {
		fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, typesDir, pkg, ver string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			writeCompleteVueGraph(t, typesDir, "3.5.0")
			p := filepath.Join(typesDir, "esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts")
			return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
		}
		updateTsconfigPaths = func(path string, _ []esmresolver.TypeFetchResult) error {
			// Pretend write succeeded but left no resolvable vue mapping.
			return os.WriteFile(path, []byte(`{"compilerOptions":{"paths":{}}}`), 0o644)
		}
		err := ensureTypeAssets(context.Background(), nil, modules, "demo")
		if err == nil || !strings.Contains(err.Error(), "still missing") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestEnsureTypeAssets_DiscoversVueFromPackageJSON(t *testing.T) {
	origFetch := fetchTypeDefinition
	t.Cleanup(func() { fetchTypeDefinition = origFetch })

	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	modules := t.TempDir()
	makeDir(t, filepath.Join(modules, "web"))
	writeFile(t, filepath.Join(modules, "web", "package.json"), `{"peerDependencies":{"vue":"^3.5.35"}}`)

	fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, typesDir, pkg, ver string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
		if pkg != "vue" || ver != "3.5.35" {
			t.Fatalf("unexpected %s@%s", pkg, ver)
		}
		writeCompleteVueGraph(t, typesDir, ver)
		p := filepath.Join(typesDir, "esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts")
		return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
	}
	if err := ensureTypeAssets(context.Background(), io.Discard, modules, "web"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureNodeCompilerTypes_AdjacentVersion(t *testing.T) {
	orig := fetchTypeDefinition
	t.Cleanup(func() { fetchTypeDefinition = orig })

	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{"compilerOptions":{"paths":{}}}`)
	typesDir := t.TempDir()
	writeFile(t, filepath.Join(typesDir, "esm.sh_@types_node@18.19.0_index.d.ts.d.ts"), "export {}\n")

	fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, td, pkg, ver string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
		if pkg != "@types/node" || ver != "18.19.0" {
			t.Fatalf("unexpected %s@%s", pkg, ver)
		}
		p := filepath.Join(td, "esm.sh_@types_node@18.19.0_index.d.ts.d.ts")
		return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
	}
	if err := ensureNodeCompilerTypes(context.Background(), &http.Client{}, "https://esm.sh", typesDir, modules); err != nil {
		t.Fatal(err)
	}
}
