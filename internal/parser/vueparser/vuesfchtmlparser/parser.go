// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"io"
	"regexp"
	"strings"

	"github.com/antchfx/htmlquery"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/net/html"
)

func RenderVueSfcFromHtmlNode(node *html.Node) (string, error) {
	var buf strings.Builder
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		err := renderNode(&buf, n)
		if err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func renderNode(w io.Writer, n *html.Node) error {
	switch n.Type {
	case html.TextNode:
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}
		return nil
	case html.ElementNode:
		if _, err := w.Write([]byte("<")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}

		// Render attributes without escaping values
		for _, attr := range n.Attr {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
			if _, err := w.Write([]byte(attr.Key)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(`="`)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(attr.Val)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(`"`)); err != nil {
				return err
			}
		}

		if _, err := w.Write([]byte(">")); err != nil {
			return err
		}

		// Recursively render child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := renderNode(w, c); err != nil {
				return err
			}
		}

		if _, err := w.Write([]byte("</")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}

		_, err := w.Write([]byte(">"))
		return err

	case html.CommentNode:
		if _, err := w.Write([]byte("<!--")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}
		if _, err := w.Write([]byte("-->")); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func ParseVueSfcToHtmlNode(r io.Reader) (scriptNodes []*html.Node, templateNode *html.Node, styleNodes []*html.Node, err error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, nil, err
	}
	// x/net/html treats <textarea>/<title>/… as raw-text elements (case-insensitive).
	// A Vue-style self-closing lowercase tag like <textarea /> is not a void element:
	// the tokenizer treats '/' as ignored and keeps reading until </textarea>, so a
	// following <script setup> is swallowed. Expand those to explicit open/close
	// pairs before parse (PascalCase <Textarea /> stays for the masker below).
	//
	// Vue product SFCs often put PascalCase component tags (e.g. <Textarea>) before
	// <script setup>; without masking, the tokenizer swallows the script block.
	// Masking is limited to unquoted text inside <template> so script/style string
	// literals and attribute values keep literal "<Textarea>" unchanged.
	// Fast path: the multi-pass masker is a no-op unless a non-lowercase
	// raw-text-named tag is present (real <script>/<textarea> stay unmasked).
	masked := expandSelfClosingLowerRawTextHTMLTags(string(src))
	if sourceNeedsPascalCaseRawTextMask(masked) {
		masked = maskPascalCaseRawTextTags(masked)
	}
	doc, err := vueSfcHTMLParse(strings.NewReader(masked))
	if err != nil {
		return nil, nil, nil, err
	}
	unmaskPascalCaseRawTextTags(doc)
	scriptNodes = htmlquery.Find(doc, "//script")
	templateNode = htmlquery.FindOne(doc, "//template")
	styleNodes = htmlquery.Find(doc, "//style")

	if len(scriptNodes) == 0 {
		return nil, nil, nil, xfmt.Errorf("script node not found")
	}

	return scriptNodes, templateNode, styleNodes, nil

}

// lowerSelfClosingRawTextTag matches Vue-style self-closing *lowercase* HTML
// raw-text / RCDATA tags. PascalCase component tags (e.g. <Textarea />) are
// intentionally excluded so the PascalCase masker can still rewrite them.
// Unquoted attrs may contain '/' (e.g. src=/a/b); the trailing \s*/> still
// anchors the self-closing marker via backtracking.
var lowerSelfClosingRawTextTag = regexp.MustCompile(
	`<(textarea|title|style|script|noscript|iframe|noembed|noframes|xmp|plaintext)((?:\s+(?:[^>"']|"[^"]*"|'[^']*')*)*)\s*/>`,
)

