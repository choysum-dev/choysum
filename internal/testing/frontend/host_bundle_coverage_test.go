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
		Router:              "router",
		PageMount:           "pm",
		OPage:               "opage",
		ChildView:           "child",
		AuthStore:           "auth",
		I18n:                "i18n",
		I18nStore:           "i18nStore",
		Registry:            "reg",
		Scope:               "scope",
		Permission:          "perm",
		PageComposable:      "pageComp",
		Vuedraggable:        "drag",
		TipTapVue3:          "tiptap-vue3",
		TipTapStarterKit:    "tiptap-starter",
		TipTapExtensionLink: "tiptap-link",
		DOMPurify:           "dompurify",
		UnovisVue:           "unovis-vue",
		UnovisTs:            "unovis-ts",
		RekaUI:              "reka-ui",
		LucideVueNext:       "lucide",
		VueSonner:           "sonner",
	}
	for _, tt := range []struct {
		path string
		want string
		ok   bool
	}{
		{"vue-router", "router", true},
		{"@choysum/page-mount", "pm", true},
		{"vuedraggable", "drag", true},
		{"@tiptap/vue-3", "tiptap-vue3", true},
		{"@tiptap/starter-kit", "tiptap-starter", true},
		{"@tiptap/extension-link", "tiptap-link", true},
		{"dompurify", "dompurify", true},
		{"@unovis/vue", "unovis-vue", true},
		{"@unovis/ts", "unovis-ts", true},
		{"reka-ui", "reka-ui", true},
		{"lucide-vue-next", "lucide", true},
		{"vue-sonner", "sonner", true},
		{"element-plus", "", false},
		{"echarts/core", "", false},
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
		// Legacy Page.vue / OPage.vue still stub for product pages; ChoyPage is a normal child.
		{"@/web/web/components/page/Page.vue", "/x/Page.vue", page, "opage", true},
		{"@/web/web/components/page/Page.vue", "/x/Page.vue", "", "opage", true},
		{"@/web/web/components/page/OPage.vue", "/x/OPage.vue", page, "opage", true},
		{"@/web/web/components/layout/ChoyPage.vue", "/x/ChoyPage.vue", page, "child", true},
		{"@/web/web/components/layout/OHeader.vue", "/x/OHeader.vue", page, "child", true},
		{"./OChatterMessageItem.vue", "/repo/modules/web/web/components/chatter/OChatterMessageItem.vue", "/repo/modules/web/web/components/chatter/OChatterMessageItem.test.ts", "", false},
		{"@/web/web/components/layout/OHeader.vue", "/x/OHeader.vue", "/other.ts", "", false},
		{"./Page.vue", "/repo/modules/web/web/components/layout/Page.vue", "/repo/modules/web/web/components/layout/ChoyPageIoMenu.test.ts", "", false},
		{"./OPage.vue", "/repo/modules/web/web/components/layout/OPage.vue", "/repo/modules/web/web/components/layout/ChoyPage.storeContext.mount.test.ts", "", false},
		{"./PartnerFormView.vue", "/x/PartnerFormView.vue", page, "child", true},
		{"@/web/web/components/view/ChoyFormView.vue", "/modules/web/web/components/view/ChoyFormView.vue", page, "", false},
		{"@/web/web/components/view/ChoyFormView.vue", "/modules/web/web/components/view/ChoyFormView.vue", "", "", false},
		{"./ChoyFormView.vue", "/tmp/ChoyFormView.vue", page, "", false},
		{"./ChoyFormView.vue", "/tmp/ChoyFormView.vue", "/other.ts", "", false},
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
		{"./stores/auth", "/modules/auth/web/stores/auth", "Register.vue", "auth", true},
		{"../stores/auth", "/modules/auth/web/stores/auth", "Logout.vue", "auth", true},
		{"../stores/auth", "/modules/auth/web/stores/auth", "Logout.mount.test.ts", "auth", true},
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
	var pkgCB, pageCB, childViewCB func(api.OnResolveArgs) (api.OnResolveResult, error)
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
				case `^(vue-router|@choysum/page-mount|vuedraggable|@tiptap/vue-3|@tiptap/starter-kit|@tiptap/extension-link|dompurify|@unovis/vue|@unovis/ts|reka-ui|lucide-vue-next|vue-sonner)$`:
					pkgCB = cb
				case `(?:^|/)(?:OPage|Page)\.vue$`:
					pageCB = cb
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

	if pkgCB == nil || pageCB == nil || childViewCB == nil || len(pathCBs) == 0 {
		t.Fatal("expected FE stub OnResolve callbacks to be registered")
	}
	for _, path := range []string{"vue-router", "@choysum/page-mount", "vuedraggable", "dompurify"} {
		res, err := pkgCB(api.OnResolveArgs{Path: path})
		if err != nil || res.Path == "" {
			t.Fatalf("package stub %q: %#v err=%v", path, res, err)
		}
	}
	pageRes, err := pageCB(api.OnResolveArgs{Path: "./Page.vue", Importer: "/modules/web/web/pages/TerminologyEditor.vue"})
	if err != nil || !strings.HasSuffix(pageRes.Path, "OPage.stub.vue") {
		t.Fatalf("page stub: %#v err=%v", pageRes, err)
	}
	legacyPageRes, err := pageCB(api.OnResolveArgs{Path: "./OPage.vue", Importer: "/modules/auth/web/pages/Login.vue"})
	if err != nil || !strings.HasSuffix(legacyPageRes.Path, "OPage.stub.vue") {
		t.Fatalf("legacy OPage stub: %#v err=%v", legacyPageRes, err)
	}
	pageSkip, err := pageCB(api.OnResolveArgs{Path: "./Page.vue", Importer: "/modules/web/web/components/layout/ChoyPageIoMenu.test.ts"})
	if err != nil || pageSkip.Path != "" {
		t.Fatalf("page skip for layout unit test: %#v err=%v", pageSkip, err)
	}
	pageSkipVue, err := pageCB(api.OnResolveArgs{Path: "./Page.vue", Importer: "/modules/web/web/components/layout/ChoyPageIoMenu.vue"})
	if err != nil || pageSkipVue.Path != "" {
		t.Fatalf("page skip for web component SFC: %#v err=%v", pageSkipVue, err)
	}
	childRes, err := childViewCB(api.OnResolveArgs{Path: "@/web/web/components/view/OFormView.vue", Importer: "/modules/partner_commercial/web/views/PartnerIdentifierFormView.vue"})
	if err != nil || !strings.HasSuffix(childRes.Path, "ChildView.stub.vue") {
		t.Fatalf("child view stub: %#v err=%v", childRes, err)
	}
	childSkip, err := childViewCB(api.OnResolveArgs{Path: "./ChoyFormView.vue", Importer: "/modules/web/web/components/view/ChoyFormView.route_reload.test.ts"})
	if err != nil || childSkip.Path != "" {
		t.Fatalf("child view skip for web unit test: %#v err=%v", childSkip, err)
	}
	choyFormKeep, err := childViewCB(api.OnResolveArgs{Path: "./ChoyFormView.vue", Importer: "/modules/web/web/kit.ts"})
	if err != nil || choyFormKeep.Path != "" {
		t.Fatalf("child view skip for kit ChoyFormView: %#v err=%v", choyFormKeep, err)
	}
	var pathHit, pageFromPage, pageFromTest bool
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
		pagePage, err := pathCB(api.OnResolveArgs{
			Path:       "@/web/web/components/page/Page.vue",
			Importer:   "/modules/web/web/pages/TerminologyEditor.vue",
			ResolveDir: dir,
		})
		if err != nil {
			t.Fatalf("page from page err: %v", err)
		}
		if strings.HasSuffix(pagePage.Path, "OPage.stub.vue") {
			pageFromPage = true
		}
		testPage, err := pathCB(api.OnResolveArgs{
			Path:       "./Page.vue",
			Importer:   "/modules/web/web/components/layout/ChoyPageIoMenu.test.ts",
			ResolveDir: "/modules/web/web/components/layout",
		})
		if err != nil {
			t.Fatalf("page from test err: %v", err)
		}
		if testPage.Path == "" {
			pageFromTest = true
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
	if !pageFromPage {
		t.Fatal("expected Page stub for page/view importers")
	}
	if !pageFromTest {
		t.Fatal("expected real Page for FE unit test importers")
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
