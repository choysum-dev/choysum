// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"errors"
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

func TestScreenshotCurrent(t *testing.T) {
	var s *Session
	if err := s.ScreenshotCurrent(filepath.Join(t.TempDir(), "x.png")); err == nil || !strings.Contains(err.Error(), "nil session") {
		t.Fatalf("nil session: %v", err)
	}
	s = &Session{}
	if err := s.ScreenshotCurrent(filepath.Join(t.TempDir(), "x.png")); err == nil || !strings.Contains(err.Error(), "nil session") {
		t.Fatalf("nil browserCtx: %v", err)
	}

	session := startTestSession(t)
	if err := session.ScreenshotCurrent("   "); err == nil || !strings.Contains(err.Error(), "empty screenshot path") {
		t.Fatalf("empty path: %v", err)
	}
	path := filepath.Join(t.TempDir(), "current.png")
	if err := session.ScreenshotCurrent(path); err != nil {
		t.Fatalf("ScreenshotCurrent: %v", err)
	}
	if st, err := os.Stat(path); err != nil || st.Size() == 0 {
		t.Fatalf("screenshot missing: %v", err)
	}

	// MkdirAll fails when parent path is a file (new review path).
	badParent := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(badParent, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := session.ScreenshotCurrent(filepath.Join(badParent, "shot.png")); err == nil {
		t.Fatal("expected MkdirAll error")
	}

	// Race: close during FullScreenshot to exercise chromedp.Run error path.
	session2 := startTestSession(t)
	shotPath := filepath.Join(t.TempDir(), "race.png")
	go func() {
		time.Sleep(5 * time.Millisecond)
		session2.Close()
	}()
	_ = session2.ScreenshotCurrent(shotPath) // error expected; covers Run-fail when racing
}

func TestNewPageEnableNetworkError(t *testing.T) {
	session := startTestSession(t)
	old := enableNetworkForPage
	enableNetworkForPage = func(p *Page) error { return errors.New("net enable boom") }
	t.Cleanup(func() { enableNetworkForPage = old })
	if _, err := session.NewPage(); err == nil || !strings.Contains(err.Error(), "net enable boom") {
		t.Fatalf("got %v", err)
	}
}

func TestNewPageDeadBrowserContext(t *testing.T) {
	session := startTestSession(t)
	session.Close()
	if _, err := session.NewPage(); err == nil || !strings.Contains(err.Error(), "browser context dead") {
		t.Fatalf("expected dead context error, got %v", err)
	}
}

func TestScreenshotCurrentDeadContext(t *testing.T) {
	session := startTestSession(t)
	session.Close()
	if err := session.ScreenshotCurrent(filepath.Join(t.TempDir(), "dead.png")); err == nil {
		t.Fatal("expected dead browserCtx error")
	}
}

func TestWaitForFunctionChromedpErrorDeadline(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	old := pageEvaluate
	pageEvaluate = func(ctx context.Context, expr string, out *bool) error {
		return errors.New("eval always fails")
	}
	t.Cleanup(func() { pageEvaluate = old })
	if err := page.WaitForFunction("true", 80*time.Millisecond); err == nil || !strings.Contains(err.Error(), "waitForFunction") {
		t.Fatalf("expected wrapped waitForFunction error, got %v", err)
	}
}

func TestEvaluateEmptyResultAndJSONQuoteFallback(t *testing.T) {
	if got := normalizeEvaluateOut(""); got != "null" {
		t.Fatalf("empty evaluate: %q", got)
	}
	if got := normalizeEvaluateOut(`"x"`); got != `"x"` {
		t.Fatalf("non-empty: %q", got)
	}
	oldJM := jsonMarshalString
	jsonMarshalString = func(s string) ([]byte, error) { return nil, errors.New("marshal fail") }
	t.Cleanup(func() { jsonMarshalString = oldJM })
	if got := jsonQuote("x"); got != `""` {
		t.Fatalf("jsonQuote fallback: %q", got)
	}
}

func TestPageOpsErrorAndCancelPaths(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}

	if err := page.WaitForFunction("true", 0); err != nil {
		t.Fatalf("timeout<=0 should default and succeed: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><html><body><div id="ok">x</div></body></html>`))
	}))
	defer srv.Close()
	if err := page.Goto(srv.URL, "load"); err != nil {
		t.Fatal(err)
	}

	if _, err := page.Evaluate(`throw new Error("eval boom")`); err == nil {
		t.Fatal("expected evaluate error")
	}

	// Parent path is a file → MkdirAll fails.
	badShot := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(badShot, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := page.Screenshot(filepath.Join(badShot, "x.png")); err == nil {
		t.Fatal("expected mkdir screenshot error")
	}

	// Cancel mid-poll.
	go func() {
		time.Sleep(30 * time.Millisecond)
		page.Close()
		session.Close()
	}()
	if err := page.WaitForFunction("false", 2*time.Second); err == nil {
		t.Fatal("expected cancel/timeout error")
	}
}

func TestPageMethodsOnDeadContext(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	session.Close()
	if _, err := page.QueryCount("div"); err == nil {
		t.Fatal("expected QueryCount error")
	}
	if _, err := page.IsVisible("div"); err == nil {
		t.Fatal("expected IsVisible error")
	}
	if _, err := page.IsEnabled("div"); err == nil {
		t.Fatal("expected IsEnabled error")
	}
	if _, err := page.URL(); err == nil {
		t.Fatal("expected URL error")
	}
	if err := page.Screenshot(filepath.Join(t.TempDir(), "dead.png")); err == nil {
		t.Fatal("expected Screenshot error")
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
<textarea id="bio"></textarea>
<div id="ce" contenteditable="true"></div>
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
	if err := page.Fill("#bio", "hello"); err != nil {
		t.Fatalf("Fill textarea: %v", err)
	}
	if err := page.Fill("#ce", "editable"); err != nil {
		t.Fatalf("Fill contenteditable: %v", err)
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

func TestWaitForFunctionCancelBetweenPolls(t *testing.T) {
	// Mock evaluate so we reach the between-polls select without CDP; cancel ctx there.
	ctx, cancel := context.WithCancel(context.Background())
	page := &Page{ctx: ctx, cancel: cancel}
	old := pageEvaluate
	n := 0
	pageEvaluate = func(c context.Context, expr string, out *bool) error {
		n++
		*out = false
		if n == 1 {
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		}
		return nil
	}
	t.Cleanup(func() { pageEvaluate = old })
	if err := page.WaitForFunction("false", 2*time.Second); err == nil {
		t.Fatal("expected cancel between polls")
	}
}

func TestWaitForFunctionEvaluateErrorWhenCtxCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	page := &Page{ctx: ctx, cancel: func() {}}
	old := pageEvaluate
	pageEvaluate = func(c context.Context, expr string, out *bool) error {
		return errors.New("evaluate boom")
	}
	t.Cleanup(func() { pageEvaluate = old })
	if err := page.WaitForFunction("true", time.Second); err == nil {
		t.Fatal("expected ctx err from evaluate-fail path")
	}
}

func TestWaitForFunctionFunctionSource(t *testing.T) {
	session := startTestSession(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
<script>window.__ready = true;</script>
</body></html>`))
	}))
	defer srv.Close()
	if err := page.Goto(srv.URL, "load"); err != nil {
		t.Fatal(err)
	}
	// Function-source form: wrapper must invoke via typeof __v === 'function' ? __v() : __v.
	if err := page.WaitForFunction(`() => window.__ready === true`, 2*time.Second); err != nil {
		t.Fatalf("function-source WaitForFunction: %v", err)
	}
}