// expandSelfClosingLowerRawTextHTMLTags rewrites <textarea .../> (and siblings)
// to <textarea ...></textarea> so x/net/html does not treat the rest of the SFC
// as raw-text content. Matches inside script/style, HTML comments, mustaches,
// and quoted attribute values are left untouched.
func expandSelfClosingLowerRawTextHTMLTags(src string) string {
	if !strings.Contains(src, "/>") || !lowerSelfClosingRawTextTag.MatchString(src) {
		return src
	}
	scriptStyle := findScriptStyleRanges(src)
	comments := findHTMLCommentRanges(src, scriptStyle)
	seed := findMustacheAndQuotedAttrRanges(src, comments)
	skipForComments := append(append([][]int{}, scriptStyle...), seed...)
	comments = findHTMLCommentRanges(src, skipForComments)
	skipForMustache := append(append([][]int{}, scriptStyle...), comments...)
	mustacheAttr := findMustacheAndQuotedAttrRanges(src, skipForMustache)
	protected := append(append(append([][]int{}, scriptStyle...), comments...), mustacheAttr...)
	inProtected := func(pos int) bool {
		for _, r := range protected {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}
	var out strings.Builder
	out.Grow(len(src) + 32)
	last := 0
	for _, loc := range lowerSelfClosingRawTextTag.FindAllStringSubmatchIndex(src, -1) {
		start, end := loc[0], loc[1]
		if inProtected(start) || inProtected(end-1) {
			continue
		}
		// HTML/Vue: '/' is a self-closing marker when preceded by whitespace, a
		// closing quote (<tag id="foo"/>), or the tag name. A glued trailing
		// slash on an unquoted value (<iframe src=/a/b/>) belongs to that value.
		if end >= 3 && src[end-2] == '/' {
			before := src[end-3]
			switch before {
			case ' ', '\t', '\n', '\r', '"', '\'':
				// <tag attrs /> or <tag attr="..." /> / <tag attr="..."/>
			default:
				if loc[4] >= 0 && loc[5] > loc[4] {
					continue
				}
				// <textarea/> — slash immediately after the tag name.
			}
		}
		name := src[loc[2]:loc[3]]
		attrs := ""
		if loc[4] >= 0 {
			attrs = src[loc[4]:loc[5]]
		}
		out.WriteString(src[last:start])
		out.WriteByte('<')
		out.WriteString(name)
		out.WriteString(attrs)
		out.WriteString("></")
		out.WriteString(name)
		out.WriteByte('>')
		last = end
	}
	out.WriteString(src[last:])
	return out.String()
}

// pascalCaseRawTextTag matches tags whose local names collide with HTML raw-text
// / RCDATA elements when lowercased by the tokenizer (case-insensitive).
// replacePascalCaseRawTextTags leaves all-lowercase HTML tags untouched.
// Go regexp has no lookahead; the trailing delimiter is re-emitted by the replacer.
var pascalCaseRawTextTag = regexp.MustCompile(`(?i)</?(textarea|title|style|script|noscript|iframe|noembed|noframes|xmp|plaintext)([\s/>])`)

var sfcTemplateOpen = regexp.MustCompile(`(?i)<template\b(?:[^>"']|"[^"]*"|'[^']*')*>`)
var sfcTemplateClose = regexp.MustCompile(`(?i)</template\s*>`)

// sfcScriptStyleBlock matches top-level <script>/<style> so a "<template>" literal
// inside their source is never treated as the SFC template region.
// Each opener is paired with its own closer (RE2 has no backreferences).
// Opener attrs are quote-aware so a '>' inside a quoted value does not truncate the tag.
var sfcScriptStyleBlock = regexp.MustCompile(`(?is)<script\b(?:[^>"']|"[^"]*"|'[^']*')*>.*?</script\s*>|<style\b(?:[^>"']|"[^"]*"|'[^']*')*>.*?</style\s*>`)

const vueRawTextMaskPrefix = "VueSfcRaw"

// vueSfcHTMLParse is the HTML parse step after masking; tests may override it.
var vueSfcHTMLParse = parseWithCaseSensitive

// sourceNeedsPascalCaseRawTextMask reports whether s contains a raw-text-named tag
// that is not all-lowercase (those need masking; real HTML lowercase tags do not).
func sourceNeedsPascalCaseRawTextMask(s string) bool {
	for _, m := range pascalCaseRawTextTag.FindAllString(s, -1) {
		if rawTextTagNameFromMatch(m) != strings.ToLower(rawTextTagNameFromMatch(m)) {
			return true
		}
	}
	return false
}

func rawTextTagNameFromMatch(m string) string {
	body := m[:len(m)-1]
	if strings.HasPrefix(body, "</") {
		return body[2:]
	}
	return strings.TrimPrefix(body, "<")
}
func maskPascalCaseRawTextTags(src string) string {
	// Script/style ranges ignore openers that sit inside HTML comments (e.g.
	// <!-- <script> -->). Comment vs mustache/attr discovery is interleaved:
	// seed ranges from a first comment pass let a "<!--" inside a quoted attr
	// be ignored on the second comment pass; final mustache/attr discovery then
	// skips the corrected comments so an unclosed "{{" inside <!-- … --> cannot
	// run to EOF and hide later quoted attrs (including after a phantom <!--).
	scriptStyle := findScriptStyleRanges(src)
	comments := findHTMLCommentRanges(src, scriptStyle)
	seed := findMustacheAndQuotedAttrRanges(src, comments)
	skipForComments := append(append([][]int{}, scriptStyle...), seed...)
	comments = findHTMLCommentRanges(src, skipForComments)
	skipForMustache := append(append([][]int{}, scriptStyle...), comments...)
	mustacheAttr := findMustacheAndQuotedAttrRanges(src, skipForMustache)
	protected := append(append([][]int{}, comments...), mustacheAttr...)
	inRange := func(pos int, ranges [][]int) bool {
		for _, r := range ranges {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}
	inScriptOrStyle := func(pos int) bool { return inRange(pos, scriptStyle) }
	inProtected := func(pos int) bool { return inRange(pos, protected) }

	var out strings.Builder
	out.Grow(len(src) + 32)
	i := 0
	for i < len(src) {
		openLoc := sfcTemplateOpen.FindStringIndex(src[i:])
		if openLoc == nil {
			out.WriteString(src[i:])
			break
		}
		openStart := i + openLoc[0]
		openEnd := i + openLoc[1]
		if inScriptOrStyle(openStart) || inProtected(openStart) {
			// Literal "<template" inside script/style/comment/mustache/attr — copy through.
			out.WriteString(src[i:openEnd])
			i = openEnd
			continue
		}
		openTag := src[openStart:openEnd]
		if isSelfClosingHTMLOpenTag(openTag) {
			out.WriteString(src[i:openEnd])
			i = openEnd
			continue
		}
		out.WriteString(src[i:openEnd])
		bodyStart := openEnd
		depth := 1
		pos := bodyStart
		for depth > 0 {
			nextOpen := sfcTemplateOpen.FindStringIndex(src[pos:])
			nextClose := sfcTemplateClose.FindStringIndex(src[pos:])
			if nextClose == nil {
				// Unbalanced depth (e.g. "<template" literal in an attribute) must
				// not mask past the next script/style block — those bodies are
				// TextNodes and cannot be repaired by unmaskPascalCaseRawTextTags.
				end := len(src)
				for _, r := range scriptStyle {
					if r[0] >= bodyStart {
						end = r[0]
						break
					}
				}
				out.WriteString(maskPascalCaseRawTextTagsOutsideQuotes(src[bodyStart:end]))
				out.WriteString(src[end:])
				return out.String()
			}
			closeAt := pos + nextClose[0]
			closeEnd := pos + nextClose[1]
			// Prefer a nested opener that precedes this close — even when the close
			// sits inside a comment — so depth is not skipped past the opener.
			if nextOpen != nil && pos+nextOpen[0] < closeAt {
				openAt := pos + nextOpen[0]
				nestedEnd := pos + nextOpen[1]
				pos = nestedEnd
				if inProtected(openAt) {
					continue
				}
				nestedTag := src[openAt:nestedEnd]
				if !isSelfClosingHTMLOpenTag(nestedTag) {
					depth++
				}
				continue
			}
			if inProtected(closeAt) {
				pos = closeEnd
				continue
			}
			depth--
			if depth == 0 {
				out.WriteString(maskPascalCaseRawTextTagsOutsideQuotes(src[bodyStart:closeAt]))
				out.WriteString(src[closeAt:closeEnd])
				i = closeEnd
				break
			}
			pos = closeEnd
		}
	}
	return out.String()
}

// findScriptStyleRanges returns [start,end) spans of top-level <script>/<style>
// blocks whose openers are not inside HTML comments. A commented opener such as
// <!-- <script> --> must not pair with a later real </script> and swallow the
// template between them. Comment markers inside real script/style bodies are
// still ignored when comment ranges are computed (via findHTMLCommentRanges skip).
func findScriptStyleRanges(src string) [][]int {
	naiveComments := findHTMLCommentRanges(src, nil)
	var ranges [][]int
	inAccepted := func(pos int) bool {
		for _, r := range ranges {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}
	inComment := func(pos int) bool {
		for _, r := range naiveComments {
			if pos >= r[0] && pos < r[1] {
				// A marker inside an already-accepted script/style body is string
				// content, not an HTML comment, so it must not hide a real block.
				return !inAccepted(r[0])
			}
		}
		return false
	}
	searchFrom := 0
	for searchFrom < len(src) {
		m := sfcScriptStyleBlock.FindStringIndex(src[searchFrom:])
		if m == nil {
			break
		}
		start := searchFrom + m[0]
		end := searchFrom + m[1]
		if inComment(start) {
			// Reject this match and resume just past the false opener so a later
			// real <script>/<style> can still be found.
			searchFrom = start + 1
			continue
		}
		// Self-closing openers (<script />) satisfy the opener half of the
		// paired regex and can latch onto a later real </script>. Skip them.
		if openEnd := htmlOpenTagEnd(src, start); openEnd >= 0 && isSelfClosingHTMLOpenTag(src[start:openEnd+1]) {
			searchFrom = openEnd + 1
			continue
		}
		ranges = append(ranges, []int{start, end})
		searchFrom = end
	}
	return ranges
}

// htmlOpenTagEnd returns the index of the closing '>' for an HTML open tag that
// starts at start, skipping quoted attribute values. Returns -1 if unterminated.
func htmlOpenTagEnd(src string, start int) int {
	if start < 0 || start >= len(src) || src[start] != '<' {
		return -1
	}
	inQuote := byte(0)
	for j := start + 1; j < len(src); j++ {
		c := src[j]
		if inQuote != 0 {
			if c == '\\' {
				j++
				continue
			}
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			inQuote = c
			continue
		}
		if c == '>' {
			return j
		}
	}
	return -1
}

// findMustacheAndQuotedAttrRanges returns [start,end) spans that cannot hold a
// real template tag: Vue mustaches {{ … }} and quoted attribute values inside
// HTML tags. Positions inside skip (typically HTML comments) are ignored so an
// unclosed "{{" there cannot run to EOF. A literal "</template>" in returned
// spans must not affect depth.
func findMustacheAndQuotedAttrRanges(src string, skip [][]int) [][]int {
	inSkip := func(pos int) bool {
		for _, r := range skip {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}
	var ranges [][]int
	for i := 0; i < len(src); {
		if inSkip(i) {
			// Jump to the end of the covering skip span.
			end := i + 1
			for _, r := range skip {
				if i >= r[0] && i < r[1] && r[1] > end {
					end = r[1]
				}
			}
			i = end
			continue
		}
		if i+1 < len(src) && src[i] == '{' && src[i+1] == '{' {
			start := i
			j := i + 2
			for j+1 < len(src) {
				if src[j] == '}' && src[j+1] == '}' {
					ranges = append(ranges, []int{start, j + 2})
					i = j + 2
					break
				}
				j++
			}
			if j+1 >= len(src) {
				ranges = append(ranges, []int{start, len(src)})
				break
			}
			continue
		}
		if src[i] == '<' && i+1 < len(src) {
			n := src[i+1]
			// Lighter than looksLikeHTMLTagOpener so an unterminated tag still
			// ends the scan (and covers the EOF path) without treating "{{ a < b }}".
			if (n >= 'a' && n <= 'z') || (n >= 'A' && n <= 'Z') || n == '/' {
				inQuote := byte(0)
				quoteStart := 0
				j := i + 1
				for j < len(src) {
					c := src[j]
					if inQuote != 0 {
						if c == '\\' {
							j++
							if j < len(src) {
								j++
							}
							continue
						}
						if c == inQuote {
							ranges = append(ranges, []int{quoteStart, j + 1})
							inQuote = 0
							j++
							continue
						}
						j++
						continue
					}
					if c == '\'' || c == '"' || c == '`' {
						inQuote = c
						quoteStart = j
						j++
						continue
					}
					if c == '>' {
						i = j + 1
						break
					}
					j++
				}
				if j >= len(src) {
					break
				}
				continue
			}
		}
		i++
	}
	return ranges
}

// findHTMLCommentRanges returns [start,end) spans of <!-- … --> in HTML context.
// Positions inside skip (typically script/style blocks) are ignored so a "<!--"
// string literal there cannot open a comment through EOF. An unclosed comment
// in HTML context still runs to EOF.
func findHTMLCommentRanges(src string, skip [][]int) [][]int {
	inSkip := func(pos int) bool {
		for _, r := range skip {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}
	var ranges [][]int
	for i := 0; i < len(src); {
		start := strings.Index(src[i:], "<!--")
		if start < 0 {
			break
		}
		start += i
		if inSkip(start) {
			i = start + 4
			continue
		}
		endRel := strings.Index(src[start+4:], "-->")
		if endRel < 0 {
			ranges = append(ranges, []int{start, len(src)})
			break
		}
		end := start + 4 + endRel + 3
		ranges = append(ranges, []int{start, end})
		i = end
	}
	return ranges
}

// isSelfClosingHTMLOpenTag reports whether openTag (including trailing '>') ends with />.
func isSelfClosingHTMLOpenTag(openTag string) bool {
	if openTag == "" || openTag[len(openTag)-1] != '>' {
		return false
	}
	inner := strings.TrimSpace(openTag[:len(openTag)-1])
	return strings.HasSuffix(inner, "/")
}

// maskPascalCaseRawTextTagsOutsideQuotes rewrites PascalCase raw-text tags in a
// template body, skipping only quoted attribute values inside tags. Apostrophes
// in text (e.g. "User's") must not enter quote-skip mode.
func maskPascalCaseRawTextTagsOutsideQuotes(s string) string {
	var out strings.Builder
	out.Grow(len(s) + 16)
	inTag := false
	segStart := 0
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case !inTag && c == '<' && strings.HasPrefix(s[i:], "<!--"):
			// Comments are CommentNodes after parse; leave their text unmasked.
			out.WriteString(replacePascalCaseRawTextTags(s[segStart:i]))
			endRel := strings.Index(s[i+4:], "-->")
			if endRel < 0 {
				out.WriteString(s[i:])
				return out.String()
			}
			end := i + 4 + endRel + 3
			out.WriteString(s[i:end])
			i = end
			segStart = i
		case c == '<':
			if looksLikeHTMLTagOpener(s, i) {
				inTag = true
			}
			i++
		case c == '>':
			inTag = false
			i++
		case inTag && (c == '\'' || c == '"' || c == '`'):
			out.WriteString(replacePascalCaseRawTextTags(s[segStart:i]))
			quote := c
			k := i + 1
			for k < len(s) {
				if s[k] == '\\' {
					k++
					if k < len(s) {
						k++
					}
					continue
				}
				if s[k] == quote {
					k++
					break
				}
				k++
			}
			out.WriteString(s[i:k])
			i = k
			segStart = i
		default:
			i++
		}
	}
	out.WriteString(replacePascalCaseRawTextTags(s[segStart:]))
	return out.String()
}

// looksLikeHTMLTagOpener reports whether s[start] begins a real tag ('<' plus a
// name/'/'/'!' that reaches '>' before another '<', skipping quoted attrs).
// Comparisons in text like `{{ a <b }}` must not enter inTag mode.
func looksLikeHTMLTagOpener(s string, start int) bool {
	if start < 0 || start >= len(s) || s[start] != '<' || start+1 >= len(s) {
		return false
	}
	n := s[start+1]
	if !((n >= 'a' && n <= 'z') || (n >= 'A' && n <= 'Z') || n == '/' || n == '!') {
		return false
	}
	inQuote := byte(0)
	for j := start + 1; j < len(s); j++ {
		c := s[j]
		if inQuote != 0 {
			if c == '\\' {
				j++
				if j >= len(s) {
					return false
				}
				continue
			}
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			inQuote = c
			continue
		}
		if c == '<' {
			return false
		}
		if c == '>' {
			return true
		}
	}
	return false
}

func replacePascalCaseRawTextTags(s string) string {
	return pascalCaseRawTextTag.ReplaceAllStringFunc(s, func(m string) string {
		delim := m[len(m)-1:]
		body := m[:len(m)-1]
		name := rawTextTagNameFromMatch(m)
		// Real lowercase HTML raw-text/RCDATA tags must stay untouched.
		if name == strings.ToLower(name) {
			return m
		}
		if strings.HasPrefix(body, "</") {
			return "</" + vueRawTextMaskPrefix + body[2:] + delim
		}
		return "<" + vueRawTextMaskPrefix + body[1:] + delim
	})
}

func unmaskPascalCaseRawTextTags(n *html.Node) {
	if n == nil {
		return
	}
	switch n.Type {
	case html.ElementNode:
		if strings.HasPrefix(n.Data, vueRawTextMaskPrefix) {
			n.Data = strings.TrimPrefix(n.Data, vueRawTextMaskPrefix)
		}
	case html.CommentNode, html.TextNode:
		// Restore only masker-generated tag sentinels (<VueSfcRaw… / </VueSfcRaw…),
		// not a bare literal "VueSfcRaw" in author text.
		n.Data = unmaskRawTextSentinelsInPlainData(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		unmaskPascalCaseRawTextTags(c)
	}
}

// unmaskRawTextSentinelsInPlainData strips the mask prefix only where the masker
// inserted it as a tag name (after < or </).
func unmaskRawTextSentinelsInPlainData(s string) string {
	s = strings.ReplaceAll(s, "<"+vueRawTextMaskPrefix, "<")
	s = strings.ReplaceAll(s, "</"+vueRawTextMaskPrefix, "</")
	return s
}

func parseWithCaseSensitive(r io.Reader) (*html.Node, error) {
	root := &html.Node{
		Type: html.ElementNode,
		Data: "root",
	}

	z := newCasePreservingTokenizer(r)
	stack := []*html.Node{root}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return root, nil
			}
			return nil, z.Err()

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			node := &html.Node{
				Type: html.ElementNode,
				Data: string(tn),
			}

			if hasAttr {
				for {
					key, val, moreAttr := z.TagAttr()
					node.Attr = append(node.Attr, html.Attribute{
						Key: string(key),
						Val: string(val),
					})
					if !moreAttr {
						break
					}
				}
			}

			parent := stack[len(stack)-1]
			parent.AppendChild(node)

			if tt != html.SelfClosingTagToken {
				stack = append(stack, node)
			}
			continue

		case html.EndTagToken:
			stack = stack[:len(stack)-1]
			continue

		case html.TextToken:
			text := &html.Node{
				Type: html.TextNode,
				Data: string(z.Text()),
			}
			parent := stack[len(stack)-1]
			parent.AppendChild(text)
			continue

		case html.CommentToken:
			comment := &html.Node{
				Type: html.CommentNode,
				Data: string(z.Text()),
			}
			parent := stack[len(stack)-1]
			parent.AppendChild(comment)
			continue
		}
	}

}
