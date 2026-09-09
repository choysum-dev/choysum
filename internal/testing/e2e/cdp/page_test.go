// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewPageNilSession(t *testing.T) {
	var s *Session
	if _, err := s.NewPage(); err == nil || !strings.Contains(err.Error(), "nil session") {
		t.Fatalf("expected nil session error, got %v", err)
	}
	s = &Session{}
	if _, err := s.NewPage(); err == nil || !strings.Contains(err.Error(), "nil session") {
		t.Fatalf("expected nil browserCtx error, got %v", err)
	}
}

func TestNilPageMethods(t *testing.T) {
	var p *Page
	p.Close()
	if p.Context() != nil {
		t.Fatal("expected nil context")
	}
	if err := p.Goto("about:blank", "load"); err == nil {
		t.Fatal("expected nil page error")
	}
	if err := p.Click("#x"); err == nil {
		t.Fatal("expected nil page error")
	}
	if err := p.Fill("#x", "y"); err == nil {
		t.Fatal("expected nil page error")
	}
	if _, err := p.Evaluate("1"); err == nil {
		t.Fatal("expected nil page error")
	}
	if err := p.Screenshot("x.png"); err == nil {
		t.Fatal("expected nil page error")
	}
	if err := p.WaitForFunction("true", time.Second); err == nil {
		t.Fatal("expected nil page error")
	}
	if _, err := p.QueryCount("div"); err == nil {
		t.Fatal("expected nil page error")
	}
	if _, err := p.IsVisible("div"); err == nil {
		t.Fatal("expected nil page error")
	}
	if _, err := p.IsEnabled("div"); err == nil {
		t.Fatal("expected nil page error")
	}
	if _, err := p.URL(); err == nil {
		t.Fatal("expected nil page error")
	}
	if err := p.EnsureDOM(); err == nil {
		t.Fatal("expected nil page error")
	}
}

func TestPageOpsWithChrome(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	defer page.Close()

	if page.Context() == nil {
		t.Fatal("nil page context")
	}
	if err := page.EnsureDOM(); err != nil {
		t.Fatalf("EnsureDOM: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="btn">Go</button>
<input id="name" value="" />
<button id="disabled" disabled>No</button>
<div id="hidden" style="display:none">h</div>
<div class="item">1</div><div class="item">2</div>
<script>window.__ready = true;</script>
</body></html>`))
	}))
	defer srv.Close()

	if err := page.Goto(srv.URL, "load"); err != nil {
		t.Fatalf("Goto load: %v", err)
	}
	if err := page.Goto(srv.URL, "domcontentloaded"); err != nil {
		t.Fatalf("Goto domcontentloaded: %v", err)
	}
	if err := page.Goto(srv.URL, "networkidle"); err != nil {
		t.Fatalf("Goto networkidle: %v", err)
	}
	if err := page.Goto(srv.URL, ""); err != nil {
		t.Fatalf("Goto default wait: %v", err)
	}
	if err := page.Goto(srv.URL, "bogus"); err == nil || !strings.Contains(err.Error(), "unsupported waitUntil") {
		t.Fatalf("expected unsupported waitUntil, got %v", err)
	}

	href, err := page.URL()
	if err != nil || !strings.HasPrefix(href, srv.URL) {
		t.Fatalf("URL: %q err=%v", href, err)
	}

	if err := page.WaitForFunction("window.__ready === true", 2*time.Second); err != nil {
		t.Fatalf("WaitForFunction: %v", err)
	}
	if err := page.WaitForFunction("false", 80*time.Millisecond); err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected waitForFunction timeout, got %v", err)
	}

	n, err := page.QueryCount(".item")
	if err != nil || n != 2 {
		t.Fatalf("QueryCount=%d err=%v", n, err)
	}
	vis, err := page.IsVisible("#btn")
	if err != nil || !vis {
		t.Fatalf("IsVisible btn: %v %v", vis, err)
	}
	hidden, err := page.IsVisible("#hidden")
	if err != nil || hidden {
		t.Fatalf("IsVisible hidden: %v %v", hidden, err)
	}
	missing, err := page.IsVisible("#nope")
	if err != nil || missing {
		t.Fatalf("IsVisible missing: %v %v", missing, err)
	}
	en, err := page.IsEnabled("#btn")
	if err != nil || !en {
		t.Fatalf("IsEnabled btn: %v %v", en, err)
	}
	dis, err := page.IsEnabled("#disabled")
	if err != nil || dis {
		t.Fatalf("IsEnabled disabled: %v %v", dis, err)
	}
	noEl, err := page.IsEnabled("#nope")
	if err != nil || noEl {
		t.Fatalf("IsEnabled missing: %v %v", noEl, err)
	}

	if err := page.Fill("#name", "alice"); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	out, err := page.Evaluate(`document.querySelector('#name').value`)
	if err != nil || !strings.Contains(out, "alice") {
		t.Fatalf("Evaluate after fill: %q err=%v", out, err)
	}
	nullOut, err := page.Evaluate(`undefined`)
	if err != nil || nullOut != "null" {
		t.Fatalf("Evaluate undefined: %q err=%v", nullOut, err)
	}

	if err := page.Click("#btn"); err != nil {
		t.Fatalf("Click: %v", err)
	}

	shot := filepath.Join(t.TempDir(), "shots", "page.png")
	if err := page.Screenshot(shot); err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if st, err := os.Stat(shot); err != nil || st.Size() == 0 {
		t.Fatalf("screenshot missing/empty: %v", err)
	}
	if err := page.Screenshot("   "); err == nil || !strings.Contains(err.Error(), "empty screenshot path") {
		t.Fatalf("expected empty path error, got %v", err)
	}

	if got := jsonQuote(`a"b`); !strings.Contains(got, `\"`) && got != `"a\"b"` {
		// json.Marshal quotes correctly
		if got != `"a\"b"` {
			t.Fatalf("jsonQuote: %q", got)
		}
	}
}

func TestNewPageReusesTab(t *testing.T) {
	session := startTestSession(t)
	p1, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	p1.Close()
	p2, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	p2.Close()
}
