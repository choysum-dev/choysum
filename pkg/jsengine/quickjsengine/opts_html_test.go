// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"strings"
	"testing"
)

func TestSanitizeHTMLStripsDangerousMarkup(t *testing.T) {
	got := SanitizeHTML(`<script>alert(1)</script><p onclick="x">Hello</p><a href="javascript:alert(1)">x</a>`)
	if strings.Contains(got, "<script") || strings.Contains(got, "onclick") || strings.Contains(got, "javascript:") {
		t.Fatalf("dangerous markup survived: %q", got)
	}
	if !strings.Contains(got, "Hello") {
		t.Fatalf("expected safe content preserved, got %q", got)
	}
}

func TestSanitizeHTMLKeepsTipTapBasics(t *testing.T) {
	in := `<h2>Title</h2><p>Hello <strong>world</strong></p><ul><li>one</li></ul><a href="https://example.com">link</a>`
	got := SanitizeHTML(in)
	for _, want := range []string{"<h2>", "<strong>", "<ul>", "<li>", `href="https://example.com"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in sanitized output %q", want, got)
		}
	}
}

func TestSanitizeHTMLDropsTableAndImage(t *testing.T) {
	got := SanitizeHTML(`<p>x</p><table><tr><td>cell</td></tr></table><img src="x.png" alt="a">`)
	if strings.Contains(got, "<table") || strings.Contains(got, "<img") {
		t.Fatalf("table/image should be stripped: %q", got)
	}
	if !strings.Contains(got, "x") {
		t.Fatalf("expected paragraph text preserved, got %q", got)
	}
}

func TestWithHtmlExposesSanitize(t *testing.T) {
	engine := newTestQuickjsEngine(t, WithHtml())

	got := evalString(t, engine, `$choysum.html.sanitize("<script>x</script><p>ok</p>")`)
	if strings.Contains(got, "<script") {
		t.Fatalf("script survived via bridge: %q", got)
	}
	if !strings.Contains(got, "ok") {
		t.Fatalf("expected safe content via bridge, got %q", got)
	}

	errText := evalString(t, engine, `(() => { try { $choysum.html.sanitize(); return ''; } catch (e) { return String(e); } })()`)
	if !strings.Contains(errText, "sanitize requires a string argument") {
		t.Fatalf("unexpected sanitize arity error: %q", errText)
	}
	errText = evalString(t, engine, `(() => { try { $choysum.html.sanitize(1); return ''; } catch (e) { return String(e); } })()`)
	if !strings.Contains(errText, "sanitize requires a string argument") {
		t.Fatalf("unexpected sanitize type error: %q", errText)
	}
}
