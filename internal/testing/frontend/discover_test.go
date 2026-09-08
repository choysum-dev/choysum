// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
	"github.com/evanw/esbuild/pkg/api"
)

func TestDiscoverAndScanIllegalFrontendMarks(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(filepath.Join(web, "node_modules", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "node_modules", "x", "skip.test.ts"), []byte("mount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(web, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "dist", "skip.test.ts"), []byte("mount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pure := filepath.Join(web, "pure.test.ts")
	if err := os.WriteFile(pure, []byte("import { describe, it } from 'vitest'\nit('ok', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsTest := filepath.Join(web, "legacy.test.js")
	if err := os.WriteFile(jsTest, []byte("import Comp from './Comp.vue'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "note.md"), []byte("mount(\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	illegal := filepath.Join(web, "bad.test.ts")
	illegalSrc := strings.Join([]string{
		"// @vitest-environment happy-dom",
		"import {",
		"  mount,",
		"} from '@vue/test-utils'",
		"import Comp from './Comp.vue'",
		"import './Side.vue'",
		"await import('./Dyn.vue')",
		"import {",
		"  something,",
		"} from 'happy-dom'",
		"mount(Comp)",
		"",
	}, "\n")
	if err := os.WriteFile(illegal, []byte(illegalSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverFrontendTests(repo, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("DiscoverFrontendTests = %#v", files)
	}

	hits, err := ScanIllegalFrontendMarks(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected illegal hits")
	}
	kinds := map[IllegalKind]bool{}
	for _, h := range hits {
		kinds[h.Kind] = true
	}
	for _, want := range []IllegalKind{IllegalDOMEnvironment, IllegalVTU, IllegalVueImport, IllegalDOMPackage} {
		if !kinds[want] {
			t.Fatalf("missing kind %s in %#v", want, hits)
		}
	}

	warn := FormatIllegalMarksWarn(hits, repo)
	if !strings.Contains(warn, "illegal FE unit mark") {
		t.Fatalf("warn = %q", warn)
	}
	ann := FormatIllegalMarksGitHubAnnotations(hits, repo)
	if !strings.Contains(ann, "::warning file=") {
		t.Fatalf("annotations = %q", ann)
	}

	if _, err := CheckIllegalFrontendMarks(repo, "demo", ScanModeError); err == nil {
		t.Fatal("expected error mode failure")
	}
	if hits2, err := CheckIllegalFrontendMarks(repo, "demo", ScanModeWarn); err != nil || len(hits2) == 0 {
		t.Fatalf("warn mode = %v hits=%d", err, len(hits2))
	}

	vueOnlyRepo := t.TempDir()
	vueOnlyWeb := filepath.Join(vueOnlyRepo, "modules", "vueonly", "web")
	if err := os.MkdirAll(vueOnlyWeb, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vueOnlyWeb, "sfc.test.ts"), []byte("import Comp from './Comp.vue'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vueHits, err := CheckIllegalFrontendMarks(vueOnlyRepo, "vueonly", ScanModeWarn)
	if err != nil || len(vueHits) == 0 {
		t.Fatalf("vue-only warn = %v hits=%#v", err, vueHits)
	}
	for _, h := range vueHits {
		if h.Kind != IllegalVueImport {
			t.Fatalf("unexpected hard-cut kind in vue-only inventory: %#v", h)
		}
	}
	if failHits := FilterHardCutFailHits(vueHits); len(failHits) != 0 {
		t.Fatalf("vue-only hard-cut hits = %#v", failHits)
	}
	if _, err := CheckIllegalFrontendMarks(vueOnlyRepo, "vueonly", ScanModeError); err != nil {
		t.Fatalf("vue-only ScanModeError must allow .vue imports: %v", err)
	}

	pureHits, err := ScanIllegalFrontendMarks([]string{pure})
	if err != nil {
		t.Fatal(err)
	}
	if len(pureHits) != 0 {
		t.Fatalf("pure file hits = %#v", pureHits)
	}

	sameLine := filepath.Join(web, "same_line.test.ts")
	if err := os.WriteFile(sameLine, []byte("import 'happy-dom'; mount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sameHits, err := ScanIllegalFrontendMarks([]string{sameLine})
	if err != nil {
		t.Fatal(err)
	}
	if len(sameHits) < 2 {
		t.Fatalf("same-line kinds = %#v", sameHits)
	}
}

func TestDiscoverFrontendTestsGuardsAndSkips(t *testing.T) {
	if _, err := DiscoverFrontendTests("", "x"); err == nil {
		t.Fatal("expected empty repo error")
	}
	if _, err := DiscoverFrontendTests(t.TempDir(), ""); err == nil {
		t.Fatal("expected empty app error")
	}
	repo := t.TempDir()
	files, err := DiscoverFrontendTests(repo, "missing")
	if err != nil || len(files) != 0 {
		t.Fatalf("missing app = %#v err=%v", files, err)
	}
	webAsFile := filepath.Join(repo, "modules", "fileapp", "web")
	if err := os.MkdirAll(filepath.Dir(webAsFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(webAsFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err = DiscoverFrontendTests(repo, "fileapp")
	if err != nil || len(files) != 0 {
		t.Fatalf("file web root = %#v err=%v", files, err)
	}

	prevStat := osStat
	osStat = func(string) (os.FileInfo, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { osStat = prevStat })
	if _, err := DiscoverFrontendTests(repo, "demo"); err == nil || !strings.Contains(err.Error(), "stat") {
		t.Fatalf("stat err = %v", err)
	}
	osStat = prevStat

	web := filepath.Join(repo, "modules", "walk", "web")
	locked := filepath.Join(web, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "a.test.ts"), []byte("test('x', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
		if _, err := DiscoverFrontendTests(repo, "walk"); err == nil {
			t.Fatal("expected walk error")
		}
	}

	prevAbs := filepathAbs
	filepathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { filepathAbs = prevAbs })
	okWeb := filepath.Join(repo, "modules", "abs", "web")
	if err := os.MkdirAll(okWeb, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(okWeb, "a.test.ts"), []byte("test('x', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverFrontendTests(repo, "abs"); err == nil {
		t.Fatal("expected abs error")
	}
}

func TestIsFrontendUnitTestFile(t *testing.T) {
	cases := map[string]bool{
		"a.test.ts":     true,
		"a.spec.tsx":    true,
		"a.test.js":     true,
		"a.spec.jsx":    true,
		"a.test.mjs":    true,
		"a.spec.cjs":    true,
		"a.test.md":     false,
		"a.ts":          false,
		"a.test.d.ts":   false,
		"foo.spec.d.ts": false,
	}
	for name, want := range cases {
		if got := isFrontendUnitTestFile(name); got != want {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
}

func TestScanIllegalFrontendMarksEdgeCases(t *testing.T) {
	hits, err := ScanIllegalFrontendMarks([]string{"", filepath.Join(t.TempDir(), "missing.test.ts")})
	if err != nil || len(hits) != 0 {
		t.Fatalf("empty/missing = %#v err=%v", hits, err)
	}

	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked.test.ts")
	if err := os.WriteFile(blocked, []byte("mount(x)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o644) })
	if runtime.GOOS != "windows" {
		if _, err := ScanIllegalFrontendMarks([]string{blocked}); err == nil {
			t.Fatal("expected read permission error")
		}
	}

	long := strings.Repeat("あ", 130) + " mount(x)"
	hits = scanIllegalContent("p.ts", "// @jest-environment jsdom\n"+long+"\n")
	if len(hits) < 2 {
		t.Fatalf("hits = %#v", hits)
	}
	for _, h := range hits {
		if h.Kind == IllegalVTU && !strings.HasSuffix(h.Snippet, "...") {
			t.Fatalf("expected truncated snippet, got %q", h.Snippet)
		}
	}

	multi := "import {\n  mount\n} from '@vue/test-utils'\n"
	hits = scanIllegalContent("m.ts", multi)
	vtuFromHits := 0
	for _, h := range hits {
		if h.Kind == IllegalVTU && strings.Contains(h.Snippet, "from '@vue/test-utils'") {
			vtuFromHits++
			if h.Line != 3 {
				t.Fatalf("expected from-clause line 3, got %#v", h)
			}
		}
	}
	if vtuFromHits != 1 {
		t.Fatalf("expected single VTU from hit, got %#v", hits)
	}

	// Same-line VTU import + mount( share IllegalVTU and exercise add() dedup.
	hits = scanIllegalContent("d.ts", "import { mount } from '@vue/test-utils'; mount(x)\n")
	if len(hits) != 1 || hits[0].Kind != IllegalVTU {
		t.Fatalf("dedup hits = %#v", hits)
	}
	hits = scanIllegalContent("choysum-mount.ts", "import { mount } from '@choysum/test-utils'\nmount(X)\n")
	if len(hits) != 0 {
		for _, h := range hits {
			if h.Kind == IllegalVTU {
				t.Fatalf("choysum mount must not flag IllegalVTU: %#v", hits)
			}
		}
	}
	// Package name only in a comment must not suppress IllegalVTU for bare mount().
	hits = scanIllegalContent("comment-only.ts", "// uses @choysum/test-utils someday\nmount(X)\n")
	if len(hits) != 1 || hits[0].Kind != IllegalVTU {
		t.Fatalf("comment-only choysum mention must still flag mount: %#v", hits)
	}
	// Unrelated named import from test-utils must not suppress bare mount().
	hits = scanIllegalContent("unrelated-import.ts", "import { flushPromises } from '@choysum/test-utils'\nmount(X)\n")
	if len(hits) != 1 || hits[0].Kind != IllegalVTU {
		t.Fatalf("unrelated choysum import must still flag mount: %#v", hits)
	}

	// Method calls like app.mount() must not be treated as VTU mount().
	if methodHits := scanIllegalContent("m.ts", "app.mount('#app')\nwrapper.mount()\n"); len(methodHits) != 0 {
		t.Fatalf("method mount false positives = %#v", methodHits)
	}

	var snippets []string
	addRegexHits("from './X.vue'", nil, reVueImport, IllegalVueImport, func(_ int, _ IllegalKind, snippet string) {
		snippets = append(snippets, snippet)
	})
	if len(snippets) != 1 || !strings.Contains(snippets[0], ".vue") {
		t.Fatalf("fallback snippets = %#v", snippets)
	}

	cjs := strings.Join([]string{
		`require('jsdom')`,
		`require("@vue/test-utils")`,
		"require(`./Comp.vue`)",
		"await import(`happy-dom`)",
		"await import(`@vue/test-utils`)",
		"import Comp from './Comp.vue?raw'",
		"from 'happy-dom/lib/index.js'",
		"from '@vue/test-utils/dist/vue-test-utils.esm-bundler.js'",
		"",
	}, "\n")
	cjsHits := scanIllegalContent("x.cjs", cjs)
	wantKinds := map[IllegalKind]bool{}
	for _, h := range cjsHits {
		wantKinds[h.Kind] = true
	}
	for _, kind := range []IllegalKind{IllegalDOMPackage, IllegalVTU, IllegalVueImport} {
		if !wantKinds[kind] {
			t.Fatalf("cjs/dynamic/query missing %s in %#v", kind, cjsHits)
		}
	}
}

func TestScanAppAndCheckErrorPropagation(t *testing.T) {
	if _, err := ScanAppIllegalFrontendMarks("", "x"); err == nil {
		t.Fatal("expected discover error")
	}
	if _, err := CheckIllegalFrontendMarks("", "x", ScanModeWarn); err == nil {
		t.Fatal("expected check discover error")
	}
}

func TestFormatAndRelativizeHelpers(t *testing.T) {
	if FormatIllegalMarksWarn(nil, "") != "" {
		t.Fatal("empty warn")
	}
	if FormatIllegalMarksGitHubAnnotations(nil, "") != "" {
		t.Fatal("empty annotations")
	}
	hits := []IllegalMark{{
		Path:    filepath.Join("C:", "abs", "weird,%path.test.ts"),
		Line:    2,
		Kind:    IllegalVTU,
		Snippet: "mount(x) 100%\r\nok",
	}}
	ann := FormatIllegalMarksGitHubAnnotations(hits, "")
	if !strings.Contains(ann, "%25") || strings.Contains(ann, "\r") {
		t.Fatalf("ann = %q", ann)
	}
	if !strings.Contains(ann, "file=") || !strings.Contains(ann, "%2C") {
		t.Fatalf("expected escaped comma in file=, got %q", ann)
	}
	if got := relativizeRepoPath("", "/tmp/x"); got == "" {
		t.Fatal("empty repo relativize")
	}
	if got := relativizeRepoPath(".", "/tmp/x"); !strings.Contains(got, "tmp") {
		t.Fatalf("dot repoRoot = %q", got)
	}
	repo := t.TempDir()
	inside := filepath.Join(repo, "modules", "a.test.ts")
	if got := relativizeRepoPath(repo+string(filepath.Separator), inside); !strings.HasPrefix(got, "modules/") {
		t.Fatalf("inside rel = %q", got)
	}
	if got := relativizeRepoPath(repo, filepath.Join(t.TempDir(), "out.ts")); strings.HasPrefix(got, "modules/") {
		t.Fatalf("outside should stay absolute-ish, got %q", got)
	}
	prevAbs := filepathAbs
	filepathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { filepathAbs = prevAbs })
	if got := relativizeRepoPath(repo, inside); !strings.HasPrefix(got, "modules/") {
		t.Fatalf("abs-fallback rel = %q", got)
	}
	if got := relativizeRepoPath("/tmp/child", "/tmp"); got != "/tmp" && got != "\\tmp" {
		// Rel yields ".."; fall through to cleaned path.
		if !strings.HasSuffix(got, "tmp") {
			t.Fatalf("abs-fail outside = %q", got)
		}
	}
}

func TestWarnIllegalFrontendMarks(t *testing.T) {
	warnIllegalFrontendMarks("", "x")

	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "a.test.ts"), []byte("import { mount } from '@vue/test-utils'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	warnIllegalFrontendMarks(repo, "demo")
	warnIllegalFrontendMarks(repo, "missing")
}

func TestBuildFrontendUnitBundleGuards(t *testing.T) {
	if _, err := BuildFrontendUnitBundle(BundleOptions{}); err == nil {
		t.Fatal("empty repo")
	}
	repo := t.TempDir()
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo}); err == nil {
		t.Fatal("empty entry")
	}
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: "x.ts"}); err == nil {
		t.Fatal("missing modules")
	}
	modulesFile := filepath.Join(repo, "modules")
	if err := os.WriteFile(modulesFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: "x.ts"}); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("modules file err = %v", err)
	}
	_ = os.Remove(modulesFile)
	if err := os.MkdirAll(filepath.Join(repo, "modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(repo, "bad-entry.ts")
	if err := os.WriteFile(entry, []byte("import './missing-mod'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: entry, WorkingDir: repo}); err == nil {
		t.Fatal("expected esbuild failure")
	}

	prevMkdir := osMkdirAll
	osMkdirAll = func(string, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osMkdirAll = prevMkdir })
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: entry}); err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("mkdir err = %v", err)
	}
	osMkdirAll = prevMkdir

	prevBuild := esbuildBuild
	esbuildBuild = func(api.BuildOptions) api.BuildResult {
		return api.BuildResult{Warnings: []api.Message{{Text: "soft-warn"}}}
	}
	t.Cleanup(func() { esbuildBuild = prevBuild })
	prevRead := osReadFile
	osReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { osReadFile = prevRead })
	if _, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: entry, Outfile: filepath.Join(repo, "out.js")}); err == nil || !strings.Contains(err.Error(), "read outfile") {
		t.Fatalf("read err = %v", err)
	}
	osReadFile = prevRead
	esbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		_ = os.WriteFile(opts.Outfile, []byte("(() => {})();"), 0o644)
		return api.BuildResult{Warnings: []api.Message{{Text: "soft-warn"}}}
	}
	res, err := BuildFrontendUnitBundle(BundleOptions{RepoRoot: repo, EntryPath: entry, Outfile: filepath.Join(repo, "warned.js")})
	if err != nil || len(res.Warnings) != 1 {
		t.Fatalf("warn bundle = %#v err=%v", res, err)
	}
	esbuildBuild = prevBuild
}

func TestWriteFrontendTestsEntryEscapes(t *testing.T) {
	if err := WriteFrontendTestsEntry("", nil); err == nil {
		t.Fatal("empty out")
	}
	outDir := t.TempDir()
	out := filepath.Join(outDir, "entry.ts")
	pathWithQuote := filepath.Join(outDir, "subdir", "o'brian.test.ts")
	if err := os.MkdirAll(filepath.Dir(pathWithQuote), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrontendTestsEntry(out, []string{pathWithQuote}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	absWant, absErr := filepath.Abs(pathWithQuote)
	if absErr != nil {
		t.Fatal(absErr)
	}
	encoded, encErr := json.Marshal(filepath.ToSlash(absWant))
	if encErr != nil {
		t.Fatal(encErr)
	}
	if !strings.Contains(string(raw), "import "+string(encoded)) {
		t.Fatalf("entry = %s want absolute import %s", raw, encoded)
	}

	prevMkdir := osMkdirAll
	osMkdirAll = func(string, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osMkdirAll = prevMkdir })
	if err := WriteFrontendTestsEntry(out, nil); err == nil || !strings.Contains(err.Error(), "mkdir entry") {
		t.Fatalf("mkdir entry err = %v", err)
	}
	osMkdirAll = prevMkdir

	prevMarshal := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, os.ErrInvalid }
	t.Cleanup(func() { jsonMarshal = prevMarshal })
	if err := WriteFrontendTestsEntry(out, []string{"a.ts"}); err == nil || !strings.Contains(err.Error(), "encode import path") {
		t.Fatalf("marshal err = %v", err)
	}
	jsonMarshal = prevMarshal

	prevAbs := filepathAbs
	filepathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { filepathAbs = prevAbs })
	if err := WriteFrontendTestsEntry(out, []string{"a.ts"}); err == nil || !strings.Contains(err.Error(), "resolve test file path") {
		t.Fatalf("abs resolve err = %v", err)
	}
	filepathAbs = prevAbs

	prevWrite := osWriteFile
	osWriteFile = func(string, []byte, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osWriteFile = prevWrite })
	if err := WriteFrontendTestsEntry(out, []string{pathWithQuote}); err == nil || !strings.Contains(err.Error(), "write entry") {
		t.Fatalf("write err = %v", err)
	}
}

func TestBuildFrontendUnitBundleAndChoysumtestFixture(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "testdata", "fe_qjs")
	if _, err := os.Stat(filepath.Join(fixtureDir, "math.test.ts")); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	outDir := t.TempDir()
	for _, name := range []string{"math.ts", "math.test.ts"} {
		src := filepath.Join(fixtureDir, name)
		dst := filepath.Join(outDir, name)
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	testFile := filepath.Join(outDir, "math.test.ts")
	entry := filepath.Join(outDir, "entry.ts")
	if err := WriteFrontendTestsEntry(entry, []string{testFile}); err != nil {
		t.Fatal(err)
	}
	bundle, err := BuildFrontendUnitBundle(BundleOptions{
		RepoRoot:  repoRoot,
		EntryPath: entry,
		Sourcemap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.MapPath == "" {
		t.Fatal("expected linked sourcemap path")
	}
	if !strings.Contains(bundle.JS, "add sums two numbers") {
		snip := bundle.JS
		if len(snip) > 400 {
			snip = snip[:400]
		}
		t.Fatalf("unexpected bundle:\n%s", snip)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
		{FileName: bundle.JSPath, Content: bundle.JS},
	}); err != nil {
		t.Fatalf("Load: %v", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("engine type %T", engine)
	}
	val := qjs.Ctx.Eval(`(async () => {
  const r = await globalThis.__choysum_test_run__();
  return JSON.stringify(r);
})()`, quickjs.EvalAwait(true))
	defer val.Free()
	if val.IsException() {
		t.Fatalf("eval report: %v", qjs.Ctx.Exception())
	}
	raw := val.String()
	var report struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	}
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("parse report %q: %v", raw, err)
	}
	if report.Total < 2 || report.Failed != 0 || report.Passed != report.Total {
		t.Fatalf("report = %+v raw=%s", report, raw)
	}
}

func TestScanCoverageProbeBanned(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web", "testing")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(web, "CoverageProbe.vue")
	if err := os.WriteFile(probe, []byte("<template><div/></template>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(web, "coverage_probe.test.ts")
	if err := os.WriteFile(testFile, []byte("import CoverageProbe from './CoverageProbe.vue';\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	hits, err := ScanAppIllegalFrontendMarks(repo, "demo")
	if err != nil {
		t.Fatal(err)
	}
	var sawImport, sawFile bool
	for _, h := range hits {
		if h.Kind != IllegalCoverageProbe {
			continue
		}
		if h.Path == testFile && strings.Contains(h.Snippet, "CoverageProbe.vue") {
			sawImport = true
		}
		if strings.HasSuffix(filepath.Base(h.Path), "CoverageProbe.vue") {
			sawFile = true
		}
	}
	if !sawImport || !sawFile {
		t.Fatalf("expected CoverageProbe import+file hits, got %#v", hits)
	}
	if _, err := CheckIllegalFrontendMarks(repo, "demo", ScanModeError); err == nil {
		t.Fatal("expected hard-cut failure for CoverageProbe")
	}
}

func TestCoverageProbeImportIgnoresSimilarBasenames(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(web, "similar.test.ts")
	if err := os.WriteFile(testFile, []byte("import X from './PartnerCoverageProbe.vue';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := ScanAppIllegalFrontendMarks(repo, "demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.Kind == IllegalCoverageProbe {
			t.Fatalf("PartnerCoverageProbe.vue must not be flagged: %#v", hits)
		}
	}
}
