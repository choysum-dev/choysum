// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"io/fs"
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

func TestCheck_ScopeAll_DefaultQuickJS(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeAll,
	})
	if err != nil {
		t.Fatalf("Check with default QuickJSCoder: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Diagnostics)
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

func TestCheck_DoesNotCloseCallerCoder(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	inner := vue.NewGoldenCoder(vueGoldenDir(t))
	c := &closeTrackingCoder{inner: inner}
	_, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeAll,
		Coder:       c,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.closed {
		t.Fatal("Check closed caller-supplied Coder")
	}
}

type closeTrackingCoder struct {
	inner  vue.Coder
	closed bool
}

func (c *closeTrackingCoder) CreateServiceScript(path, source string, opts vue.CodegenOptions) (vue.ServiceScript, error) {
	return c.inner.CreateServiceScript(path, source, opts)
}

func (c *closeTrackingCoder) Close() error {
	c.closed = true
	return nil
}

func TestCollectVueOverlayPaths(t *testing.T) {
	if got := collectVueOverlayPaths("/m", "demo", nil, true); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	modules := "/repo/modules"
	app := "demo"
	overlays := map[string]string{
		"/repo/modules/demo/web/App.vue":         "<script setup lang=\"ts\"></script>",
		"/repo/modules/demo/web/skip.spec.vue":   "x",
		"/repo/modules/demo/service/X.vue":       "x",
		"/repo/modules/demo/web/ui.ts":           "x",
		"/repo/modules/other/web/O.vue":          "x",
		"/repo/modules/demo/web/__tests__/T.vue": "x",
		"/REPO/MODULES/demo/web/Case.vue":        "x",
	}
	got := collectVueOverlayPaths(modules, app, overlays, true)
	if len(got) != 1 || !strings.HasSuffix(got[0], "web/App.vue") {
		t.Fatalf("%v", got)
	}
	gotCI := collectVueOverlayPaths(modules, app, overlays, false)
	foundCase := false
	for _, p := range gotCI {
		if strings.Contains(strings.ToLower(p), "web/case.vue") {
			foundCase = true
		}
	}
	if !foundCase {
		t.Fatalf("case-insensitive overlay miss: %v", gotCI)
	}
	merged := mergeVuePaths([]string{"/disk/A.vue", "", "/disk/A.vue"}, append(got, "", got[0]))
	if len(merged) != 2 {
		t.Fatalf("%v", merged)
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
	overlays := BuiltInVueAmbientOverlays(dir, dir)
	// No resolvable vue types → vite + subpath + vue shim + directives + vue module stub.
	if len(overlays) != 5 {
		t.Fatalf("want vite+subpath+vue shim+directives+vue stub, got %d", len(overlays))
	}
}

type errCoder struct{}

func (errCoder) CreateServiceScript(string, string, vue.CodegenOptions) (vue.ServiceScript, error) {
	return vue.ServiceScript{}, errors.New("coder boom")
}

type emptySourceCoder struct {
	inner vue.Coder
}

func (c emptySourceCoder) CreateServiceScript(path, source string, opts vue.CodegenOptions) (vue.ServiceScript, error) {
	s, err := c.inner.CreateServiceScript(path, source, opts)
	if err != nil {
		return s, err
	}
	s.SourceContent = ""
	return s, nil
}

func TestPrepareVueOverlays_ErrorsAndOverlaySource(t *testing.T) {
	if _, _, err := prepareVueOverlays(nil, nil, "", nil); err == nil {
		t.Fatal("nil coder")
	}
	dir := t.TempDir()
	vuePath := filepath.Join(dir, "Missing.vue")
	if _, _, err := prepareVueOverlays(vue.NewGoldenCoder(vueGoldenDir(t)), []string{vuePath}, dir, nil); err == nil {
		t.Fatal("missing disk source")
	}
	norm := normalizePathKey(vuePath)
	overlays := map[string]string{norm: "<script setup lang=\"ts\"></script>\n"}
	if _, _, err := prepareVueOverlays(errCoder{}, []string{vuePath}, dir, overlays); err == nil || !strings.Contains(err.Error(), "coder boom") {
		t.Fatalf("err = %v", err)
	}

	fixture := string(mustReadFixture(t, "script_setup_ok.vue"))
	baseDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	known := filepath.Join(baseDir, "script_setup_ok.vue")
	mustWrite(t, known, fixture)
	// Build an unclean path without filepath.Join (Join Clean's ".." away).
	unclean := dir + string(filepath.Separator) + "nested" + string(filepath.Separator) + ".." + string(filepath.Separator) + "nested" + string(filepath.Separator) + "script_setup_ok.vue"
	if filepath.ToSlash(unclean) == normalizePathKey(unclean) {
		t.Fatal("test setup: unclean path collapsed; cannot exercise ToSlash overlay key")
	}
	overlaysSlash := map[string]string{filepath.ToSlash(unclean): fixture}
	got, scripts, err := prepareVueOverlays(emptySourceCoder{inner: vue.NewGoldenCoder(vueGoldenDir(t))}, []string{unclean}, dir, overlaysSlash)
	if err != nil {
		t.Fatal(err)
	}
	if scripts[normalizePathKey(unclean)].SourceContent == "" {
		t.Fatal("SourceContent should be filled from src when coder leaves it empty")
	}
	if got[normalizePathKey(unclean)] == "" {
		t.Fatal("missing overlay content")
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "vue", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRemapDiagnostics_Helpers(t *testing.T) {
	diags := []Diagnostic{{File: "/x.ts", Start: 1, Length: 1}}
	if got := remapDiagnostics(diags, nil); len(got) != 1 {
		t.Fatal("empty scripts passthrough")
	}
	src := "a\nb\nc"
	tmp := t.TempDir()
	vueDisk := filepath.Join(tmp, "A.vue")
	mustWrite(t, vueDisk, src)

	scripts := map[string]vue.ServiceScript{
		normalizePathKey(vueDisk + ".ts"): {
			SourceContent: src,
			Mappings: []vue.SpanMapping{
				{SourceStart: 2, SourceEnd: 3, GeneratedStart: 10, GeneratedEnd: 11, Verification: true},
			},
		},
	}
	got := remapDiagnostics([]Diagnostic{{
		File:   vueDisk + ".ts",
		Start:  10,
		Length: 1,
		Line:   99,
		Column: 99,
	}}, scripts)
	if len(got) != 1 || got[0].File != normalizePathKey(vueDisk) || got[0].Start != 2 || got[0].Line != 2 {
		t.Fatalf("%#v", got)
	}

	// Mapped but SourceContent empty → fall back to disk line/col.
	scriptsNoSrc := map[string]vue.ServiceScript{
		normalizePathKey(vueDisk + ".ts"): {
			Mappings: []vue.SpanMapping{
				{SourceStart: 2, SourceEnd: 3, GeneratedStart: 10, GeneratedEnd: 11, Verification: true},
			},
		},
	}
	got = remapDiagnostics([]Diagnostic{{File: vueDisk + ".ts", Start: 10, Length: 1}}, scriptsNoSrc)
	if got[0].Line != 2 {
		t.Fatalf("disk fallback %#v", got[0])
	}

	// Mapped but neither SourceContent nor disk → clear line/col.
	scriptsMissing := map[string]vue.ServiceScript{
		normalizePathKey("/no/disk/A.vue.ts"): {
			Mappings: []vue.SpanMapping{
				{SourceStart: 2, SourceEnd: 3, GeneratedStart: 10, GeneratedEnd: 11, Verification: true},
			},
		},
	}
	got = remapDiagnostics([]Diagnostic{{File: "/no/disk/A.vue.ts", Start: 10, Length: 1, Line: 3}}, scriptsMissing)
	if got[0].Line != 0 || got[0].Column != 0 || got[0].File != normalizePathKey("/no/disk/A.vue") {
		t.Fatalf("%#v", got[0])
	}

	// Lookup via filepath.ToSlash when normalize key misses (unclean path).
	uncleanVue := tmp + string(filepath.Separator) + "sub" + string(filepath.Separator) + ".." + string(filepath.Separator) + "A.vue"
	scriptsSlash := map[string]vue.ServiceScript{
		filepath.ToSlash(uncleanVue): {
			SourceContent: src,
			Mappings: []vue.SpanMapping{
				{SourceStart: 0, SourceEnd: 1, GeneratedStart: 1, GeneratedEnd: 2, Verification: true},
			},
		},
	}
	got = remapDiagnostics([]Diagnostic{{File: uncleanVue, Start: 1, Length: 1}}, scriptsSlash)
	if got[0].File != normalizePathKey(uncleanVue) {
		t.Fatalf("%#v", got[0])
	}

	// .vue.ts diagnostic with script keyed only under .vue path.
	scriptsVueOnly := map[string]vue.ServiceScript{
		normalizePathKey(vueDisk): {
			SourceContent: src,
			Mappings: []vue.SpanMapping{
				{SourceStart: 2, SourceEnd: 3, GeneratedStart: 10, GeneratedEnd: 11, Verification: true},
			},
		},
	}
	got = remapDiagnostics([]Diagnostic{{File: vueDisk + ".ts", Start: 10, Length: 1}}, scriptsVueOnly)
	if got[0].File != normalizePathKey(vueDisk) || got[0].Start != 2 {
		t.Fatalf("%#v", got[0])
	}

	// Unmapped diagnostic still strips .vue.ts suffix, but clears generated coords.
	got = remapDiagnostics([]Diagnostic{{File: vueDisk + ".ts", Start: 999, Length: 1, Line: 7}}, scripts)
	if got[0].File != normalizePathKey(vueDisk) || got[0].Line != 0 || got[0].Column != 0 || got[0].Start != 0 {
		t.Fatalf("%#v", got)
	}

	line, col, ok := lineColumnFromBytes(nil, 0)
	if ok || line != 0 || col != 0 {
		t.Fatal("empty bytes")
	}
	line, col, ok = lineColumnFromBytes([]byte("ab"), -1)
	if ok {
		t.Fatal("negative")
	}
	line, col, ok = lineColumnFromBytes([]byte("ab"), 100)
	if !ok || line != 1 || col != 3 {
		t.Fatalf("clamp %d %d %v", line, col, ok)
	}
	if _, _, ok := lineColumnFromFile("/no/such/file", 0); ok {
		t.Fatal("missing file")
	}
	if _, _, ok := lineColumnFromFile(vueDisk, -1); ok {
		t.Fatal("negative pos")
	}
	if line, col, ok := lineColumnFromFile(vueDisk, 2); !ok || line != 2 || col != 1 {
		t.Fatalf("disk line %d %d %v", line, col, ok)
	}
}

func TestCollectModulesWebVuePaths_WalkEntryError(t *testing.T) {
	modules := t.TempDir()
	mustMkdir(t, filepath.Join(modules, "demo", "web"))
	orig := walkModulesWebVueDir
	t.Cleanup(func() { walkModulesWebVueDir = orig })
	walkModulesWebVueDir = func(root string, fn fs.WalkDirFunc) error {
		return fn(filepath.Join(root, "broken.vue"), nil, errors.New("walk entry"))
	}
	if got := collectModulesWebVuePaths(modules); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCollectModulesWebVuePaths(t *testing.T) {
	if got := collectModulesWebVuePaths(filepath.Join(t.TempDir(), "missing")); len(got) != 0 {
		t.Fatalf("missing modules root should return nil, got %v", got)
	}

	modules := t.TempDir()
	mustMkdir(t, filepath.Join(modules, ".choysum"))
	mustMkdir(t, filepath.Join(modules, "tmp"))
	mustMkdir(t, filepath.Join(modules, "no-web", "service"))
	mustMkdir(t, filepath.Join(modules, "demo", "web", "ui"))
	mustWrite(t, filepath.Join(modules, "demo", "web", "ui", "App.vue"), "<template></template>\n")
	mustWrite(t, filepath.Join(modules, "demo", "web", "ui", "skip.spec.vue"), "<template></template>\n")
	mustMkdir(t, filepath.Join(modules, "demo", "web", "__tests__"))
	mustWrite(t, filepath.Join(modules, "demo", "web", "__tests__", "Hidden.vue"), "<template></template>\n")
	mustMkdir(t, filepath.Join(modules, "demo", "web", "node_modules", "pkg"))
	mustWrite(t, filepath.Join(modules, "demo", "web", "node_modules", "pkg", "X.vue"), "<template></template>\n")

	got := collectModulesWebVuePaths(modules)
	if len(got) != 1 || !strings.HasSuffix(got[0], "web/ui/App.vue") {
		t.Fatalf("got %v", got)
	}
}

func TestResolveVueCoder_AbsError(t *testing.T) {
	orig := absPath
	t.Cleanup(func() { absPath = orig })
	absPath = func(string) (string, error) { return "", errors.New("abs boom") }
	if _, err := resolveVueCoder(Options{VueGoldenDir: "x"}); err == nil || !strings.Contains(err.Error(), "abs boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_VueGoldenDirError(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	orig := absPath
	t.Cleanup(func() { absPath = orig })
	absPath = func(p string) (string, error) {
		if strings.TrimSpace(p) == "relative/golden" {
			return "", errors.New("abs boom")
		}
		return orig(p)
	}
	_, err := Check(t.Context(), Options{
		ModulesPath:  modules,
		RepoRoot:     repo,
		App:          "demo",
		Scope:        ScopeAll,
		VueGoldenDir: "relative/golden",
	})
	if err == nil || !strings.Contains(err.Error(), "abs boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_VueCoderError(t *testing.T) {
	repo, modules := fixtureRoots(t, "vue_check_ok")
	_, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeAll,
		Coder:       errCoder{},
	})
	if err == nil || !strings.Contains(err.Error(), "coder boom") {
		t.Fatalf("err = %v", err)
	}
}
