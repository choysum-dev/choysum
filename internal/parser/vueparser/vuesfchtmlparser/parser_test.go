// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func TestParseVueSfcToHtmlNodeAndRenderPreserveCase(t *testing.T) {
	source := `<template><Div data-ID="A"><span>Hello</span><!--note--></Div></template><script setup lang="ts">const A=1</script><style scoped lang="scss">.x{color:red;}</style>`
	scripts, templateNode, styles, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode error: %v", err)
	}
	if len(scripts) != 1 || len(styles) != 1 || templateNode == nil {
		t.Fatalf("unexpected parsed nodes: scripts=%d styles=%d template=%v", len(scripts), len(styles), templateNode)
	}
	if scripts[0].Attr[0].Key != "setup" || styles[0].Attr[0].Key != "scoped" {
		t.Fatalf("unexpected attrs: script=%#v style=%#v", scripts[0].Attr, styles[0].Attr)
	}
	rendered, err := RenderVueSfcFromHtmlNode(templateNode)
	if err != nil {
		t.Fatalf("RenderVueSfcFromHtmlNode error: %v", err)
	}
	if !strings.Contains(rendered, `<Div data-ID="A">`) || !strings.Contains(rendered, `<!--note-->`) {
		t.Fatalf("rendered template did not preserve case/comment: %q", rendered)
	}
}

func TestParseVueSfcToHtmlNodeRequiresScript(t *testing.T) {
	_, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(`<template><div/></template>`))
	if err == nil || !strings.Contains(err.Error(), "script node not found") {
		t.Fatalf("expected missing script error, got %v", err)
	}
}

func TestCloneNodeCreatesDetachedDeepCopy(t *testing.T) {
	original, err := htmlquery.Parse(strings.NewReader(`<root><div id="a"><span>text</span></div></root>`))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	div := htmlquery.FindOne(original, "//div")
	clone := CloneNode(div)
	if clone == nil || clone == div || clone.FirstChild == div.FirstChild {
		t.Fatal("expected deep clone")
	}
	clone.Attr[0].Val = "b"
	if div.Attr[0].Val != "a" {
		t.Fatal("mutating clone should not affect original")
	}
	if CloneNode(nil) != nil {
		t.Fatal("expected cloning nil to return nil")
	}
}

func TestApplyXPathToTemplateSupportsPositions(t *testing.T) {
	source, _ := htmlquery.Parse(strings.NewReader(`<template><div id="root"><p id="target">x</p><section/></div></template>`))
	template, _ := htmlquery.Parse(strings.NewReader(`<template>
<xpath expr="//*[@id='target']" position="before"><span id="before">b</span></xpath>
<xpath expr="//*[@id='target']" position="after"><span id="after">a</span></xpath>
<xpath expr="//*[@id='root']" position="inside"><em id="inside">i</em></xpath>
<xpath expr="//*[@id='target']" position="attribute" attr-name="class" attr-value="hot"></xpath>
</template>`))

	merged, err := ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), htmlquery.FindOne(template, "//template"))
	if err != nil {
		t.Fatalf("ApplyXPathToTemplate error: %v", err)
	}
	rendered, err := RenderVueSfcFromHtmlNode(merged)
	if err != nil {
		t.Fatalf("render merged: %v", err)
	}
	for _, want := range []string{`<span id="before">b</span>`, `<span id="after">a</span>`, `<em id="inside">i</em>`, `<p id="target" class="hot">x</p>`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered merge missing %q in %q", want, rendered)
		}
	}
}

func TestApplyXPathToTemplateReplaceAndErrors(t *testing.T) {
	source, _ := htmlquery.Parse(strings.NewReader(`<template><div><p id="target">x</p></div></template>`))
	replaceTemplate, _ := htmlquery.Parse(strings.NewReader(`<template><xpath expr="//*[@id='target']" position="replace"><strong id="new">n</strong></xpath></template>`))
	merged, err := ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), htmlquery.FindOne(replaceTemplate, "//template"))
	if err != nil {
		t.Fatalf("replace apply error: %v", err)
	}
	rendered, _ := RenderVueSfcFromHtmlNode(merged)
	if strings.Contains(rendered, `id="target"`) || !strings.Contains(rendered, `<strong id="new">n</strong>`) {
		t.Fatalf("unexpected replace result: %q", rendered)
	}

	badPosition, _ := htmlquery.Parse(strings.NewReader(`<template><xpath expr="//*[@id='target']" position="unknown"><x/></xpath></template>`))
	_, err = ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), htmlquery.FindOne(badPosition, "//template"))
	if err == nil || !strings.Contains(err.Error(), "unsupported position") {
		t.Fatalf("expected unsupported position error, got %v", err)
	}

	missingNode, _ := htmlquery.Parse(strings.NewReader(`<template><xpath expr="//*[@id='missing']"><x/></xpath></template>`))
	_, err = ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), htmlquery.FindOne(missingNode, "//template"))
	if err == nil || !strings.Contains(err.Error(), "no node found") {
		t.Fatalf("expected missing node error, got %v", err)
	}

	attrTemplate, _ := htmlquery.Parse(strings.NewReader(`<template><xpath expr="//*[@id='target']" position="attribute"></xpath></template>`))
	_, err = ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), htmlquery.FindOne(attrTemplate, "//template"))
	if err == nil || !strings.Contains(err.Error(), "attrName/attr-name") {
		t.Fatalf("expected missing attrName error, got %v", err)
	}
}

func TestApplyXPathToTemplateNilTemplateReturnsClone(t *testing.T) {
	source, _ := htmlquery.Parse(strings.NewReader(`<template><div id="x">x</div></template>`))
	merged, err := ApplyXPathToTemplate(htmlquery.FindOne(source, "//template"), nil)
	if err != nil {
		t.Fatalf("ApplyXPathToTemplate nil template error: %v", err)
	}
	if merged == htmlquery.FindOne(source, "//template") {
		t.Fatal("expected cloned node when template xpath node is nil")
	}
}

func TestRenderNodeHandlesTextElementAndComment(t *testing.T) {
	root := &html.Node{Type: html.ElementNode, Data: "root"}
	root.AppendChild(&html.Node{Type: html.TextNode, Data: "hello"})
	root.AppendChild(&html.Node{Type: html.CommentNode, Data: "note"})
	root.AppendChild(&html.Node{Type: html.ElementNode, Data: "span", Attr: []html.Attribute{{Key: "x", Val: "1"}}})
	out, err := RenderVueSfcFromHtmlNode(root)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	for _, want := range []string{"hello", "<!--note-->", `<span x="1"></span>`} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q in %q", want, out)
		}
	}
}
