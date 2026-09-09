// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestFeUnitPackageAndPathStubMatchers(t *testing.T) {
	stubs := feUnitStubPaths{
		ElementPlus:    "ep",
		Icons:          "icons",
		Router:         "router",
		PageMount:      "pm",
		OPage:          "opage",
		ChildView:      "child",
		AuthStore:      "auth",
		I18n:           "i18n",
		I18nStore:      "i18nStore",
		Registry:       "reg",
		Scope:          "scope",
		Permission:     "perm",
		PageComposable: "pageComp",
		Vicons:         "vicons",
		VueEcharts:     "vchart",
		Vuedraggable:   "drag",
		Echarts:        "echarts",
	}
	for _, tt := range []struct {
		path string
		want string
		ok   bool
	}{
		{"element-plus", "ep", true},
		{"@element-plus/icons-vue", "icons", true},
		{"@vicons/material", "vicons", true},
		{"vue-router", "router", true},
		{"@choysum/page-mount", "pm", true},
		{"vue-echarts", "vchart", true},
		{"vuedraggable", "drag", true},
		{"echarts/core", "echarts", true},
		{"other", "", false},
	} {
		got, ok := feUnitPackageStubPath(tt.path, stubs)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("%q: got (%q,%v) want (%q,%v)", tt.path, got, ok, tt.want, tt.ok)
		}
	}

	page := "/repo/modules/auth/web/pages/Login.vue"
	view := "/repo/modules/partner/web/views/PartnerListView.vue"
	testImporter := "/repo/modules/web/web/stores/i18nStore.test.ts"
	cases := []struct {
		p, joined, importer, want string
		ok                        bool
	}{
		{"@/web/web/components/OPage.vue", "/x/OPage.vue", page, "opage", true},
		{"@/web/web/components/OPage.vue", "/x/OPage.vue", "", "opage", true},
		{"@/web/web/components/layout/OHeader.vue", "/x/OHeader.vue", page, "child", true},
		{"./OChatterMessageItem.vue", "/repo/modules/web/web/components/chatter/OChatterMessageItem.vue", "/repo/modules/web/web/components/chatter/OChatterMessageItem.test.ts", "", false},
		{"@/web/web/components/layout/OHeader.vue", "/x/OHeader.vue", "/other.ts", "", false},
		{"./OPage.vue", "/repo/modules/web/web/components/page/OPage.vue", "/repo/modules/web/web/components/page/OPage.mapping.test.ts", "", false},
		{"./PartnerFormView.vue", "/x/PartnerFormView.vue", page, "child", true},
		{"./PartnerListView.vue", "/x/PartnerListView.vue", page, "child", true},
		{"./ModuleKanbanView.vue", "/x/ModuleKanbanView.vue", view, "child", true},
		{"./PartnerFormView.vue", "/x/PartnerFormView.vue", "/other.ts", "", false},
		{"@/web/web/stores/registry", "/web/web/stores/registry", page, "reg", true},
		{"./storeScopeManager", "/x/storeScopeManager", page, "scope", true},
		{"@/web/web/stores/registry", "/web/web/stores/registry", "/modules/web/web/stores/registry.test.ts", "", false},
		{"@/core/web/stores/registry", "/modules/core/web/stores/registry", page, "", false},
		{"./storeScopeManager", "/x/storeScopeManager", "/other.ts", "scope", true},
		{"./storeScopeManager", "/x/storeScopeManager", "/modules/web/web/stores/scope.test.ts", "", false},
		{"@/web/web/stores/registry", "/modules/web/web/stores/registry", "/modules/web/web/controllers/formController.ts", "reg", true},
		{"@/auth/web/composables/usePermission", "/x/usePermission", page, "perm", true},
		{"../composables/usePermission.ts", "/x/usePermission.ts", page, "perm", true},
		{"@/auth/web/composables/usePermission", "/x/usePermission", "/other.ts", "", false},
		{"@/web/web/composables/usePageContext", "/x/composables/usePageContext", page, "pageComp", true},
		{"@/web/web/composables/useListView", "/x/composables/useListView", view, "pageComp", true},
		{"@/auth/web/stores/auth", "/modules/auth/web/stores/auth", page, "auth", true},
		{"../stores/auth", "/modules/auth/web/stores/auth", page, "auth", true},
		{"./stores/auth", "/modules/auth/web/stores/auth", "Login.vue", "auth", true},
		{"@/auth/web/stores/auth/index.ts", "/modules/auth/web/stores/auth/index.ts", page, "auth", true},
		{"@/web/web/i18n", "/web/web/i18n/index", page, "i18n", true},
		{"@/web/web/i18n", "/web/web/i18n/index", "/other.ts", "", false},
		{"@/web/web/stores/i18nStore", "/stores/i18nStore", page, "i18nStore", true},
		{"@/web/web/stores/i18nStore", "/stores/i18nStore", testImporter, "", false},
		{"@/web/web/stores/i18nStore/foo", "/stores/i18nStore/foo", page, "i18nStore", true},
		{"@/web/web/stores/i18nStore/foo", "/stores/i18nStore/foo", "/other.ts", "", false},
		{"unrelated", "/unrelated", page, "", false},
		{"@/web/web/components/readme.md", "/web/web/components/readme.md", page, "", false},
	}
	for _, tt := range cases {
		got, ok := feUnitPathStubPath(tt.p, tt.joined, tt.importer, stubs)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("p=%q joined=%q importer=%q: got (%q,%v) want (%q,%v)",
				tt.p, tt.joined, tt.importer, got, ok, tt.want, tt.ok)
		}
	}
}

