// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadWebInputDigestInputs(t *testing.T) {
	t.Parallel()
	if _, err := LoadWebInputDigestInputs(nil, "", false, true, true, false); err == nil {
		t.Fatal("expected nil scope error")
	}
	nilSession := &testScope{cfg: &config.Config{}}
	if _, err := LoadWebInputDigestInputs(nilSession, "", false, true, true, false); err == nil {
		t.Fatal("expected nil session error")
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "digest.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	modulesPath := t.TempDir()
	modPath := filepath.Join(modulesPath, "webmod")
	if err := os.MkdirAll(filepath.Join(modPath, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	absEntry := filepath.Join(modPath, "web", "abs.ts")
	if err := os.WriteFile(absEntry, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write abs: %v", err)
	}
	for _, mod := range []meta.Module{
		{Name: "webmod", Version: "1.0.0", Status: meta.Installed, Path: modPath, WebEntryPoint: "web/index.ts"},
		{Name: "absmod", Version: "2.0.0", Status: meta.Installed, Path: modPath, WebEntryPoint: absEntry},
		{Name: "blank", Version: "1.0.0", Status: meta.Installed, Path: modPath, WebEntryPoint: "   "},
		{Name: "gone", Version: "1.0.0", Status: meta.Uninstalled, Path: modPath, WebEntryPoint: "web/x.ts"},
	} {
		m := mod
		if err := db.Create(&m).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	runtimeScope := &testScope{cfg: &config.Config{ModulesPath: modulesPath}, db: db}
	in, err := LoadWebInputDigestInputs(runtimeScope, modulesPath, true, false, true, true)
	if err != nil {
		t.Fatalf("LoadWebInputDigestInputs: %v", err)
	}
	if !in.SourceMap || in.Minify || !in.TreeShaking || !in.ForceRebuild {
		t.Fatalf("flags = %#v", in)
	}
	if len(in.WebEntryPoints) != 2 {
		t.Fatalf("entries = %#v, want 2", in.WebEntryPoints)
	}
	rel := in.WebEntryPoints[0]
	if rel.ModuleName != "absmod" && rel.ModuleName != "webmod" {
		t.Fatalf("unexpected first entry %#v", rel)
	}
	joined := false
	for _, ref := range in.WebEntryPoints {
		if ref.ModuleName == "webmod" {
			joined = filepath.IsAbs(ref.EntryPath) && filepath.Base(ref.EntryPath) == "index.ts"
		}
	}
	if !joined {
		t.Fatalf("relative entry was not joined: %#v", in.WebEntryPoints)
	}
	// Prefer mod.Path over modulesPath/mod.Name when they diverge.
	otherRoot := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(filepath.Join(otherRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir elsewhere: %v", err)
	}
	if err := os.WriteFile(filepath.Join(otherRoot, "web", "index.ts"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write elsewhere entry: %v", err)
	}
	if err := db.Create(&meta.Module{
		Name: "pathmod", Version: "1.0.0", Status: meta.Installed,
		Path: otherRoot, WebEntryPoint: "web/index.ts",
	}).Error; err != nil {
		t.Fatalf("create pathmod: %v", err)
	}
	in2, err := LoadWebInputDigestInputs(runtimeScope, modulesPath, false, true, true, false)
	if err != nil {
		t.Fatalf("LoadWebInputDigestInputs pathmod: %v", err)
	}
	foundPath := false
	wantPath, _ := filepath.Abs(filepath.Join(otherRoot, "web", "index.ts"))
	for _, ref := range in2.WebEntryPoints {
		if ref.ModuleName == "pathmod" {
			foundPath = true
			if ref.EntryPath != wantPath {
				t.Fatalf("pathmod entry = %q, want %q", ref.EntryPath, wantPath)
			}
		}
	}
	if !foundPath {
		t.Fatal("pathmod entry missing")
	}

	// Empty Path falls back to modulesPath/mod.Name.
	if err := db.Create(&meta.Module{
		Name: "nopath", Version: "1.0.0", Status: meta.Installed,
		Path: "", WebEntryPoint: "web/index.ts",
	}).Error; err != nil {
		t.Fatalf("create nopath: %v", err)
	}
	// Ensure modulesPath/nopath/web/index.ts exists for a resolvable join.
	noPathEntry := filepath.Join(modulesPath, "nopath", "web", "index.ts")
	if err := os.MkdirAll(filepath.Dir(noPathEntry), 0o755); err != nil {
		t.Fatalf("mkdir nopath: %v", err)
	}
	if err := os.WriteFile(noPathEntry, []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write nopath: %v", err)
	}
	in3, err := LoadWebInputDigestInputs(runtimeScope, modulesPath, false, true, true, false)
	if err != nil {
		t.Fatalf("LoadWebInputDigestInputs nopath: %v", err)
	}
	foundNoPath := false
	wantNoPath, _ := filepath.Abs(noPathEntry)
	for _, ref := range in3.WebEntryPoints {
		if ref.ModuleName == "nopath" {
			foundNoPath = true
			if ref.EntryPath != wantNoPath {
				t.Fatalf("nopath entry = %q, want %q", ref.EntryPath, wantNoPath)
			}
		}
	}
	if !foundNoPath {
		t.Fatal("nopath entry missing")
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWebInputDigestInputs(runtimeScope, modulesPath, false, true, true, false); err == nil {
		t.Fatal("expected closed-db Find error")
	}
}

func TestComputeWebInputDigestStableAndSensitive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modPath := filepath.Join(root, "web")
	entry := filepath.Join(modPath, "web", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(entry, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	in := WebInputDigestInputs{
		ModulesPath: root,
		SourceMap:   false,
		Minify:      true,
		TreeShaking: true,
		WebEntryPoints: []webEntryRef{{
			ModuleName: "web",
			Version:    "1.0.0",
			EntryPath:  entry,
			ModulePath: modPath,
		}},
	}
	a, err := ComputeWebInputDigest(in)
	if err != nil || a == "" {
		t.Fatalf("ComputeWebInputDigest() = %q, %v", a, err)
	}
	b, err := ComputeWebInputDigest(in)
	if err != nil || b != a {
		t.Fatalf("digest not stable: %q vs %q (%v)", a, b, err)
	}
	in.SourceMap = true
	c, err := ComputeWebInputDigest(in)
	if err != nil || c == a {
		t.Fatalf("sourcemap change should alter digest: %q vs %q (%v)", a, c, err)
	}
	in.SourceMap = false
	if err := os.WriteFile(entry, []byte("export default { changed: true }\n"), 0o644); err != nil {
		t.Fatalf("rewrite entry: %v", err)
	}
	d, err := ComputeWebInputDigest(in)
	if err != nil || d == a {
		t.Fatalf("source edit should alter digest: %q vs %q (%v)", a, d, err)
	}
	apiWeb := filepath.Join(root, "api", "web", "crm", "index.ts")
	if err := os.MkdirAll(filepath.Dir(apiWeb), 0o755); err != nil {
		t.Fatalf("mkdir api web: %v", err)
	}
	if err := os.WriteFile(apiWeb, []byte("export const client = 1\n"), 0o644); err != nil {
		t.Fatalf("write api web: %v", err)
	}
	e, err := ComputeWebInputDigest(in)
	if err != nil || e == d {
		t.Fatalf("api/web change should alter digest: %q vs %q (%v)", d, e, err)
	}
	in.ForceRebuild = true
	forced, err := ComputeWebInputDigest(in)
	if err != nil || forced == "" || forced != e {
		t.Fatalf("ForceRebuild must still return a stampable digest, got %q (%v)", forced, err)
	}

	choyStyles := filepath.Join(root, "choy_ui", "web", "styles")
	if err := os.MkdirAll(choyStyles, 0o755); err != nil {
		t.Fatalf("mkdir choy styles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(choyStyles, "theme.css"), []byte(`@theme { --color-primary: red; }`), 0o644); err != nil {
		t.Fatalf("write theme: %v", err)
	}
	in.ForceRebuild = false
	withTailwind, err := ComputeWebInputDigest(in)
	if err != nil || withTailwind == e {
		t.Fatalf("choy_ui dialect should alter digest: %q vs %q (%v)", e, withTailwind, err)
	}
	// Place the generated artifact under the choy_ui kit web tree (also hashed as a
	// web entry) so the digest walker exercises the kit-scoped exclusion branch.
	choyMod := filepath.Join(root, "choy_ui")
	choyEntry := filepath.Join(choyMod, "web", "index.ts")
	if err := os.MkdirAll(filepath.Dir(choyEntry), 0o755); err != nil {
		t.Fatalf("mkdir choy entry: %v", err)
	}
	if err := os.WriteFile(choyEntry, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write choy entry: %v", err)
	}
	in.WebEntryPoints = append(in.WebEntryPoints, webEntryRef{
		ModuleName: "choy_ui",
		Version:    "0.0.0",
		EntryPath:  choyEntry,
		ModulePath: choyMod,
	})
	withChoyEntry, err := ComputeWebInputDigest(in)
	if err != nil {
		t.Fatalf("digest with choy_ui entry: %v", err)
	}
	hashedGen := filepath.Join(choyMod, "web", "styles", "choy-tailwind.generated.css")
	if err := os.WriteFile(hashedGen, []byte("/* noise */\n.flex{}\n"), 0o644); err != nil {
		t.Fatalf("write generated under choy_ui: %v", err)
	}
	afterGenerated, err := ComputeWebInputDigest(in)
	if err != nil || afterGenerated != withChoyEntry {
		t.Fatalf("choy_ui choy-tailwind.generated.css must not alter digest: %q vs %q (%v)", withChoyEntry, afterGenerated, err)
	}
	// Same basename outside the kit is a real input and must invalidate.
	if err := os.WriteFile(filepath.Join(modPath, "web", "choy-tailwind.generated.css"), []byte("/* other module */\n"), 0o644); err != nil {
		t.Fatalf("write generated under other module: %v", err)
	}
	afterOtherName, err := ComputeWebInputDigest(in)
	if err != nil || afterOtherName == afterGenerated {
		t.Fatalf("same-named generated.css outside choy_ui should alter digest: %q vs %q (%v)", afterGenerated, afterOtherName, err)
	}
	if err := os.WriteFile(filepath.Join(modPath, "web", "other.generated.css"), []byte(".other{}\n"), 0o644); err != nil {
		t.Fatalf("write other generated: %v", err)
	}
	afterOther, err := ComputeWebInputDigest(in)
	if err != nil || afterOther == afterOtherName {
		t.Fatalf("non-choy *.generated.css under module should alter digest: %q vs %q (%v)", afterOtherName, afterOther, err)
	}

	// A tailwind-go engine bump must invalidate the digest even when dialect and
	// candidates are unchanged; readBuildInfo is stubbed to simulate the bump.
	prevReadBuildInfo := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Deps: []*debug.Module{{Path: choyTailwindGoModulePath, Version: "v9.9.9"}}}, true
	}
	bumpedEngine, err := ComputeWebInputDigest(in)
	readBuildInfo = prevReadBuildInfo
	if err != nil || bumpedEngine == afterOther {
		t.Fatalf("engine version change must alter digest: %q vs %q (%v)", afterOther, bumpedEngine, err)
	}

	// TailwindInputDigest error should fail the digest.
	badTheme := filepath.Join(root, "choy_ui", "web", "styles", "theme.css")
	if err := os.Chmod(badTheme, 0o000); err != nil {
		t.Fatalf("chmod theme: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(badTheme, 0o644) })
	if _, err := os.ReadFile(badTheme); err == nil {
		t.Skip("theme.css still readable on this runner (e.g. root)")
	} else if _, err := ComputeWebInputDigest(in); err == nil {
		_ = os.Chmod(badTheme, 0o644)
		t.Fatal("expected ComputeWebInputDigest to surface TailwindInputDigest error")
	}
	_ = os.Chmod(badTheme, 0o644)

	// Empty roots / "." must not walk the process cwd.
	empty, err := ComputeWebInputDigest(WebInputDigestInputs{
		WebEntryPoints: []webEntryRef{{ModuleName: "x", EntryPath: "", ModulePath: "."}},
	})
	if err != nil || empty == "" {
		t.Fatalf("empty/dot roots digest = %q (%v)", empty, err)
	}

	// Entry under dist/ must still affect the digest; siblings under dist/ are included.
	distRoot := t.TempDir()
	distEntry := filepath.Join(distRoot, "dist", "web", "main.ts")
	if err := os.MkdirAll(filepath.Dir(distEntry), 0o755); err != nil {
		t.Fatalf("mkdir dist entry: %v", err)
	}
	if err := os.WriteFile(distEntry, []byte("export default 1\n"), 0o644); err != nil {
		t.Fatalf("write dist entry: %v", err)
	}
	sibling := filepath.Join(distRoot, "dist", "web", "sibling.ts")
	if err := os.WriteFile(sibling, []byte("export const s = 1\n"), 0o644); err != nil {
		t.Fatalf("write sibling: %v", err)
	}
	distIn := WebInputDigestInputs{
		ModulesPath: distRoot,
		Minify:      true,
		TreeShaking: true,
		WebEntryPoints: []webEntryRef{{
			ModuleName: "web",
			Version:    "1.0.0",
			EntryPath:  distEntry,
			ModulePath: distRoot,
		}},
	}
	before, err := ComputeWebInputDigest(distIn)
	if err != nil || before == "" {
		t.Fatalf("dist entry digest = %q (%v)", before, err)
	}
	if err := os.WriteFile(distEntry, []byte("export default 2\n"), 0o644); err != nil {
		t.Fatalf("rewrite dist entry: %v", err)
	}
	after, err := ComputeWebInputDigest(distIn)
	if err != nil || after == before {
		t.Fatalf("dist/ entry edit should alter digest: %q vs %q (%v)", before, after, err)
	}
	if err := os.WriteFile(distEntry, []byte("export default 2\n"), 0o644); err != nil {
		t.Fatalf("rewrite dist entry again: %v", err)
	}
	mid, err := ComputeWebInputDigest(distIn)
	if err != nil {
		t.Fatalf("mid digest: %v", err)
	}
	if err := os.WriteFile(sibling, []byte("export const s = 2\n"), 0o644); err != nil {
		t.Fatalf("rewrite sibling: %v", err)
	}
	sib, err := ComputeWebInputDigest(distIn)
	if err != nil || sib == mid {
		t.Fatalf("dist/ sibling edit should alter digest: %q vs %q (%v)", mid, sib, err)
	}

	// File-as-root and ignored extensions / missing paths.
	fileRoot := filepath.Join(t.TempDir(), "only.ts")
	if err := os.WriteFile(fileRoot, []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write file root: %v", err)
	}
	if _, err := ComputeWebInputDigest(WebInputDigestInputs{
		WebEntryPoints: []webEntryRef{{ModuleName: "f", EntryPath: fileRoot, ModulePath: fileRoot}},
	}); err != nil {
		t.Fatalf("file root digest: %v", err)
	}
	modWithJunk := t.TempDir()
	if err := os.WriteFile(filepath.Join(modWithJunk, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(modWithJunk, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modWithJunk, "node_modules", "pkg", "x.ts"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write nm: %v", err)
	}
	if _, err := ComputeWebInputDigest(WebInputDigestInputs{
		WebEntryPoints: []webEntryRef{{ModuleName: "j", EntryPath: filepath.Join(modWithJunk, "missing.ts"), ModulePath: modWithJunk}},
	}); err != nil {
		t.Fatalf("missing entry digest: %v", err)
	}
	if err := hashWebSourceTree(nilWriter{}, filepath.Join(t.TempDir(), "missing-root")); err != nil {
		t.Fatalf("missing root: %v", err)
	}
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestShouldSkipAndStampHelpers(t *testing.T) {
	t.Parallel()
	dist := t.TempDir()
	if skip, err := ShouldSkipGlobalWebBuild(dist, ""); err != nil || skip {
		t.Fatalf("empty digest: skip=%v err=%v", skip, err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("missing index should not skip: skip=%v err=%v", skip, err)
	}
	index := filepath.Join(dist, "index.html")
	if err := os.Mkdir(index, 0o755); err != nil {
		t.Fatalf("mkdir index-as-dir: %v", err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("dir index should not skip: skip=%v err=%v", skip, err)
	}
	if err := os.Remove(index); err != nil {
		t.Fatalf("remove dir index: %v", err)
	}
	if err := os.WriteFile(index, nil, 0o644); err != nil {
		t.Fatalf("write empty index: %v", err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("empty index should not skip: skip=%v err=%v", skip, err)
	}
	if err := os.WriteFile(index, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("missing stamp should not skip: skip=%v err=%v", skip, err)
	}
	if err := WriteStoredWebInputDigest(dist, "abc"); err != nil {
		t.Fatalf("WriteStoredWebInputDigest: %v", err)
	}
	skip, err := ShouldSkipGlobalWebBuild(dist, "abc")
	if err != nil || !skip {
		t.Fatalf("matching digest should skip: skip=%v err=%v", skip, err)
	}
	skip, err = ShouldSkipGlobalWebBuild(dist, "other")
	if err != nil || skip {
		t.Fatalf("mismatch should not skip: skip=%v err=%v", skip, err)
	}
	if err := WriteStoredWebInputDigest("", "x"); err != nil {
		t.Fatalf("empty dist write: %v", err)
	}
	if err := WriteStoredWebInputDigest(dist, ""); err != nil {
		t.Fatalf("empty digest write: %v", err)
	}
	got, err := ReadStoredWebInputDigest(filepath.Join(t.TempDir(), "missing"))
	if err != nil || got != "" {
		t.Fatalf("missing stamp read = %q (%v)", got, err)
	}
	// MkdirAll failure when parent path is a file.
	notDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("write not-a-dir: %v", err)
	}
	if err := WriteStoredWebInputDigest(filepath.Join(notDir, "web"), "abc"); err == nil {
		t.Fatal("expected MkdirAll error")
	}
	// index.html Stat non-IsNotExist (symlink loop).
	loopDist := t.TempDir()
	loopIndex := filepath.Join(loopDist, "index.html")
	loopA := filepath.Join(loopDist, "a")
	loopB := filepath.Join(loopDist, "b")
	if err := os.Symlink(loopB, loopA); err != nil {
		t.Fatalf("symlink a: %v", err)
	}
	if err := os.Symlink(loopA, loopB); err != nil {
		t.Fatalf("symlink b: %v", err)
	}
	if err := os.Symlink(loopA, loopIndex); err != nil {
		t.Fatalf("symlink index: %v", err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(loopDist, "abc"); err == nil || skip {
		t.Fatalf("symlink-loop index: skip=%v err=%v", skip, err)
	}
	// Stamp path that is a directory surfaces a non-IsNotExist read error.
	stampDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stampDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write index for stamp-dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(stampDir, webInputDigestFileName), 0o755); err != nil {
		t.Fatalf("mkdir stamp: %v", err)
	}
	if _, err := ReadStoredWebInputDigest(stampDir); err == nil {
		t.Fatal("expected read error for directory stamp")
	}
	if skip, err := ShouldSkipGlobalWebBuild(stampDir, "abc"); err == nil || skip {
		t.Fatalf("directory stamp skip: skip=%v err=%v", skip, err)
	}

	// Unreadable file hash errors (skip when mode 0 is still readable, e.g. UID 0).
	t.Run("unreadable entry", func(t *testing.T) {
		blocked := filepath.Join(t.TempDir(), "blocked.ts")
		if err := os.WriteFile(blocked, []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write blocked: %v", err)
		}
		if err := os.Chmod(blocked, 0); err != nil {
			t.Fatalf("chmod blocked: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(blocked, 0o644) })
		if _, err := os.ReadFile(blocked); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if _, err := ComputeWebInputDigest(WebInputDigestInputs{
			WebEntryPoints: []webEntryRef{{ModuleName: "b", EntryPath: blocked, ModulePath: filepath.Dir(blocked)}},
		}); err == nil {
			t.Fatal("expected hashFile permission error")
		}
	})

	// demo/ sibling tree hashing + Separator root guard.
	demoRoot := t.TempDir()
	demoEntry := filepath.Join(demoRoot, "demo", "web", "main.ts")
	if err := os.MkdirAll(filepath.Dir(demoEntry), 0o755); err != nil {
		t.Fatalf("mkdir demo: %v", err)
	}
	if err := os.WriteFile(demoEntry, []byte("export default 1\n"), 0o644); err != nil {
		t.Fatalf("write demo entry: %v", err)
	}
	if _, err := ComputeWebInputDigest(WebInputDigestInputs{
		WebEntryPoints: []webEntryRef{{
			ModuleName: "demo", Version: "1", EntryPath: demoEntry, ModulePath: "",
		}},
	}); err != nil {
		t.Fatalf("demo entry digest: %v", err)
	}
	if err := hashWebSourceTree(nilWriter{}, string(filepath.Separator)); err != nil {
		t.Fatalf("separator root: %v", err)
	}
	if pathHasSkippedWebComponent("a/demo/b") != true || pathHasSkippedWebComponent("a/src/b") {
		t.Fatal("pathHasSkippedWebComponent")
	}

	// Sort stability across module names / entry paths.
	sortRoot := t.TempDir()
	e1 := filepath.Join(sortRoot, "a.ts")
	e2 := filepath.Join(sortRoot, "b.ts")
	e3 := filepath.Join(sortRoot, "c.ts")
	for _, p := range []string{e1, e2, e3} {
		if err := os.WriteFile(p, []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	sortIn := WebInputDigestInputs{WebEntryPoints: []webEntryRef{
		{ModuleName: "z", EntryPath: e3, ModulePath: sortRoot},
		{ModuleName: "a", EntryPath: e2, ModulePath: sortRoot},
		{ModuleName: "a", EntryPath: e1, ModulePath: sortRoot},
	}}
	if _, err := ComputeWebInputDigest(sortIn); err != nil {
		t.Fatalf("sort digest: %v", err)
	}

	// api/web and module-root / dist-sibling hash errors surface.
	t.Run("unreadable api web", func(t *testing.T) {
		apiRoot := t.TempDir()
		apiBlocked := filepath.Join(apiRoot, "api", "web", "x.ts")
		if err := os.MkdirAll(filepath.Dir(apiBlocked), 0o755); err != nil {
			t.Fatalf("mkdir api: %v", err)
		}
		if err := os.WriteFile(apiBlocked, []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write api: %v", err)
		}
		if err := os.Chmod(apiBlocked, 0); err != nil {
			t.Fatalf("chmod api: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(apiBlocked, 0o644) })
		if _, err := os.ReadFile(apiBlocked); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if _, err := ComputeWebInputDigest(WebInputDigestInputs{ModulesPath: apiRoot}); err == nil {
			t.Fatal("expected api/web hash error")
		}
	})

	t.Run("unreadable module root file", func(t *testing.T) {
		modBlockedRoot := t.TempDir()
		modFile := filepath.Join(modBlockedRoot, "app.ts")
		if err := os.WriteFile(modFile, []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write mod: %v", err)
		}
		if err := os.Chmod(modFile, 0); err != nil {
			t.Fatalf("chmod mod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(modFile, 0o644) })
		if _, err := os.ReadFile(modFile); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if _, err := ComputeWebInputDigest(WebInputDigestInputs{
			WebEntryPoints: []webEntryRef{{ModuleName: "m", EntryPath: filepath.Join(modBlockedRoot, "missing.ts"), ModulePath: modBlockedRoot}},
		}); err == nil {
			t.Fatal("expected module-root hash error")
		}
	})

	t.Run("unreadable dist sibling", func(t *testing.T) {
		distBlocked := t.TempDir()
		distEntry := filepath.Join(distBlocked, "dist", "web", "main.ts")
		sib := filepath.Join(distBlocked, "dist", "web", "sib.ts")
		if err := os.MkdirAll(filepath.Dir(distEntry), 0o755); err != nil {
			t.Fatalf("mkdir dist: %v", err)
		}
		if err := os.WriteFile(distEntry, []byte("export default 1\n"), 0o644); err != nil {
			t.Fatalf("write dist entry: %v", err)
		}
		if err := os.WriteFile(sib, []byte("export const s = 1\n"), 0o644); err != nil {
			t.Fatalf("write sib: %v", err)
		}
		if err := os.Chmod(sib, 0); err != nil {
			t.Fatalf("chmod sib: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(sib, 0o644) })
		if _, err := os.ReadFile(sib); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if _, err := ComputeWebInputDigest(WebInputDigestInputs{
			WebEntryPoints: []webEntryRef{{ModuleName: "d", EntryPath: distEntry, ModulePath: distBlocked}},
		}); err == nil {
			t.Fatal("expected dist sibling hash error")
		}
	})

	t.Run("unreadable nested walk dir", func(t *testing.T) {
		walkRoot := t.TempDir()
		nested := filepath.Join(walkRoot, "nested")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		if err := os.WriteFile(filepath.Join(nested, "x.ts"), []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write nested: %v", err)
		}
		if err := os.Chmod(nested, 0); err != nil {
			t.Fatalf("chmod nested: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(nested, 0o755) })
		if _, err := os.ReadDir(nested); err == nil {
			t.Skip("filesystem permits readdir despite mode 0")
		}
		if err := hashWebSourceTree(nilWriter{}, walkRoot); err == nil {
			t.Fatal("expected walkErr")
		}
	})

	// WalkDir / Stat non-IsNotExist failures (symlink loops).
	loopRoot := t.TempDir()
	a := filepath.Join(loopRoot, "a")
	b := filepath.Join(loopRoot, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatalf("symlink a: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatalf("symlink b: %v", err)
	}
	if err := hashWebSourceTree(nilWriter{}, a); err == nil {
		t.Fatal("expected symlink-loop root error")
	}
	if err := hashFile(nilWriter{}, a); err == nil {
		t.Fatal("expected symlink-loop file error")
	}

	// Non-regular paths are skipped (do not block on FIFOs/devices).
	dirAsFile := t.TempDir()
	if err := hashFile(nilWriter{}, dirAsFile); err != nil {
		t.Fatalf("directory hashFile: %v", err)
	}
}

func TestIsChoyTailwindGeneratedKitPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/abs/modules/web/web/styles/" + choyTailwindGeneratedCSSName, true},
		{"web/web/styles/" + choyTailwindGeneratedCSSName, true},
		{"web/web/" + choyTailwindGeneratedCSSName, false},
		{"/abs/modules/choy_ui/web/styles/" + choyTailwindGeneratedCSSName, true},
		{"choy_ui/web/styles/" + choyTailwindGeneratedCSSName, true},
		{"choy_ui/web/" + choyTailwindGeneratedCSSName, false},
		{"other/web/styles/" + choyTailwindGeneratedCSSName, false},
		{"choy_ui_extra/web/styles/" + choyTailwindGeneratedCSSName, false},
		{"choy_ui/web/styles/other.css", false},
		{"not-the-file.css", false},
	}
	for _, tc := range cases {
		if got := isChoyTailwindGeneratedKitPath(tc.path); got != tc.want {
			t.Fatalf("isChoyTailwindGeneratedKitPath(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}

func TestChoyTailwindGoModuleVersion(t *testing.T) {
	t.Cleanup(func() { readBuildInfo = debug.ReadBuildInfo })

	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	if got := choyTailwindGoModuleVersion(); got != "" {
		t.Fatalf("!ok => empty, got %q", got)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, true }
	if got := choyTailwindGoModuleVersion(); got != "" {
		t.Fatalf("nil BuildInfo => empty, got %q", got)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Deps: []*debug.Module{{Path: "example.com/other", Version: "v1.0.0"}}}, true
	}
	if got := choyTailwindGoModuleVersion(); got != "" {
		t.Fatalf("missing dep => empty, got %q", got)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Deps: []*debug.Module{
			{Path: "example.com/other", Version: "v1.0.0"},
			{Path: choyTailwindGoModulePath, Version: "v0.4.0"},
		}}, true
	}
	if got := choyTailwindGoModuleVersion(); got != "v0.4.0" {
		t.Fatalf("matched dep => v0.4.0, got %q", got)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Deps: []*debug.Module{{
			Path:    choyTailwindGoModulePath,
			Version: "v0.4.0",
			Replace: &debug.Module{Path: "github.com/dhamidi/tailwind-go", Version: "v0.4.1"},
		}}}, true
	}
	if got := choyTailwindGoModuleVersion(); got != "github.com/dhamidi/tailwind-go@v0.4.1" {
		t.Fatalf("replace path@version => %q", got)
	}

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Deps: []*debug.Module{nil, {
			Path:    choyTailwindGoModulePath,
			Version: "v0.4.0",
			Replace: &debug.Module{Path: "../tailwind-go"},
		}}}, true
	}
	if got := choyTailwindGoModuleVersion(); got != "../tailwind-go" {
		t.Fatalf("replace path => ../tailwind-go, got %q", got)
	}
}

func TestIndexCSSBareAtRuleSemi(t *testing.T) {
	if got := indexCSSBareAtRuleSemi(`@import url("a;b.css");`); got != len(`@import url("a;b.css")`) {
		t.Fatalf("quoted semi: %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@import url("a\"b;c.css");`); got != len(`@import url("a\"b;c.css")`) {
		t.Fatalf("escaped quote in url: %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@import url('a\'b;c.css');`); got != len(`@import url('a\'b;c.css')`) {
		t.Fatalf("escaped single quote: %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@layer utilities { .a{} }`); got != -1 {
		t.Fatalf("block at-rule => -1, got %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@import url(a)`); got != -1 {
		t.Fatalf("no terminator => -1, got %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@supports (display: flex) { .a{} }`); got != -1 {
		t.Fatalf("paren then brace => -1, got %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@import /* ; */ url(x.css);`); got != len(`@import /* ; */ url(x.css)`) {
		t.Fatalf("semi inside comment: %d", got)
	}
	if got := indexCSSBareAtRuleSemi(`@import /* unterminated`); got != -1 {
		t.Fatalf("unterminated comment => -1, got %d", got)
	}
}
