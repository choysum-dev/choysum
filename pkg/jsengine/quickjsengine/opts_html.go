// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

var (
	htmlSanitizePolicy     *bluemonday.Policy
	htmlSanitizePolicyOnce sync.Once
)

// tipTapAlignedSanitizePolicy returns the P1 UGC-strict allowlist aligned with
// TipTap StarterKit + Link (no table / image).
func tipTapAlignedSanitizePolicy() *bluemonday.Policy {
	htmlSanitizePolicyOnce.Do(func() {
		p := bluemonday.NewPolicy()
		p.AllowElements(
			"p", "br", "hr",
			"h1", "h2", "h3", "h4", "h5", "h6",
			"strong", "b", "em", "i", "u", "s", "strike", "del",
			"ul", "ol", "li",
			"blockquote",
			"code", "pre",
			"a",
		)
		p.AllowAttrs("href").OnElements("a")
		// Keep target but force noopener/noreferrer after sanitize (see forceBlankTargetRel).
		p.AllowAttrs("target").OnElements("a")
		p.AllowAttrs("rel").OnElements("a")
		p.AllowAttrs("class").OnElements("code", "pre")
		p.AllowURLSchemes("http", "https", "mailto")
		p.RequireParseableURLs(true)
		p.AllowStandardURLs()
		htmlSanitizePolicy = p
	})
	return htmlSanitizePolicy
}

func hasRelToken(rel string, token string) bool {
	for _, part := range strings.Fields(strings.ToLower(rel)) {
		if part == token {
			return true
		}
	}
	return false
}

// forceBlankTargetRel ensures target=_blank links cannot keep window.opener control.
func forceBlankTargetRel(fragment string) string {
	if fragment == "" || !strings.Contains(strings.ToLower(fragment), "target") {
		return fragment
	}
	doc, err := html.Parse(strings.NewReader("<div id=\"choysum-html-root\">" + fragment + "</div>"))
	if err != nil {
		return fragment
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			target := ""
			relIdx := -1
			for i := range n.Attr {
				switch strings.ToLower(n.Attr[i].Key) {
				case "target":
					target = strings.TrimSpace(n.Attr[i].Val)
				case "rel":
					relIdx = i
				}
			}
			if strings.EqualFold(target, "_blank") {
				rel := ""
				if relIdx >= 0 {
					rel = n.Attr[relIdx].Val
				}
				if !hasRelToken(rel, "noopener") {
					rel = strings.TrimSpace(rel + " noopener")
				}
				if !hasRelToken(rel, "noreferrer") {
					rel = strings.TrimSpace(rel + " noreferrer")
				}
				if relIdx >= 0 {
					n.Attr[relIdx].Val = rel
				} else {
					n.Attr = append(n.Attr, html.Attribute{Key: "rel", Val: rel})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	var root *html.Node
	var findRoot func(*html.Node)
	findRoot = func(n *html.Node) {
		if root != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "choysum-html-root" {
					root = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findRoot(c)
		}
	}
	findRoot(doc)
	if root == nil {
		return fragment
	}
	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return fragment
		}
	}
	return buf.String()
}

// SanitizeHTML applies the P1 default HTML allowlist (authoritative write-path policy).
func SanitizeHTML(dirty string) string {
	return forceBlankTargetRel(tipTapAlignedSanitizePolicy().Sanitize(dirty))
}

func sanitizeHTML(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("sanitize requires a string argument"))
	}
	if !args[0].IsString() {
		return ctx.ThrowError(fmt.Errorf("sanitize requires a string argument"))
	}
	return ctx.String(SanitizeHTML(args[0].String()))
}

// WithHtml registers $choysum.html.sanitize in the JavaScript environment.
func WithHtml() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		htmlObj := jse.Ctx.Object()
		htmlObj.Set("sanitize", jse.Ctx.Function(sanitizeHTML))

		choysumObj.Set("html", htmlObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}