func TestBuildFrontendVueHostBundle_FEStubsAndExtras(t *testing.T) {
	repo := vueHostRepoRoot(t)
	dir := t.TempDir()
	entry := filepath.Join(dir, "entry_stubs.ts")
	if err := os.WriteFile(entry, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	extraStub := filepath.Join(dir, "extra.js")
	if err := os.WriteFile(extraStub, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	prevRel := hostFilepathRel
	hostFilepathRel = func(string, string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { hostFilepathRel = prevRel })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: entry,
		Outfile:   filepath.Join(dir, "rel.js"),
		CacheDir:  t.TempDir(),
	}); err == nil || !strings.Contains(err.Error(), "modules relpath") {
		t.Fatalf("rel err: %v", err)
	}
	hostFilepathRel = prevRel

	prevMarshal := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, os.ErrInvalid }
	t.Cleanup(func() { jsonMarshal = prevMarshal })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: entry,
		Outfile:   filepath.Join(dir, "json.js"),
		CacheDir:  t.TempDir(),
	}); err == nil || !strings.Contains(err.Error(), "tsconfig") {
		t.Fatalf("json err: %v", err)
	}
	jsonMarshal = prevMarshal

	prevBuild := esbuildBuild
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		if _, ok := opts.Alias["extra-pkg"]; !ok {
			t.Fatalf("extra alias missing: %#v", opts.Alias)
		}
		if _, ok := opts.Alias[""]; ok {
			t.Fatal("empty alias key must be skipped")
		}
		_ = os.WriteFile(opts.Outfile, []byte("/*disabled*/"), 0o644)
		return api.BuildResult{}
	}
	t.Cleanup(func() { esbuildBuild = prevBuild })
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:              repo,
		EntryPath:             entry,
		Outfile:               filepath.Join(dir, "disabled.js"),
		CacheDir:              t.TempDir(),
		DisableDefaultFEStubs: true,
		ExtraStubAliases:      map[string]string{"extra-pkg": extraStub, "": "skip", " ": "skip", "left-empty": ""},
	}); err != nil {
		t.Fatal(err)
	}

	// Drive FE stub OnResolve callbacks without network by invoking Setup on a capture PluginBuild.
	var pkgCB, opageCB, childViewCB func(api.OnResolveArgs) (api.OnResolveResult, error)
	var pathCBs []func(api.OnResolveArgs) (api.OnResolveResult, error)
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		pb := api.PluginBuild{
			InitialOptions: &opts,
			Resolve:        func(string, api.ResolveOptions) api.ResolveResult { return api.ResolveResult{} },
			OnStart:        func(func() (api.OnStartResult, error)) {},
			OnEnd:          func(func(*api.BuildResult) (api.OnEndResult, error)) {},
			OnLoad:         func(api.OnLoadOptions, func(api.OnLoadArgs) (api.OnLoadResult, error)) {},
			OnDispose:      func(func()) {},
			OnResolve: func(o api.OnResolveOptions, cb func(api.OnResolveArgs) (api.OnResolveResult, error)) {
				switch o.Filter {
				case `^(element-plus|@element-plus/icons-vue|@vicons/material|vue-router|@choysum/page-mount|vue-echarts|vuedraggable|echarts(/.*)?)$`:
					pkgCB = cb
				case `OPage\.vue$`:
					opageCB = cb
				case `(FormView|ListView|KanbanView)\.vue$`:
					childViewCB = cb
				case `.*`:
					pathCBs = append(pathCBs, cb)
				}
			},
		}
		for _, p := range opts.Plugins {
			if p.Setup != nil {
				p.Setup(pb)
			}
		}
		_ = os.WriteFile(opts.Outfile, []byte("/*stubs*/"), 0o644)
		return api.BuildResult{}
	}
	if _, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:  repo,
		EntryPath: entry,
		Outfile:   filepath.Join(dir, "stubs.js"),
		CacheDir:  t.TempDir(),
	}); err != nil {
		t.Fatal(err)
	}
	esbuildBuild = prevBuild

	if pkgCB == nil || opageCB == nil || childViewCB == nil || len(pathCBs) == 0 {
		t.Fatal("expected FE stub OnResolve callbacks to be registered")
	}
	for _, path := range []string{"element-plus", "@element-plus/icons-vue", "vue-router", "@choysum/page-mount"} {
		res, err := pkgCB(api.OnResolveArgs{Path: path})
		if err != nil || res.Path == "" {
			t.Fatalf("package stub %q: %#v err=%v", path, res, err)
		}
	}
	opageRes, err := opageCB(api.OnResolveArgs{Path: "./OPage.vue", Importer: "/modules/auth/web/pages/Login.vue"})
	if err != nil || !strings.HasSuffix(opageRes.Path, "OPage.stub.vue") {
		t.Fatalf("opage stub: %#v err=%v", opageRes, err)
	}
	opageSkip, err := opageCB(api.OnResolveArgs{Path: "./OPage.vue", Importer: "/modules/web/web/components/page/OPage.mapping.test.ts"})
	if err != nil || opageSkip.Path != "" {
		t.Fatalf("opage skip for mapping test: %#v err=%v", opageSkip, err)
	}
	opageSkipVue, err := opageCB(api.OnResolveArgs{Path: "./OPage.vue", Importer: "/modules/web/web/components/page/OPageIoMenu.vue"})
	if err != nil || opageSkipVue.Path != "" {
		t.Fatalf("opage skip for web component SFC: %#v err=%v", opageSkipVue, err)
	}
	childRes, err := childViewCB(api.OnResolveArgs{Path: "@/web/web/components/view/OFormView.vue", Importer: "/modules/partner_commercial/web/views/PartnerIdentifierFormView.vue"})
	if err != nil || !strings.HasSuffix(childRes.Path, "ChildView.stub.vue") {
		t.Fatalf("child view stub: %#v err=%v", childRes, err)
	}
	childSkip, err := childViewCB(api.OnResolveArgs{Path: "./OFormView.vue", Importer: "/modules/web/web/components/view/OFormView.route_reload.test.ts"})
	if err != nil || childSkip.Path != "" {
		t.Fatalf("child view skip for web unit test: %#v err=%v", childSkip, err)
	}
	var pathHit, opageFromPage, opageFromTest bool
	for _, pathCB := range pathCBs {
		pathRes, err := pathCB(api.OnResolveArgs{
			Path:       "@/web/web/stores/registry",
			Importer:   "/modules/auth/web/pages/Login.vue",
			ResolveDir: dir,
		})
		if err != nil {
			t.Fatalf("path stub err: %v", err)
		}
		if strings.HasSuffix(pathRes.Path, "store_registry.js") {
			pathHit = true
		}
		pageOPage, err := pathCB(api.OnResolveArgs{
			Path:       "@/web/web/components/page/OPage.vue",
			Importer:   "/modules/auth/web/pages/Login.vue",
			ResolveDir: dir,
		})
		if err != nil {
			t.Fatalf("opage from page err: %v", err)
		}
		if strings.HasSuffix(pageOPage.Path, "OPage.stub.vue") {
			opageFromPage = true
		}
		testOPage, err := pathCB(api.OnResolveArgs{
			Path:       "./OPage.vue",
			Importer:   "/modules/web/web/components/page/OPage.mapping.test.ts",
			ResolveDir: "/modules/web/web/components/page",
		})
		if err != nil {
			t.Fatalf("opage from test err: %v", err)
		}
		if testOPage.Path == "" {
			opageFromTest = true
		}
		// Relative join branch + miss path.
		miss, err := pathCB(api.OnResolveArgs{Path: "./nope.ts", Importer: "/x.ts", ResolveDir: dir})
		if err != nil {
			t.Fatalf("path miss err: %v", err)
		}
		_ = miss
	}
	if !pathHit {
		t.Fatal("expected FE path stub to resolve registry")
	}
	if !opageFromPage {
		t.Fatal("expected OPage stub for page/view importers")
	}
	if !opageFromTest {
		t.Fatal("expected real OPage for FE unit test importers")
	}
}

