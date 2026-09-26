// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"errors"
	"io"
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

func TestParseVueSfcTemplateFirstWithPascalTextarea(t *testing.T) {
	// Product SFCs are template-first; <Textarea> must not eat the following script.
	source := `<template><Textarea v-model="x" /></template>
<script setup lang="ts">const x = 'ok'</script>`
	scripts, templateNode, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode: %v", err)
	}
	if len(scripts) != 1 || templateNode == nil {
		t.Fatalf("scripts=%d template=%v", len(scripts), templateNode)
	}
	rendered, err := RenderVueSfcFromHtmlNode(templateNode)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `<Textarea`) {
		t.Fatalf("expected PascalCase Textarea preserved, got %q", rendered)
	}
	if !strings.Contains(scripts[0].FirstChild.Data, "const x") {
		t.Fatalf("expected script body, got %#v", scripts[0])
	}
}

func TestParseVueSfcDoesNotMaskScriptOrAttributeLiterals(t *testing.T) {
	source := `<template>
  <div data-sample="<Textarea>" :title="'<Style>'">
    <Textarea>body</Textarea>
  </div>
</template>
<script setup lang="ts">
const label = "<Textarea>";
const other = '<Title>';
</script>`
	scripts, templateNode, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode: %v", err)
	}
	scriptText := scripts[0].FirstChild.Data
	if strings.Contains(scriptText, vueRawTextMaskPrefix) {
		t.Fatalf("script literals must not be masked, got %q", scriptText)
	}
	if !strings.Contains(scriptText, `const label = "<Textarea>"`) {
		t.Fatalf("expected preserved Textarea string literal, got %q", scriptText)
	}
	if !strings.Contains(scriptText, `const other = '<Title>'`) {
		t.Fatalf("expected preserved Title string literal, got %q", scriptText)
	}
	rendered, err := RenderVueSfcFromHtmlNode(templateNode)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered, vueRawTextMaskPrefix) {
		t.Fatalf("template must unmask element names, got %q", rendered)
	}
	if !strings.Contains(rendered, `data-sample="<Textarea>"`) {
		t.Fatalf("attribute value must stay unmasked, got %q", rendered)
	}
	if !strings.Contains(rendered, `<Textarea>`) || !strings.Contains(rendered, `</Textarea>`) {
		t.Fatalf("expected paired Textarea elements, got %q", rendered)
	}
}

func TestParseVueSfcHTMLParseError(t *testing.T) {
	prev := vueSfcHTMLParse
	t.Cleanup(func() { vueSfcHTMLParse = prev })
	vueSfcHTMLParse = func(r io.Reader) (*html.Node, error) {
		return nil, errors.New("forced parse failure")
	}
	_, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(`<template/><script setup></script>`))
	if err == nil || !strings.Contains(err.Error(), "forced parse failure") {
		t.Fatalf("expected forced parse failure, got %v", err)
	}
}

func TestUnmaskPascalCaseRawTextTagsNil(t *testing.T) {
	unmaskPascalCaseRawTextTags(nil)
}

func TestMaskPascalCaseRawTextTagsUnclosedTemplate(t *testing.T) {
	src := `<template><Textarea />`
	got := maskPascalCaseRawTextTags(src)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("expected mask in unclosed template, got %q", got)
	}
	src = `<div/><script>const x = "<Textarea>"</script>`
	got = maskPascalCaseRawTextTags(src)
	if strings.Contains(got, vueRawTextMaskPrefix) {
		t.Fatalf("script outside template must not be masked, got %q", got)
	}
}

func TestMaskPascalCaseRawTextTagsNestedSlotTemplates(t *testing.T) {
	src := `<template>
  <Shell>
    <template #header><span/></template>
    <Textarea v-model="x" />
  </Shell>
</template>
<script setup>const x = "<Textarea>"</script>`
	got := maskPascalCaseRawTextTags(src)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("root-template Textarea must be masked, got %q", got)
	}
	if strings.Contains(got, `const x = "<`+vueRawTextMaskPrefix) {
		t.Fatalf("script literal must stay unmasked, got %q", got)
	}
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scripts[0].FirstChild.Data, `const x = "<Textarea>"`) {
		t.Fatalf("parsed script corrupted: %q", scripts[0].FirstChild.Data)
	}
}

func TestMaskPascalCaseRawTextTagsQuoteInTemplateAttr(t *testing.T) {
	src := `<template data-sample="a > b">
  <Textarea v-model="x" />
</template>
<script setup>const x = 'ok'</script>`
	got := maskPascalCaseRawTextTags(src)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea inside template with '>' in attr must mask, got %q", got)
	}
	if !strings.Contains(got, `data-sample="a > b"`) {
		t.Fatalf("template opener attr must stay intact, got %q", got)
	}
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 1 {
		t.Fatalf("expected script, got %d", len(scripts))
	}
}

