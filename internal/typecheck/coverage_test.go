// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/buke/typescript-go-internal/v7/pkg/diagnostics"
	"github.com/buke/typescript-go-internal/v7/pkg/locale"
)

func TestCollectRootFiles_Validation(t *testing.T) {
	if _, err := CollectRootFiles("  ", "demo", ScopeService); !errors.Is(err, ErrModulesPathRequired) {
		t.Fatalf("err = %v", err)
	}
	if _, err := CollectRootFiles(t.TempDir(), "", ScopeService); !errors.Is(err, ErrAppRequired) {
		t.Fatalf("err = %v", err)
	}
	if _, err := CollectRootFiles(t.TempDir(), "missing", ScopeService); !errors.Is(err, ErrNoRootFiles) {
		t.Fatalf("err = %v", err)
	}
	modules := t.TempDir()
	mustMkdir(t, filepath.Join(modules, "demo"))
	if _, err := CollectRootFiles(modules, "demo", Scope(99)); !errors.Is(err, ErrUnsupportedScope) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectRootFiles_EmptyApp(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, app)
	if _, err := CollectRootFiles(modules, "demo", ScopeService); !errors.Is(err, ErrNoRootFiles) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectRootFiles_ServiceExtras(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "service", "node_modules", "pkg"))
	mustMkdir(t, filepath.Join(app, "service", "__tests__"))
	mustMkdir(t, filepath.Join(app, "service", "test"))
	mustWrite(t, filepath.Join(app, "env.d.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "skip.gen.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "ok.spec.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "ok.test.d.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "a.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "a.ts"), "export {};\n") // duplicate path via add
	mustWrite(t, filepath.Join(app, "service", "types.d.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "skip.gen.d.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "node_modules", "pkg", "x.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "__tests__", "t.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "test", "helper.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "readme.md"), "x\n")

	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(files, "\n")
	if !strings.Contains(joined, "env.d.ts") || !strings.Contains(joined, "types.d.ts") || !strings.Contains(joined, "a.ts") {
		t.Fatalf("missing expected files: %v", files)
	}
	for _, ban := range []string{"skip.gen.ts", "ok.spec.ts", "ok.test.d.ts", "skip.gen.d.ts", "node_modules/pkg/x.ts", "__tests__/t.ts", "test/helper.ts"} {
		if strings.Contains(joined, ban) {
			t.Fatalf("unexpected %s in %v", ban, files)
		}
	}
}

func TestCollectRootFiles_ReadDirError(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, app)
	orig := readDir
	t.Cleanup(func() { readDir = orig })
	readDir = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("readdir boom")
	}
	if _, err := CollectRootFiles(modules, "demo", ScopeService); err == nil || !strings.Contains(err.Error(), "readdir boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectRootFiles_AbsFallback(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "service"))
	mustWrite(t, filepath.Join(app, "service", "a.ts"), "export {};\n")
	orig := absPath
	t.Cleanup(func() { absPath = orig })
	absPath = func(string) (string, error) {
		return "", errors.New("abs boom")
	}
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %v", files)
	}
}

func TestCheck_MoreBranches(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	coreTypes := filepath.Join(modules, "core", "types")
	mustMkdir(t, coreTypes)
	mustWrite(t, filepath.Join(coreTypes, "$choysum.d.ts"), "export {};\n")
	mustMkdir(t, filepath.Join(repo, "as_dir"))
	mustWrite(t, filepath.Join(repo, "as_dir", "x"), "x")

	res, err := Check(nil, Options{ // nil ctx
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Overlays: map[string]string{
			filepath.ToSlash(filepath.Join(modules, "demo", "service", "math.ts")): "export const answer: number = 3;\n",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected: %#v", res.Diagnostics)
	}
	if !fileExists(filepath.Join(coreTypes, "$choysum.d.ts")) {
		t.Fatal("expected ambient")
	}
	if fileExists(filepath.Join(repo, "as_dir")) {
		t.Fatal("directory must not count as fileExists")
	}
	files := []string{"a", "b"}
	files = appendUniqueSlash(files, "a")
	if len(files) != 2 {
		t.Fatalf("dedupe failed: %v", files)
	}
}

func TestCheck_OverlayAmbient(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	ambient := filepath.ToSlash(filepath.Join(modules, "core", "types", "$choysum.d.ts"))
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Overlays: map[string]string{
			ambient: "export {};\n",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected: %#v", res.Diagnostics)
	}
	if fileExists(filepath.Join(modules, "core", "types", "$choysum.d.ts")) {
		t.Fatal("ambient should exist only via overlay")
	}
}

func TestCheck_AbsErrors(t *testing.T) {
	orig := absPath
	t.Cleanup(func() { absPath = orig })
	calls := 0
	absPath = func(string) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("modules abs")
		}
		return "/x", nil
	}
	if _, err := Check(t.Context(), Options{ModulesPath: "m", RepoRoot: "r", App: "a"}); err == nil || !strings.Contains(err.Error(), "modules abs") {
		t.Fatalf("err = %v", err)
	}
	calls = 0
	absPath = func(string) (string, error) {
		calls++
		if calls == 1 {
			return "/m", nil
		}
		return "", errors.New("repo abs")
	}
	if _, err := Check(t.Context(), Options{ModulesPath: "m", RepoRoot: "r", App: "a"}); err == nil || !strings.Contains(err.Error(), "repo abs") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_CollectError(t *testing.T) {
	dir := t.TempDir()
	if _, err := Check(t.Context(), Options{ModulesPath: dir, RepoRoot: dir, App: "missing"}); !errors.Is(err, ErrNoRootFiles) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Check(ctx, Options{ModulesPath: "m", RepoRoot: "r", App: "a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_CanceledDuringDiagnostics(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	ctx, cancel := context.WithCancel(t.Context())
	orig := absPath
	t.Cleanup(func() { absPath = orig })
	calls := 0
	absPath = func(path string) (string, error) {
		abs, err := orig(path)
		calls++
		if calls >= 2 {
			cancel()
		}
		return abs, err
	}
	if _, err := Check(ctx, Options{ModulesPath: modules, RepoRoot: repo, App: "demo"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectDiagnostics_CancelPaths(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	opts, err := BuildCompilerOptions(modules, repo)
	if err != nil {
		t.Fatal(err)
	}
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	program := buildProgram(newHost(modules, newTypecheckFS(nil)), files, opts)

	orig := runProgramDiagnostics
	t.Cleanup(func() { runProgramDiagnostics = orig })

	ctx, cancel := context.WithCancel(t.Context())
	runProgramDiagnostics = func(context.Context, *compiler.Program) []*ast.Diagnostic {
		cancel()
		return nil
	}
	if _, err := collectDiagnostics(ctx, program); !errors.Is(err, context.Canceled) {
		t.Fatalf("soft cancel err = %v", err)
	}

	runProgramDiagnostics = func(context.Context, *compiler.Program) []*ast.Diagnostic {
		panic("boom")
	}
	defer func() {
		if r := recover(); r != "boom" {
			t.Fatalf("recover = %#v", r)
		}
	}()
	_, _ = collectDiagnostics(t.Context(), program)
	t.Fatal("expected panic")
}

func TestCollectRootFiles_StatErrors(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, app)
	mustWrite(t, filepath.Join(app, "a.ts"), "export {};\n")

	orig := stat
	t.Cleanup(func() { stat = orig })

	stat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "demo" {
			return nil, errors.New("app stat boom")
		}
		return orig(name)
	}
	if _, err := CollectRootFiles(modules, "demo", ScopeService); err == nil || !strings.Contains(err.Error(), "app stat boom") {
		t.Fatalf("err = %v", err)
	}

	stat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "service" {
			return nil, errors.New("service stat boom")
		}
		return orig(name)
	}
	if _, err := CollectRootFiles(modules, "demo", ScopeService); err == nil || !strings.Contains(err.Error(), "service stat boom") {
		t.Fatalf("err = %v", err)
	}

	// App path exists but is a file → ErrNoRootFiles.
	modules2 := filepath.Join(dir, "modules2")
	mustMkdir(t, modules2)
	mustWrite(t, filepath.Join(modules2, "demo"), "not a dir")
	stat = orig
	if _, err := CollectRootFiles(modules2, "demo", ScopeService); !errors.Is(err, ErrNoRootFiles) {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildCompilerOptions_BaseURLAndErrors(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, filepath.Join(modules, "lib"))
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": "lib",
    "paths": {
      "@abs/*": ["/abs/*"],
      "@rel/*": ["./rel/*"]
    }
  }
}
`)
	opts, err := BuildCompilerOptions(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	rel, ok := opts.Paths.Get("@rel/*")
	if !ok || len(rel) != 1 || !strings.HasSuffix(rel[0], "/lib/rel/*") {
		t.Fatalf("rel = %v", rel)
	}
	abs, ok := opts.Paths.Get("@abs/*")
	if !ok || abs[0] != "/abs/*" {
		t.Fatalf("abs = %v", abs)
	}
	if opts.PathsBasePath != filepath.ToSlash(filepath.Join(modules, "lib")) {
		t.Fatalf("PathsBasePath = %q", opts.PathsBasePath)
	}

	mustWrite(t, filepath.Join(modules, "tsconfig.json"), "{ not json")
	if _, err := BuildCompilerOptions(modules, dir); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildCompilerOptions_AbsBaseURL(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	base := filepath.Join(dir, "custom-base")
	mustMkdir(t, modules)
	mustMkdir(t, base)
	mustWrite(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": "`+filepath.ToSlash(base)+`",
    "paths": { "@x/*": ["x/*"] }
  }
}
`)
	opts, err := BuildCompilerOptions(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	if opts.PathsBasePath != filepath.ToSlash(base) {
		t.Fatalf("PathsBasePath = %q want %q", opts.PathsBasePath, base)
	}
}

func TestResolveTypeRootsAndTypes(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "node_modules", "@types", "node")
	mustMkdir(t, local)
	roots := resolveTypeRoots(dir)
	if len(roots) != 1 {
		t.Fatalf("roots = %v", roots)
	}
	if got := resolveCompilerTypes(roots); len(got) != 1 || got[0] != "node" {
		t.Fatalf("types = %v", got)
	}
	if got := resolveCompilerTypes(nil); got != nil {
		t.Fatalf("empty = %v", got)
	}

	fileNode := t.TempDir()
	mustWrite(t, filepath.Join(fileNode, "node"), "not a dir")
	if got := resolveCompilerTypes([]string{fileNode}); got != nil {
		t.Fatalf("file named node must be ignored, got %v", got)
	}

	global := t.TempDir()
	mustMkdir(t, filepath.Join(global, "@types"))
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", global)
	roots = resolveTypeRoots(dir)
	if len(roots) != 2 {
		t.Fatalf("roots with global = %v", roots)
	}
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", filepath.Join(dir, "missing-global"))
	roots = resolveTypeRoots(t.TempDir())
	if len(roots) != 0 {
		t.Fatalf("expected empty roots, got %v", roots)
	}

	fileAsTypes := t.TempDir()
	mustMkdir(t, filepath.Join(fileAsTypes, "node_modules"))
	mustWrite(t, filepath.Join(fileAsTypes, "node_modules", "@types"), "not a dir")
	if got := resolveTypeRoots(fileAsTypes); len(got) != 0 {
		t.Fatalf("file @types must be ignored, got %v", got)
	}
}

func TestResolveModulePathsForTest(t *testing.T) {
	_, modules := fixtureRoots(t, "service_ok")
	paths, base, err := ResolveModulePathsForTest(modules)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := paths["@/*"]; !ok {
		t.Fatalf("expected @/* in %#v", paths)
	}
	if base == "" {
		t.Fatal("empty base")
	}
}

func TestHostLookupCaseInsensitive(t *testing.T) {
	overlays := normalizeOverlayMap(map[string]string{
		"/Foo/Bar.ts": "x",
	})
	if _, ok := lookupOverlay(overlays, "/foo/bar.ts", false); !ok {
		t.Fatal("expected case-insensitive hit")
	}
	if _, ok := lookupOverlay(overlays, "/other.ts", false); ok {
		t.Fatal("expected case-insensitive miss")
	}
	if _, ok := lookupOverlay(overlays, "/foo/bar.ts", true); ok {
		t.Fatal("expected case-sensitive miss")
	}
	if _, ok := lookupOverlay(nil, "/x", true); ok {
		t.Fatal("nil overlay")
	}
	if normalizePathKey("  ") != "" {
		t.Fatal("blank key")
	}
	if normalizeOverlayMap(nil) != nil {
		t.Fatal("nil map")
	}
	if normalizeOverlayMap(map[string]string{"  ": "x", "": "y"}) != nil {
		t.Fatal("blank-only overlays should normalize to nil")
	}
	if _, ok := lookupOverlay(map[string]string{"/a.ts": "x"}, "  ", true); ok {
		t.Fatal("blank lookup must miss")
	}
	h := newHost(".", newTypecheckFS(nil))
	if h == nil {
		t.Fatal("host")
	}
	h = newHost("", newTypecheckFS(nil))
	if h == nil {
		t.Fatal("host empty")
	}
}

func TestReportAndResultBranches(t *testing.T) {
	FormatStderr(nil, []Diagnostic{{Message: "x"}})
	var buf strings.Builder
	FormatStderr(&buf, []Diagnostic{
		{Category: "error", Code: 1, Message: "no file"},
		{File: "a.ts", Category: "error", Code: 2, Message: "no loc"},
	})
	out := buf.String()
	if !strings.Contains(out, "<unknown>") || !strings.Contains(out, "a.ts - error TS2") {
		t.Fatalf("out = %q", out)
	}

	res := toResult([]*ast.Diagnostic{nil})
	if len(res.Diagnostics) != 0 {
		t.Fatalf("%#v", res.Diagnostics)
	}
	if (Result{}).HasErrors() {
		t.Fatal("empty has errors")
	}
	if (Result{Diagnostics: []Diagnostic{{Category: "warning"}}}).Err() != nil {
		t.Fatal("warning should not Err")
	}

	for _, c := range []diagnostics.Category{
		diagnostics.CategoryError,
		diagnostics.CategoryWarning,
		diagnostics.CategorySuggestion,
		diagnostics.CategoryMessage,
		diagnostics.Category(99),
	} {
		_ = normalizeCategory(c)
	}

	line, col := positionToLineColumn(nil, 0)
	if line != 0 || col != 0 {
		t.Fatalf("%d:%d", line, col)
	}
	line, col = positionToLineColumn(&ast.SourceFile{}, -1)
	if line != 0 || col != 0 {
		t.Fatalf("%d:%d", line, col)
	}
	origMap := fileLineMap
	t.Cleanup(func() { fileLineMap = origMap })
	fileLineMap = func(*ast.SourceFile) []core.TextPos { return nil }
	line, col = positionToLineColumn(&ast.SourceFile{}, 3)
	if line != 1 || col != 4 {
		t.Fatalf("empty map: %d:%d", line, col)
	}

	// SourceFile with empty ECMALineMap path via synthetic diagnostic mapping without file.
	d := mapASTDiagnostic(ast.NewDiagnostic(nil, core.NewTextRange(0, 1), diagnostics.Unterminated_string_literal))
	if d.File != "" || d.Message == "" {
		t.Fatalf("%#v", d)
	}
	_ = locale.Default
}

func TestProgram_NilContext(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	opts, err := BuildCompilerOptions(modules, repo)
	if err != nil {
		t.Fatal(err)
	}
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	program := buildProgram(newHost(modules, newTypecheckFS(nil)), files, opts)
	diags, err := collectDiagnostics(nil, program)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diags {
		if d.Category().Name() == "error" {
			t.Fatalf("%s", d.Localize(locale.Default))
		}
	}
}

func TestValidateOptions_All(t *testing.T) {
	if err := validateOptions(Options{ModulesPath: "m", RepoRoot: "r"}); !errors.Is(err, ErrAppRequired) {
		t.Fatalf("%v", err)
	}
	if err := validateOptions(Options{ModulesPath: "m", App: "a"}); !errors.Is(err, ErrRepoRootRequired) {
		t.Fatalf("%v", err)
	}
}

func TestBuildCompilerOptions_MissingTsconfigDefaultAlias(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	opts, err := BuildCompilerOptions(modules, dir)
	if err != nil {
		t.Fatal(err)
	}
	targets, ok := opts.Paths.Get("@/*")
	if !ok || len(targets) == 0 {
		t.Fatal("expected default @/*")
	}
}

func TestCollectRootFiles_DedupAndWalkError(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "service"))
	mustWrite(t, filepath.Join(app, "a.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "b.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "c.ts"), "export {};\n")

	origAbs := absPath
	t.Cleanup(func() { absPath = origAbs })
	absPath = func(string) (string, error) { return "/dedup", nil }
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "/dedup" {
		t.Fatalf("dedup files = %v", files)
	}

	absPath = origAbs
	origWalk := walkDir
	t.Cleanup(func() { walkDir = origWalk })
	walkDir = func(string, fs.WalkDirFunc) error {
		return errors.New("walk boom")
	}
	if _, err := CollectRootFiles(modules, "demo", ScopeService); err == nil || !strings.Contains(err.Error(), "walk boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectRootFiles_WalkCallbackError(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "service"))
	mustWrite(t, filepath.Join(app, "service", "a.ts"), "export {};\n")

	origWalk := walkDir
	t.Cleanup(func() { walkDir = origWalk })
	walkDir = func(root string, fn fs.WalkDirFunc) error {
		return fn(root, nil, errors.New("entry boom"))
	}
	if _, err := CollectRootFiles(modules, "demo", ScopeService); err == nil || !strings.Contains(err.Error(), "entry boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestShouldSkipTSFileName_Dts(t *testing.T) {
	if shouldSkipTSFileName("types.d.ts") {
		t.Fatal("ambient d.ts must not be skipped")
	}
	if !shouldSkipTSFileName("ok.test.d.ts") || !shouldSkipTSFileName("ok.spec.d.ts") {
		t.Fatal("test declaration files must be skipped")
	}
}

func TestBuildCompilerOptions_ReadError(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	orig := readFile
	t.Cleanup(func() { readFile = orig })
	readFile = func(string) ([]byte, error) {
		return nil, errors.New("read boom")
	}
	if _, err := BuildCompilerOptions(modules, dir); err == nil || !strings.Contains(err.Error(), "read boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck_BuildCompilerOptionsError(t *testing.T) {
	repo, modules := fixtureRoots(t, "service_ok")
	orig := readFile
	t.Cleanup(func() { readFile = orig })
	readFile = func(string) ([]byte, error) {
		return nil, errors.New("opts boom")
	}
	if _, err := Check(t.Context(), Options{ModulesPath: modules, RepoRoot: repo, App: "demo"}); err == nil || !strings.Contains(err.Error(), "opts boom") {
		t.Fatalf("err = %v", err)
	}
}
