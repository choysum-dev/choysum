// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tw "github.com/dhamidi/tailwind-go"
)

func TestScanTailwindCandidatesEdgePaths(t *testing.T) {
	root := t.TempDir()

	// Empty / whitespace roots are skipped.
	got, err := ScanTailwindCandidates([]string{"", "  ", root})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty scan of empty dir, got %v", got)
	}

	// Missing path is ignored.
	got, err = ScanTailwindCandidates([]string{filepath.Join(root, "missing")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty for missing root, got %v", got)
	}

	// Stat error other than NotExist (path is a file we cannot... use an
	// unreadable directory on Unix).
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	if _, err := ScanTailwindCandidates([]string{blocked}); err == nil {
		// Some environments (e.g. running as root) can still walk mode 000.
		t.Log("stat/walk on mode 000 did not error; continuing")
	} else {
		_ = os.Chmod(blocked, 0o755)
	}

	vue := filepath.Join(root, "Ok.vue")
	if err := os.WriteFile(vue, []byte(`<div class="flex p-2"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	skipGen := filepath.Join(root, "choy-tailwind.generated.css")
	if err := os.WriteFile(skipGen, []byte(`.hidden-gen{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	skipTheme := filepath.Join(root, "theme.css")
	if err := os.WriteFile(skipTheme, []byte(`@theme { --color-x: red; } .from-theme{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	skipExt := filepath.Join(root, "notes.md")
	if err := os.WriteFile(skipExt, []byte(`class="md-only"`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"node_modules", "dist", ".git"} {
		sub := filepath.Join(root, name)
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sub, "x.vue"), []byte(`<div class="from-skip-dir"></div>`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Single-file root (non-directory).
	got, err = ScanTailwindCandidates([]string{vue})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "flex") || !strings.Contains(joined, "p-2") {
		t.Fatalf("single-file scan missing classes: %v", got)
	}

	// Single-file root that should be skipped.
	got, err = ScanTailwindCandidates([]string{skipGen})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("generated css file should not contribute candidates, got %v", got)
	}

	got, err = ScanTailwindCandidates([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c == "from-skip-dir" || c == "from-theme" || c == "hidden-gen" || c == "md-only" {
			t.Fatalf("unexpected candidate %q in %v", c, got)
		}
	}
}

func TestIsPlausibleTailwindCandidate(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"flex", true},
		{"", false},
		{strings.Repeat("a", maxTailwindCandidateLen+1), false},
		{"has space", false},
		{"has\nnewline", false},
		{"has\ttab", false},
		{"has{brace", false},
		{"has;semi", false},
		{"ok-arbitrary-[1px]", true},
		{"bad\x00null", false},
	}
	for _, tc := range cases {
		if got := isPlausibleTailwindCandidate(tc.in); got != tc.want {
			t.Fatalf("isPlausibleTailwindCandidate(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestGenerateChoyTailwindForModuleErrorsAndStableWrite(t *testing.T) {
	if _, err := GenerateChoyTailwindForModule(""); err == nil {
		t.Fatal("expected error for empty module root")
	}
	if _, err := GenerateChoyTailwindForModule("   "); err == nil {
		t.Fatal("expected error for whitespace module root")
	}
	if _, err := GenerateChoyTailwindForModule(t.TempDir()); err == nil {
		t.Fatal("expected error when theme.css missing")
	}

	root := t.TempDir()
	web := filepath.Join(root, "web")
	styles := filepath.Join(web, "styles")
	pages := filepath.Join(web, "pages")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pages, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(styles, "theme.css"), []byte(`@theme { --color-primary: red; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pages, "A.vue"), []byte(`<div class="flex"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}

	res1, err := GenerateChoyTailwindForModule(root)
	if err != nil {
		t.Fatal(err)
	}
	st1, err := os.Stat(res1.OutputPath)
	if err != nil {
		t.Fatal(err)
	}

	res2, err := GenerateChoyTailwindForModule(root)
	if err != nil {
		t.Fatal(err)
	}
	st2, err := os.Stat(res2.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !st1.ModTime().Equal(st2.ModTime()) {
		t.Fatal("unchanged generate should skip rewrite (mtime stable)")
	}
	if res1.ContentHash != res2.ContentHash {
		t.Fatal("content hash should be stable")
	}
}

func TestWriteFileAtomicIfChangedErrors(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "out.css")
	if err := writeFileAtomicIfChanged(path, "one"); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomicIfChanged(path, "one"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "one" {
		t.Fatalf("read back = %q err=%v", data, err)
	}

	// Parent path component is a file → MkdirAll fails.
	fileParent := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(fileParent, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomicIfChanged(filepath.Join(fileParent, "child.css"), "css"); err == nil {
		t.Fatal("expected MkdirAll failure when parent is a file")
	}

	// Force OS-stage failures via injectable helpers.
	t.Cleanup(func() {
		atomicWriteWriteString = func(tmp *os.File, content string) error {
			_, err := tmp.WriteString(content)
			return err
		}
		atomicWriteChmod = func(tmp *os.File, mode os.FileMode) error { return tmp.Chmod(mode) }
		atomicWriteSync = func(tmp *os.File) error { return tmp.Sync() }
		atomicWriteClose = func(tmp *os.File) error { return tmp.Close() }
		atomicWriteRename = os.Rename
	})
	atomicWriteWriteString = func(*os.File, string) error { return os.ErrInvalid }
	if err := writeFileAtomicIfChanged(filepath.Join(root, "fail-write.css"), "x"); err == nil {
		t.Fatal("expected WriteString failure")
	}
	atomicWriteWriteString = func(tmp *os.File, content string) error {
		_, err := tmp.WriteString(content)
		return err
	}
	atomicWriteChmod = func(*os.File, os.FileMode) error { return os.ErrInvalid }
	if err := writeFileAtomicIfChanged(filepath.Join(root, "fail-chmod.css"), "x"); err == nil {
		t.Fatal("expected Chmod failure")
	}
	atomicWriteChmod = func(tmp *os.File, mode os.FileMode) error { return tmp.Chmod(mode) }
	atomicWriteSync = func(*os.File) error { return os.ErrInvalid }
	if err := writeFileAtomicIfChanged(filepath.Join(root, "fail-sync.css"), "x"); err == nil {
		t.Fatal("expected Sync failure")
	}
	atomicWriteSync = func(tmp *os.File) error { return tmp.Sync() }
	atomicWriteClose = func(*os.File) error { return os.ErrInvalid }
	if err := writeFileAtomicIfChanged(filepath.Join(root, "fail-close.css"), "x"); err == nil {
		t.Fatal("expected Close failure")
	}
	atomicWriteClose = func(tmp *os.File) error { return tmp.Close() }
	atomicWriteRename = func(string, string) error { return os.ErrInvalid }
	if err := writeFileAtomicIfChanged(filepath.Join(root, "fail-rename.css"), "x"); err == nil {
		t.Fatal("expected Rename failure")
	}
	atomicWriteRename = os.Rename

	// CreateTemp failure: read-only directory.
	ro := filepath.Join(root, "ro")
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	if err := writeFileAtomicIfChanged(filepath.Join(ro, "x.css"), "x"); err == nil {
		t.Fatal("expected CreateTemp failure on read-only dir")
	}
}

func TestScanTailwindCandidatesSingleFileReadError(t *testing.T) {
	root := t.TempDir()
	vue := filepath.Join(root, "Bad.vue")
	if err := os.WriteFile(vue, []byte(`<div class="flex"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(vue, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(vue, 0o644) })
	_, err := ScanTailwindCandidates([]string{vue})
	_ = os.Chmod(vue, 0o644)
	if err == nil {
		t.Fatal("expected read error for unreadable single-file root")
	}
}

func TestScanTailwindCandidatesStatPermissionError(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	child := filepath.Join(blocked, "x.vue")
	// Stat(child) should fail with permission denied when parent is mode 000.
	_, err := ScanTailwindCandidates([]string{child})
	_ = os.Chmod(blocked, 0o755)
	if err == nil {
		t.Log("stat permission error not observed on this runner")
	}
}

func TestEnsureChoyTailwindCSSEmptyAndStatError(t *testing.T) {
	res, err := EnsureChoyTailwindCSS("")
	if err != nil || res != nil {
		t.Fatalf("empty modulesPath => nil,nil got %#v %v", res, err)
	}

	root := t.TempDir()
	styles := filepath.Join(root, "choy_ui", "web", "styles")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	theme := filepath.Join(styles, "theme.css")
	if err := os.WriteFile(theme, []byte(`@theme { --color-primary: blue; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(styles, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(styles, 0o755) })
	_, err = EnsureChoyTailwindCSS(root)
	_ = os.Chmod(styles, 0o755)
	if err == nil {
		t.Log("permission-denied stat not observed on this runner")
	}
}

func TestGenerateChoyTailwindForModuleWriteFailure(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	styles := filepath.Join(web, "styles")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(styles, "theme.css"), []byte(`@theme { --color-primary: red; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Read-only styles dir → atomic write CreateTemp fails.
	if err := os.Chmod(styles, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(styles, 0o755) })
	_, err := GenerateChoyTailwindForModule(root)
	_ = os.Chmod(styles, 0o755)
	if err == nil {
		t.Fatal("expected write failure on read-only styles dir")
	}
}

func TestGenerateChoyTailwindForModuleScanFailure(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	styles := filepath.Join(web, "styles")
	pages := filepath.Join(web, "pages")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pages, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(styles, "theme.css"), []byte(`@theme { --color-primary: red; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(pages, "Bad.vue")
	if err := os.WriteFile(bad, []byte(`<div class="flex"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })
	_, err := GenerateChoyTailwindForModule(root)
	_ = os.Chmod(bad, 0o644)
	if err == nil {
		t.Fatal("expected scan failure for unreadable vue")
	}
}

func TestTailwindInputDigestErrors(t *testing.T) {
	d, c, err := TailwindInputDigest("")
	if err != nil || d != "" || c != "" {
		t.Fatalf("empty => empty hashes, got %q %q %v", d, c, err)
	}

	root := t.TempDir()
	styles := filepath.Join(root, "choy_ui", "web", "styles")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	theme := filepath.Join(styles, "theme.css")
	if err := os.WriteFile(theme, []byte(`@theme{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Unreadable theme → non-NotExist read error.
	if err := os.Chmod(theme, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(theme, 0o644) })
	_, _, err = TailwindInputDigest(root)
	_ = os.Chmod(theme, 0o644)
	if err == nil {
		t.Fatal("expected read error for unreadable theme.css")
	}

	// Scan error: unreadable candidate file under web/.
	pages := filepath.Join(root, "choy_ui", "web", "pages")
	if err := os.MkdirAll(pages, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(pages, "Bad.vue")
	if err := os.WriteFile(bad, []byte(`<div class="flex"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })
	_, _, err = TailwindInputDigest(root)
	_ = os.Chmod(bad, 0o644)
	if err == nil {
		t.Fatal("expected scan error for unreadable vue file")
	}
}

func TestGenerateTailwindCSSLoadCSSError(t *testing.T) {
	t.Cleanup(func() {
		choyLoadCSS = func(eng *tw.Engine, css []byte) error { return eng.LoadCSS(css) }
	})
	choyLoadCSS = func(*tw.Engine, []byte) error { return os.ErrInvalid }
	_, _, err := GenerateTailwindCSS(`@theme { --color-primary: red; }`, []string{"flex"})
	if err == nil || !strings.Contains(err.Error(), "load Tailwind dialect") {
		t.Fatalf("expected dialect load error, got %v", err)
	}
}

func TestGenerateTailwindCSSEmptyCandidates(t *testing.T) {
	css, _, err := GenerateTailwindCSS(`@theme { --color-primary: red; }`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(css, "--color-primary:") {
		t.Fatalf("dialect-only generate should still emit ThemeCSS aliases:\n%s", css)
	}
}

func TestScopeChoyUtilityCSS(t *testing.T) {
	in := "@property --tw-x { syntax: \"*\"; inherits: false; }\n\n.flex {\n  display: flex;\n}\n\n.bg-primary, .text-primary {\n  color: red;\n}\n"
	got := scopeChoyUtilityCSS(in, choyGalleryRootSelector)
	if !strings.Contains(got, "@property --tw-x") {
		t.Fatalf("expected @property preserved:\n%s", got)
	}
	if !strings.Contains(got, choyGalleryRootSelector+" .flex") {
		t.Fatalf("expected scoped .flex:\n%s", got)
	}
	if !strings.Contains(got, choyGalleryRootSelector+" .bg-primary") || !strings.Contains(got, choyGalleryRootSelector+" .text-primary") {
		t.Fatalf("expected scoped selector list:\n%s", got)
	}
	if strings.Contains(got, "\n.flex {") {
		t.Fatalf("unscoped .flex remained:\n%s", got)
	}

	media := "@media (min-width: 640px) {\n.flex {\n  display: flex;\n}\n}\n"
	got = scopeChoyUtilityCSS(media, choyGalleryRootSelector)
	if !strings.Contains(got, "@media (min-width: 640px)") {
		t.Fatalf("expected @media preserved:\n%s", got)
	}
	if !strings.Contains(got, choyGalleryRootSelector+" .flex") {
		t.Fatalf("expected nested .flex scoped inside @media:\n%s", got)
	}

	where := ":where(.dark, .dark *) .text-sm {\n  font-size: 14px;\n}\n"
	got = scopeChoyUtilityCSS(where, choyGalleryRootSelector)
	if !strings.Contains(got, ":where(.dark, .dark *)") {
		t.Fatalf("expected :where(...) kept intact:\n%s", got)
	}
	if strings.Contains(got, choyGalleryRootSelector+" .dark") {
		t.Fatalf("should not split inside :where():\n%s", got)
	}
}

func TestScopeChoyUtilityCSSEdgeBranches(t *testing.T) {
	if got := scopeChoyUtilityCSS("", choyGalleryRootSelector); got != "" {
		t.Fatalf("empty css => empty, got %q", got)
	}
	if got := scopeChoyUtilityCSS(".flex{}", ""); got != ".flex{}" {
		t.Fatalf("empty scope returns input, got %q", got)
	}
	// Unbalanced @rule / class block / trailing junk without '{'.
	if got := scopeChoyUtilityCSS("@media (x) { .a { color: red; }", choyGalleryRootSelector); !strings.Contains(got, "@media") {
		t.Fatalf("unbalanced @media should be copied through:\n%s", got)
	}
	if got := scopeChoyUtilityCSS(".flex { display: flex;", choyGalleryRootSelector); !strings.Contains(got, ".flex") {
		t.Fatalf("unbalanced class block should be copied through:\n%s", got)
	}
	if got := scopeChoyUtilityCSS("/* trailing */", choyGalleryRootSelector); got != "/* trailing */" {
		t.Fatalf("no-brace remainder: %q", got)
	}
	// Empty selector slot in list.
	got := prefixCSSSelectorList(".a,, .b", choyGalleryRootSelector)
	if !strings.Contains(got, choyGalleryRootSelector+" .a") || !strings.Contains(got, choyGalleryRootSelector+" .b") {
		t.Fatalf("empty selector slot: %q", got)
	}
	if indexCSSBlockEnd("{ color: red;") != -1 {
		t.Fatal("expected unbalanced block => -1")
	}
	if indexCSSBlockEnd("{ content: \"}\"; }") != len("{ content: \"}\"; }") {
		t.Fatal("quoted brace should not confuse block end")
	}
	if indexCSSBlockEnd("{ /* } */ color: red; }") != len("{ /* } */ color: red; }") {
		t.Fatal("commented brace should not confuse block end")
	}
	if indexCSSBlockEnd("{ /* unterminated") != -1 {
		t.Fatal("unterminated comment => -1")
	}
}

func TestSplitTopLevelSelectors(t *testing.T) {
	parts := splitTopLevelSelectors(".a, :where(.b, .c), .d")
	if len(parts) != 3 {
		t.Fatalf("got %d parts %#v", len(parts), parts)
	}
	if !strings.Contains(parts[1], ":where(.b, .c)") {
		t.Fatalf("where clause split: %#v", parts)
	}
	parts = splitTopLevelSelectors(`.a[title="hello, world"], .b`)
	if len(parts) != 2 || !strings.Contains(parts[0], `"hello, world"`) {
		t.Fatalf("quoted comma: %#v", parts)
	}
	parts = splitTopLevelSelectors(`.a[title="say \"hi\", ok"], .b`)
	if len(parts) != 2 {
		t.Fatalf("escaped quote in attr: %#v", parts)
	}
	parts = splitTopLevelSelectors(`.a[title='x,y'], .b`)
	if len(parts) != 2 {
		t.Fatalf("single-quoted attr: %#v", parts)
	}
}

func TestIndexCSSBlockEndQuotesEscapes(t *testing.T) {
	s := `{ content: "x}"; color: red; }`
	if indexCSSBlockEnd(s) != len(s) {
		t.Fatalf("quoted } must not end block, got %d want %d", indexCSSBlockEnd(s), len(s))
	}
	s = `{ content: "a\\"; }`
	if indexCSSBlockEnd(s) != len(s) {
		t.Fatalf("escaped backslash in quote: %d", indexCSSBlockEnd(s))
	}
	s = `{ content: 'a\'}b'; color: red; }`
	if indexCSSBlockEnd(s) != len(s) {
		t.Fatalf("escaped single quote: %d", indexCSSBlockEnd(s))
	}
}

func TestEnsureChoyTailwindCSSWebNotDir(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "choy_ui", "web")
	if err := os.MkdirAll(filepath.Dir(web), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(web, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := EnsureChoyTailwindCSS(root)
	if err != nil || res != nil {
		t.Fatalf("web file => nil,nil got %#v %v", res, err)
	}
}

func TestEnsureChoyTailwindCSSWebStatError(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "choy_ui")
	if err := os.MkdirAll(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	_, err := EnsureChoyTailwindCSS(root)
	_ = os.Chmod(parent, 0o755)
	if err == nil {
		t.Log("permission-denied web stat not observed on this runner")
	}
}

func TestGenerateChoyTailwindForModulePropagatesGenerateError(t *testing.T) {
	root := t.TempDir()
	styles := filepath.Join(root, "web", "styles")
	if err := os.MkdirAll(styles, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(styles, "theme.css"), []byte(`@theme { --color-primary: red; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		choyLoadCSS = func(eng *tw.Engine, css []byte) error { return eng.LoadCSS(css) }
	})
	choyLoadCSS = func(*tw.Engine, []byte) error { return os.ErrInvalid }
	if _, err := GenerateChoyTailwindForModule(root); err == nil {
		t.Fatal("expected generate error from LoadCSS failure")
	}
}

func TestEnsureChoyTailwindCSSIncompleteKit(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "choy_ui", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureChoyTailwindCSS(root)
	if err == nil || !strings.Contains(err.Error(), "dialect") {
		t.Fatalf("expected missing dialect error, got %v", err)
	}
}

func TestScopeChoyThemeCSS(t *testing.T) {
	in := ":root, :host {\n  --color-primary: red;\n}\n"
	got := scopeChoyThemeCSS(in)
	if strings.Contains(got, ":root") || strings.Contains(got, ":host") {
		t.Fatalf("expected :root/:host rebound, got:\n%s", got)
	}
	if !strings.Contains(got, choyGalleryRootSelector+" {") {
		t.Fatalf("expected gallery root theme:\n%s", got)
	}
}
