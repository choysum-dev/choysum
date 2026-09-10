// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
)

// enableNetworkForPage enables the CDP Network domain; tests may override.
var enableNetworkForPage = EnableNetwork

// emptyEvaluateJSON is returned when CDP yields an empty Evaluate string.
var emptyEvaluateJSON = "null"

// pageEvaluate runs a chromedp Evaluate; tests may override to force errors.
var pageEvaluate = func(ctx context.Context, expr string, out *bool) error {
	return chromedp.Run(ctx, chromedp.Evaluate(expr, out))
}

// Page is a browser tab owned by a Session.
type Page struct {
	session *Session
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPage returns the session's primary tab.
// E2E runs workers=1; each test reuses one tab (about:blank between tests)
// instead of spawning additional targets (which can cancel the root context).
func (s *Session) NewPage() (*Page, error) {
	if s == nil || s.browserCtx == nil {
		return nil, fmt.Errorf("cdp: nil session")
	}
	if err := s.browserCtx.Err(); err != nil {
		return nil, fmt.Errorf("cdp: new page: browser context dead: %w", err)
	}
	p := &Page{session: s, ctx: s.browserCtx, cancel: func() {}}
	if err := enableNetworkForPage(p); err != nil {
		return nil, err
	}
	// Drop cookies so a prior login cannot skip the login form on the next test.
	_ = chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return network.ClearBrowserCookies().Do(ctx)
	}))
	// Reset document between tests.
	_ = chromedp.Run(p.ctx, chromedp.Navigate("about:blank"))
	return p, nil
}

// ClearOriginStorage clears cookies/local/session storage for originURL's origin.
// Call between tests when the same browser context is reused (workers=1).
func (p *Page) ClearOriginStorage(originURL string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	originURL = strings.TrimSpace(originURL)
	if originURL == "" {
		return fmt.Errorf("cdp: empty origin URL")
	}
	u, err := url.Parse(originURL)
	if err != nil {
		return fmt.Errorf("cdp: parse origin URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("cdp: origin URL missing scheme/host")
	}
	origin := u.Scheme + "://" + u.Host
	return chromedp.Run(p.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.ClearBrowserCookies().Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// CDP ClearDataForOrigin has no session_storage type; clear that via JS below.
			return storage.ClearDataForOrigin(origin, "cookies,local_storage,indexeddb,cache_storage").Do(ctx)
		}),
		chromedp.Evaluate(`(() => {
  try { sessionStorage.clear(); } catch (e) {}
  try { localStorage.clear(); } catch (e) {}
  return true;
})()`, nil),
	)
}

// ScreenshotCurrent writes a PNG of the session's current tab without navigating away.
func (s *Session) ScreenshotCurrent(path string) error {
	if s == nil || s.browserCtx == nil {
		return fmt.Errorf("cdp: nil session")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("cdp: empty screenshot path")
	}
	if err := s.browserCtx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf []byte
	if err := chromedp.Run(s.browserCtx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// Close closes the tab.
func (p *Page) Close() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
}

// Context returns the tab context.
func (p *Page) Context() context.Context {
	if p == nil {
		return nil
	}
	return p.ctx
}

// Goto navigates to url. waitUntil is "load" (default) or "domcontentloaded".
func (p *Page) Goto(url string, waitUntil string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	actions, err := waitUntilActions(waitUntil, chromedp.Navigate(url))
	if err != nil {
		return err
	}
	return chromedp.Run(p.ctx, actions...)
}

// Reload reloads the current document. waitUntil matches Goto.
func (p *Page) Reload(waitUntil string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	actions, err := waitUntilActions(waitUntil, chromedp.Reload())
	if err != nil {
		return err
	}
	return chromedp.Run(p.ctx, actions...)
}

func waitUntilActions(waitUntil string, nav chromedp.Action) ([]chromedp.Action, error) {
	waitUntil = strings.ToLower(strings.TrimSpace(waitUntil))
	if waitUntil == "" {
		waitUntil = "load"
	}
	actions := []chromedp.Action{nav}
	switch waitUntil {
	case "domcontentloaded":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
	case "load", "networkidle":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
		// Navigate/Reload already wait for the load event by default.
	default:
		return nil, fmt.Errorf("cdp: unsupported waitUntil %q", waitUntil)
	}
	return actions, nil
}

// Click clicks the first element matching css (DOM click for Vue handlers).
func (p *Page) Click(css string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	// Pass selector via JSON literal (not single-quoted) to avoid quote-break issues.
	// Do not WaitVisible here: Element Plus popper/dropdown nodes are often
	// "invisible" to chromedp while still clickable; callers use expect.toBeVisible.
	script := fmt.Sprintf(`(() => {
  const css = %s;
  const el = document.querySelector(css);
  if (!el) throw new Error('click: no element for ' + css);
  el.click();
  return true;
})()`, jsonQuote(css))
	return chromedp.Run(p.ctx, chromedp.Evaluate(script, nil))
}

// Fill focuses and sets text so Vue v-model / Element Plus pick up the value.
func (p *Page) Fill(css string, text string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	script := fmt.Sprintf(`(() => {
  const css = %s;
  const text = %s;
  const el = document.querySelector(css);
  if (!el) throw new Error('fill: no element for ' + css);
  el.focus();
  const proto = el instanceof HTMLTextAreaElement
    ? Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value')
    : (el instanceof HTMLInputElement
        ? Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')
        : null);
  if (proto && proto.set) proto.set.call(el, text);
  else el.value = text;
  el.dispatchEvent(new InputEvent('input', { bubbles: true, composed: true, data: text }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
  return true;
})()`, jsonQuote(css), jsonQuote(text))
	return chromedp.Run(p.ctx, chromedp.Evaluate(script, nil))
}

// Evaluate runs js in the page and returns a JSON string of the result.
func (p *Page) Evaluate(js string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("cdp: nil page")
	}
	// Await then stringify so Promise-returning expressions serialize the resolved value.
	expr := fmt.Sprintf(`Promise.resolve((function(){ return (%s); })()).then(v => JSON.stringify(v === undefined ? null : v))`, js)
	var out string
	if err := chromedp.Run(p.ctx, chromedp.Evaluate(expr, &out, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	})); err != nil {
		return "", err
	}
	return normalizeEvaluateOut(out), nil
}

