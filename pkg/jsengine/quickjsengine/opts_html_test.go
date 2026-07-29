// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
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

func TestSanitizeHTMLForcesNoopenerOnBlankTargets(t *testing.T) {
	got := SanitizeHTML(`<a href="https://example.com" target="_blank">x</a>`)
	if !strings.Contains(got, `target="_blank"`) {
		t.Fatalf("expected target preserved, got %q", got)
	}
	if !strings.Contains(got, "noopener") || !strings.Contains(got, "noreferrer") {
		t.Fatalf("expected rel noopener noreferrer, got %q", got)
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

func TestForceBlankTargetRelBranches(t *testing.T) {
	if got := forceBlankTargetRel(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := forceBlankTargetRel(`<p>no-target</p>`); got != `<p>no-target</p>` {
		t.Fatalf("no target early return: %q", got)
	}
	if got := forceBlankTargetRel(`<a href="https://example.com" target="_self">x</a>`); !strings.Contains(got, `target="_self"`) {
		t.Fatalf("non-blank target should stay untouched: %q", got)
	}
	if strings.Contains(strings.ToLower(forceBlankTargetRel(`<a href="https://example.com" target="_self">x</a>`)), "noopener") {
		t.Fatal("non-blank target should not force noopener")
	}

	noRel := forceBlankTargetRel(`<a href="https://example.com" target="_blank">x</a>`)
	if !strings.Contains(noRel, "noopener") || !strings.Contains(noRel, "noreferrer") {
		t.Fatalf("missing rel should be added: %q", noRel)
	}

	onlyNoopener := forceBlankTargetRel(`<a href="https://example.com" target="_blank" rel="noopener">x</a>`)
	if !strings.Contains(onlyNoopener, "noreferrer") || !hasRelToken("noopener noreferrer", "noopener") {
		t.Fatalf("should append noreferrer: %q", onlyNoopener)
	}

	onlyNoreferrer := forceBlankTargetRel(`<a href="https://example.com" target="_blank" rel="noreferrer">x</a>`)
	if !strings.Contains(onlyNoreferrer, "noopener") || !strings.Contains(onlyNoreferrer, "noreferrer") {
		t.Fatalf("should append noopener: %q", onlyNoreferrer)
	}

	both := forceBlankTargetRel(`<a href="https://example.com" target="_blank" rel="noopener noreferrer">x</a>`)
	if strings.Count(strings.ToLower(both), "noopener") != 1 {
		t.Fatalf("should not duplicate noopener: %q", both)
	}

	upper := forceBlankTargetRel(`<a href="https://example.com" TARGET="_Blank" REL="nofollow">x</a>`)
	if !strings.Contains(upper, "noopener") || !strings.Contains(upper, "noreferrer") {
		t.Fatalf("case-insensitive target/rel attrs: %q", upper)
	}

	if !hasRelToken("NOFOLLOW NoOpener", "noopener") {
		t.Fatal("hasRelToken should be case-insensitive")
	}
	if hasRelToken("nofollow", "noopener") {
		t.Fatal("hasRelToken false negative")
	}
}

func TestFindHTMLBodyHelpers(t *testing.T) {
	if findHTMLBody(nil) != nil {
		t.Fatal("nil node")
	}
	orphan := &html.Node{Type: html.ElementNode, Data: "div"}
	if findHTMLBody(orphan) != nil {
		t.Fatal("orphan without body")
	}
	bodyRoot := &html.Node{Type: html.ElementNode, Data: "body"}
	if findHTMLBody(bodyRoot) != bodyRoot {
		t.Fatal("expected direct body node")
	}
	nested := &html.Node{
		Type: html.ElementNode,
		Data: "html",
		FirstChild: &html.Node{
			Type: html.ElementNode,
			Data: "div",
			NextSibling: &html.Node{
				Type: html.ElementNode,
				Data: "body",
			},
		},
	}
	if findHTMLBody(nested) == nil || findHTMLBody(nested).Data != "body" {
		t.Fatal("expected body among siblings")
	}
	doc, _ := html.Parse(strings.NewReader("<p>x</p>"))
	if findHTMLBody(doc) == nil {
		t.Fatal("expected body from Parse")
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

func TestWithHtmlMergesExistingChoysumObject(t *testing.T) {
	engine := newTestQuickjsEngine(t, WithCrypto(), WithHtml())
	if !evalBool(t, engine, `typeof $choysum.crypto.hashPassword === "function" && typeof $choysum.html.sanitize === "function"`) {
		t.Fatal("expected crypto + html on shared $choysum")
	}
}
