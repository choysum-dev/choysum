// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"fmt"
	"sync"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/microcosm-cc/bluemonday"
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

// SanitizeHTML applies the P1 default HTML allowlist (authoritative write-path policy).
func SanitizeHTML(dirty string) string {
	return tipTapAlignedSanitizePolicy().Sanitize(dirty)
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