func normalizeEvaluateOut(out string) string {
	if out == "" {
		return emptyEvaluateJSON
	}
	return out
}

// Screenshot writes a PNG to path.
func (p *Page) Screenshot(path string) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("cdp: empty screenshot path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf []byte
	if err := chromedp.Run(p.ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

// WaitForFunction polls jsExpr until it is truthy or timeout.
func (p *Page) WaitForFunction(jsExpr string, timeout time.Duration) error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.Now().Add(timeout)
	// Evaluate expression or invoke function-source strings; swallow transient DOM errors.
	wrapped := fmt.Sprintf(`(() => { try {
  const __v = (%s);
  return Boolean(typeof __v === 'function' ? __v() : __v);
} catch (e) { return false; } })()`, jsExpr)
	for {
		var ok bool
		if err := pageEvaluate(p.ctx, wrapped, &ok); err != nil {
			if p.ctx.Err() != nil {
				return p.ctx.Err()
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("cdp: waitForFunction: %w", err)
			}
		} else if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("cdp: waitForFunction timeout after %s", timeout)
		}
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// QueryCount returns document.querySelectorAll(css).length.
func (p *Page) QueryCount(css string) (int, error) {
	if p == nil {
		return 0, fmt.Errorf("cdp: nil page")
	}
	var n int
	js := fmt.Sprintf(`document.querySelectorAll(%s).length`, jsonQuote(css))
	if err := chromedp.Run(p.ctx, chromedp.Evaluate(js, &n)); err != nil {
		return 0, err
	}
	return n, nil
}

// IsVisible reports whether the first matching element is visible.
func (p *Page) IsVisible(css string) (bool, error) {
	if p == nil {
		return false, fmt.Errorf("cdp: nil page")
	}
	js := fmt.Sprintf(`(() => {
  const el = document.querySelector(%s);
  if (!el) return false;
  const style = window.getComputedStyle(el);
  if (!style || style.visibility === 'hidden' || style.display === 'none' || style.opacity === '0') return false;
  const r = el.getBoundingClientRect();
  return r.width > 0 && r.height > 0;
})()`, jsonQuote(css))
	var ok bool
	if err := chromedp.Run(p.ctx, chromedp.Evaluate(js, &ok)); err != nil {
		return false, err
	}
	return ok, nil
}

// IsEnabled reports whether the first matching element is enabled.
func (p *Page) IsEnabled(css string) (bool, error) {
	if p == nil {
		return false, fmt.Errorf("cdp: nil page")
	}
	js := fmt.Sprintf(`(() => {
  const el = document.querySelector(%s);
  if (!el) return false;
  if (el.disabled) return false;
  if (el.getAttribute && el.getAttribute('aria-disabled') === 'true') return false;
  return true;
})()`, jsonQuote(css))
	var ok bool
	if err := chromedp.Run(p.ctx, chromedp.Evaluate(js, &ok)); err != nil {
		return false, err
	}
	return ok, nil
}

// URL returns location.href.
func (p *Page) URL() (string, error) {
	if p == nil {
		return "", fmt.Errorf("cdp: nil page")
	}
	var href string
	if err := chromedp.Run(p.ctx, chromedp.Location(&href)); err != nil {
		return "", err
	}
	return href, nil
}

// EnsureDOM is a no-op helper used by tests.
func (p *Page) EnsureDOM() error {
	if p == nil {
		return fmt.Errorf("cdp: nil page")
	}
	return chromedp.Run(p.ctx, dom.Enable())
}

func jsonQuote(s string) string {
	b, err := jsonMarshalString(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// jsonMarshalString is json.Marshal for strings; tests may override.
var jsonMarshalString = func(s string) ([]byte, error) { return json.Marshal(s) }
