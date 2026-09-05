// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

func vueGoldenDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", "vue", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestCheck_VueScriptSetupOk(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	res, err := Check(t.Context(), Options{
		ModulesPath:  modules,
		RepoRoot:     repo,
		App:          "demo",
		Scope:        ScopeAll,
		VueGoldenDir: vueGoldenDir(t),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Diagnostics)
	}
}

func TestCheck_VueTemplateErrorRemaps(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_err")
	res, err := Check(t.Context(), Options{
		ModulesPath:  modules,
		RepoRoot:     repo,
		App:          "demo",
		Scope:        ScopeAll,
		VueGoldenDir: vueGoldenDir(t),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.HasErrors() {
		t.Fatal("expected template type errors")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Category != "error" {
			continue
		}
		if strings.Contains(d.File, "template_unknown.vue") {
			found = true
			if d.Line <= 0 {
				t.Fatalf("expected remapped line/column, got %#v", d)
			}
			// unknownVar is in the template around line 6 of the fixture.
			if d.Line < 4 {
				t.Fatalf("remap looks wrong (still generated coords?): %#v", d)
			}
			src, err := os.ReadFile(d.File)
			if err != nil {
				t.Fatal(err)
			}
			if d.Start < 0 || d.Start >= len(src) {
				t.Fatalf("remapped start out of source range: %#v (srcLen=%d)", d, len(src))
			}
		}
	}
	if !found {
		t.Fatalf("expected diagnostic on template_unknown.vue, got %#v", res.Diagnostics)
	}
}

func TestCheck_VueImportChild(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_import")
	res, err := Check(t.Context(), Options{
		ModulesPath:  modules,
		RepoRoot:     repo,
		App:          "demo",
		Scope:        ScopeAll,
		VueGoldenDir: vueGoldenDir(t),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Diagnostics)
	}
}

func TestCheck_ScopeAll_RequiresCoder(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	_, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeAll,
	})
	if err == nil || !strings.Contains(err.Error(), "VueGoldenDir") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_VueWithExplicitCoder(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeAll,
		Coder:       vue.NewGoldenCoder(vueGoldenDir(t)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HasErrors() {
		t.Fatalf("%+v", res.Diagnostics)
	}
}

func TestRewriteVueRootsAndAmbient(t *testing.T) {
	got := rewriteVueRootsToProgramPaths([]string{"/a.ts", "/b.vue"})
	if len(got) != 2 || got[1] != "/b.vue.ts" {
		t.Fatalf("%v", got)
	}
	if _, ok := fromVueProgramPath("/b.vue"); ok {
		t.Fatal("plain vue must not match")
	}
	dir := t.TempDir()
	overlays := BuiltInVueAmbientOverlays(dir)
	if len(overlays) != 3 {
		t.Fatalf("want vite+subpath+vue shim, got %d", len(overlays))
	}
}