func TestScanCoverageProbeFilesEdges(t *testing.T) {
	repo := t.TempDir()
	hits, err := scanCoverageProbeFiles(repo+" ", " missing ")
	if err != nil || hits != nil {
		t.Fatalf("missing web: %#v err=%v", hits, err)
	}

	webFile := filepath.Join(repo, "modules", "fileapp", "web")
	if err := os.MkdirAll(filepath.Dir(webFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(webFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err = scanCoverageProbeFiles(repo, "fileapp")
	if err != nil || hits != nil {
		t.Fatalf("file web: %#v err=%v", hits, err)
	}

	prevStat := osStat
	osStat = func(string) (os.FileInfo, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { osStat = prevStat })
	if _, err := scanCoverageProbeFiles(repo, "x"); err == nil || !strings.Contains(err.Error(), "coverage-probe walk") {
		t.Fatalf("stat err: %v", err)
	}
	osStat = prevStat

	web := filepath.Join(repo, "modules", "demo", "web", "testing")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	skipDir := filepath.Join(web, "node_modules")
	if err := os.MkdirAll(skipDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skipDir, "CoverageProbe.vue"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(web, "CoverageProbe.vue")
	if err := os.WriteFile(probe, []byte("<template/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevAbs := filepathAbs
	filepathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { filepathAbs = prevAbs })
	hits, err = scanCoverageProbeFiles(repo, "demo")
	if err != nil || len(hits) != 1 || hits[0].Path != probe {
		t.Fatalf("abs fallback hits=%#v err=%v", hits, err)
	}
	filepathAbs = prevAbs

	prevWalk := filepathWalkDir
	filepathWalkDir = func(string, fs.WalkDirFunc) error { return os.ErrPermission }
	t.Cleanup(func() { filepathWalkDir = prevWalk })
	if _, err := scanCoverageProbeFiles(repo, "demo"); err == nil || !strings.Contains(err.Error(), "coverage-probe walk") {
		t.Fatalf("walk err: %v", err)
	}
	filepathWalkDir = prevWalk

	// walkErr path: callback receives non-nil walkErr
	filepathWalkDir = func(_ string, fn fs.WalkDirFunc) error {
		return fn(web, &fakeDirEntry{name: "x", dir: false}, os.ErrPermission)
	}
	if _, err := scanCoverageProbeFiles(repo, "demo"); err == nil {
		t.Fatal("expected walkErr propagation")
	}
	filepathWalkDir = prevWalk
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f *fakeDirEntry) Name() string               { return f.name }
func (f *fakeDirEntry) IsDir() bool                { return f.dir }
func (f *fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f *fakeDirEntry) Info() (fs.FileInfo, error) { return nil, os.ErrInvalid }

func TestScanAppIllegalFrontendMarksErrors(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(web, "a.test.ts")
	if err := os.WriteFile(testFile, []byte("test('x', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		if err := os.Chmod(testFile, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(testFile, 0o644) })
		if _, err := ScanAppIllegalFrontendMarks(repo, "demo"); err == nil {
			t.Fatal("expected read error from ScanIllegalFrontendMarks")
		}
		_ = os.Chmod(testFile, 0o644)
	}

	prevWalk := filepathWalkDir
	filepathWalkDir = func(string, fs.WalkDirFunc) error { return os.ErrPermission }
	t.Cleanup(func() { filepathWalkDir = prevWalk })
	if _, err := ScanAppIllegalFrontendMarks(repo, "demo"); err == nil {
		t.Fatal("expected probe walk error")
	}
}