func TestMaskPascalCaseRawTextTagsOutsideQuotesEscapes(t *testing.T) {
	// Escaped quote inside an attribute must not end the string early.
	in := `<span title="say \"hi\""></span><Textarea/>`
	got := maskPascalCaseRawTextTagsOutsideQuotes(in)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea outside quotes must mask, got %q", got)
	}
	if !strings.Contains(got, `title="say \"hi\""`) {
		t.Fatalf("escaped quotes must stay intact, got %q", got)
	}
	// Lone trailing backslash inside an attribute is skipped safely (no panic).
	got = maskPascalCaseRawTextTagsOutsideQuotes(`<div title="ok"><span title="x\`)
	if !strings.Contains(got, vueRawTextMaskPrefix) && !strings.Contains(got, `<div`) {
		t.Fatalf("expected stable copy for trailing backslash, got %q", got)
	}
}

func TestMaskPascalCaseRawTextTagsApostropheInText(t *testing.T) {
	source := `<template>
  <p>User's note</p>
  <Textarea v-model="x" />
</template>
<script setup lang="ts">const x = 'ok'</script>`
	scripts, templateNode, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("apostrophe must not leave Textarea unmasked (script swallowed), scripts=%d", len(scripts))
	}
	rendered, err := RenderVueSfcFromHtmlNode(templateNode)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "User's note") || !strings.Contains(rendered, `<Textarea`) {
		t.Fatalf("expected apostrophe text + Textarea, got %q", rendered)
	}
}

func TestMaskPascalCaseRawTextTagsBareLessThanInText(t *testing.T) {
	source := `<template>
  <p>{{ a < b }}'s note</p>
  <Textarea v-model="x" />
</template>
<script setup>const x = 'ok'</script>`
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("bare '<' in text must not swallow script, scripts=%d", len(scripts))
	}
	got := maskPascalCaseRawTextTags(source)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea after bare '<' must still mask, got %q", got)
	}
}

func TestMaskPascalCaseRawTextTagsLessThanLetterInText(t *testing.T) {
	// `{{ a <b <c }}` starts with '<'+'b' but hits another '<' before '>', so it is not a tag.
	source := `<template>
  <p>{{ a <b <c }}</p>
  <Textarea v-model="x" />
</template>
<script setup>const x = 'ok'</script>`
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseVueSfcToHtmlNode: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("'<b <c' comparison must not swallow script, scripts=%d", len(scripts))
	}
	got := maskPascalCaseRawTextTags(source)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea after '<b <c' comparison must still mask, got %q", got)
	}
	// Attribute values may contain '<' without breaking real tag detection.
	attr := `<div title="a < b"><Textarea/></div>`
	got = maskPascalCaseRawTextTagsOutsideQuotes(attr)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea after attr with '<' must mask, got %q", got)
	}
}

func TestLooksLikeHTMLTagOpenerEdges(t *testing.T) {
	if looksLikeHTMLTagOpener("", 0) || looksLikeHTMLTagOpener("x", 0) || looksLikeHTMLTagOpener("<", 0) {
		t.Fatal("empty / non-tag / lone '<' must be false")
	}
	if looksLikeHTMLTagOpener("<x", -1) || looksLikeHTMLTagOpener("<x", 99) {
		t.Fatal("out-of-range start must be false")
	}
	if looksLikeHTMLTagOpener("{{ a < 3 }}", strings.IndexByte("{{ a < 3 }}", '<')) {
		t.Fatal("'<' followed by space must be false")
	}
	if !looksLikeHTMLTagOpener("<div>", 0) || !looksLikeHTMLTagOpener("</div>", 0) || !looksLikeHTMLTagOpener("<!--x-->", 0) {
		t.Fatal("normal open/close/comment openers must be true")
	}
	if looksLikeHTMLTagOpener("<b <c>", 0) {
		t.Fatal("second '<' before '>' must be false")
	}
	if !looksLikeHTMLTagOpener(`<div title="a\\">`, 0) {
		t.Fatal("escaped backslash in attr must still look like a tag")
	}
	if looksLikeHTMLTagOpener(`<div title="x\`, 0) {
		t.Fatal("unclosed escaped attr must be false")
	}
	if looksLikeHTMLTagOpener("<div", 0) || looksLikeHTMLTagOpener("<div title='x'", 0) {
		t.Fatal("tag never closed with '>' must be false")
	}
}

func TestMaskPascalCaseRawTextTagsSkipsScriptEmbeddedTemplate(t *testing.T) {
	source := `<template><div/></template>
<script setup lang="ts">
const fixture = "<template><Textarea/></template>";
</script>`
	got := maskPascalCaseRawTextTags(source)
	if strings.Contains(got, vueRawTextMaskPrefix) {
		t.Fatalf("script-embedded template must not be masked, got %q", got)
	}
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scripts[0].FirstChild.Data, `<template><Textarea/></template>`) {
		t.Fatalf("script fixture corrupted: %q", scripts[0].FirstChild.Data)
	}
}

func TestMaskPascalCaseRawTextTagsSelfClosingSlot(t *testing.T) {
	source := `<template>
  <Shell>
    <template #header />
    <Textarea v-model="x" />
  </Shell>
</template>
<script setup>const x = "<Textarea>"</script>`
	got := maskPascalCaseRawTextTags(source)
	if !strings.Contains(got, vueRawTextMaskPrefix+"Textarea") {
		t.Fatalf("Textarea after self-closing slot must still mask, got %q", got)
	}
	if strings.Contains(got, `const x = "<`+vueRawTextMaskPrefix) {
		t.Fatalf("script must stay unmasked after self-closing slot, got %q", got)
	}
	scripts, _, _, err := ParseVueSfcToHtmlNode(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scripts[0].FirstChild.Data, `const x = "<Textarea>"`) {
		t.Fatalf("parsed script corrupted: %q", scripts[0].FirstChild.Data)
	}
}

func TestIsSelfClosingHTMLOpenTagEdges(t *testing.T) {
	if isSelfClosingHTMLOpenTag("") {
		t.Fatal("empty tag is not self-closing")
	}
	if isSelfClosingHTMLOpenTag("<template") {
		t.Fatal("tag without '>' is not self-closing")
	}
	if !isSelfClosingHTMLOpenTag("<template />") {
		t.Fatal("expected self-closing template")
	}
	if isSelfClosingHTMLOpenTag("<template>") {
		t.Fatal("open template must not count as self-closing")
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
