// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Role matching for getByRole lives in @choysum/e2e (JS). This test locks the
// DOM contract those selectors rely on: button/option/dialog (+ accessible name).
func TestPageRoleDOMContract(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
<button>Create Account</button>
<button aria-label="Save profile">icon</button>
<div role="dialog" aria-label="Switch company">dialog body</div>
<select id="s"><option>Asia/Shanghai</option><option>UTC</option></select>
</body></html>`))
	}))
	defer srv.Close()

	if err := page.Goto(srv.URL, "domcontentloaded"); err != nil {
		t.Fatalf("Goto: %v", err)
	}

	raw, err := page.Evaluate(`(() => {
  const nameOf = (el) => {
    const al = el.getAttribute('aria-label');
    if (al) return al;
    return (el.innerText || el.textContent || '').trim();
  };
  const buttons = [...document.querySelectorAll('button,[role="button"]')].map(nameOf);
  const dialogs = [...document.querySelectorAll('[role="dialog"],dialog')].map(nameOf);
  const options = [...document.querySelectorAll('option,[role="option"]')].map(nameOf);
  return { buttons, dialogs, options };
})()`)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	s := strings.ToLower(raw)
	for _, want := range []string{"create account", "save profile", "switch company", "asia/shanghai", "utc"} {
		if !strings.Contains(s, want) {
			t.Fatalf("role DOM contract missing %q in %s", want, raw)
		}
	}
}
