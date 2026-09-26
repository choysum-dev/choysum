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
	// Vue product SFCs often put PascalCase component tags (e.g. <Textarea>) before
	// <script setup>; without masking, the tokenizer swallows the script block.
	// Masking is limited to unquoted text inside <template> so script/style string
	// literals and attribute values keep literal "<Textarea>" unchanged.
	masked := maskPascalCaseRawTextTags(string(src))
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

// pascalCaseRawTextTag matches Vue component tags whose local names collide with
// HTML raw-text / RCDATA elements when lowercased by the tokenizer.
// Go regexp has no lookahead; the trailing delimiter is re-emitted by the replacer.
var pascalCaseRawTextTag = regexp.MustCompile(`</?(Textarea|Title|Style|Script|Noscript|Iframe|Noembed|Noframes|Xmp|Plaintext)([\s/>])`)

var sfcTemplateOpen = regexp.MustCompile(`(?i)<template\b[^>]*>`)
var sfcTemplateClose = regexp.MustCompile(`(?i)</template\s*>`)

// sfcScriptStyleBlock matches top-level <script>/<style> so a "<template>" literal
// inside their source is never treated as the SFC template region.
// Each opener is paired with its own closer (RE2 has no backreferences).
var sfcScriptStyleBlock = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>|<style\b[^>]*>.*?</style\s*>`)

const vueRawTextMaskPrefix = "VueSfcRaw"

// vueSfcHTMLParse is the HTML parse step after masking; tests may override it.
var vueSfcHTMLParse = parseWithCaseSensitive

func maskPascalCaseRawTextTags(src string) string {
	scriptStyle := sfcScriptStyleBlock.FindAllStringIndex(src, -1)
	inScriptOrStyle := func(pos int) bool {
		for _, r := range scriptStyle {
			if pos >= r[0] && pos < r[1] {
				return true
			}
		}
		return false
	}

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
		if inScriptOrStyle(openStart) {
			// Literal "<template" inside script/style — copy through and keep scanning.
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
				out.WriteString(maskPascalCaseRawTextTagsOutsideQuotes(src[bodyStart:]))
				return out.String()
			}
			closeAt := pos + nextClose[0]
			closeEnd := pos + nextClose[1]
			if nextOpen != nil {
				openAt := pos + nextOpen[0]
				if openAt < closeAt {
					nestedTag := src[openAt : pos+nextOpen[1]]
					pos = pos + nextOpen[1]
					if !isSelfClosingHTMLOpenTag(nestedTag) {
						depth++
					}
					continue
				}
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
	if n.Type == html.ElementNode && strings.HasPrefix(n.Data, vueRawTextMaskPrefix) {
		n.Data = strings.TrimPrefix(n.Data, vueRawTextMaskPrefix)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		unmaskPascalCaseRawTextTags(c)
	}
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
