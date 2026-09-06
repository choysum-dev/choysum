// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestDocumentMinPolyfill_DecodeEntitiesOrder(t *testing.T) {
	eng, err := quickjsengine.NewFactory(
		quickjsengine.WithScript(&jsengine.JsScript{
			FileName: "polyfills/document-min.js",
			Content:  documentMinPolyfill,
		}),
	)()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = eng.Close() })
	qje := eng.(*quickjsengine.QuickjsEngine)

	eval := func(expr string) string {
		t.Helper()
		v := qje.Ctx.Eval(expr)
		defer v.Free()
		if qje.Ctx.Exception() != nil {
			t.Fatalf("Eval(%q): %v", expr, qje.Ctx.Exception())
		}
		return v.String()
	}

	// Escaped numeric entity must stay literal (amp before numeric would decode to "A").
	got := eval(`(() => {
		var el = document.createElement("div");
		el.innerHTML = "&amp;#65;";
		return el.textContent;
	})()`)
	if got != "&#65;" {
		t.Fatalf("escaped numeric entity: got %q want &#65;", got)
	}

	got = eval(`(() => {
		var el = document.createElement("div");
		el.innerHTML = "A &amp; B &#x41;";
		return el.textContent;
	})()`)
	if got != "A & B A" {
		t.Fatalf("mixed entities: got %q", got)
	}
}
