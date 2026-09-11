// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestInstallFetchRouteBindingsErrorPaths(t *testing.T) {
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
	}()
	host, err = Install(engine, nil, "{}")
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	for _, call := range []string{
		`await globalThis.__choysum_e2e_host__.enableFetch()`,
		`await globalThis.__choysum_e2e_host__.disableFetch()`,
		`await globalThis.__choysum_e2e_host__.waitPausedRequest(10)`,
		`await globalThis.__choysum_e2e_host__.fulfillRequest('x', '{}')`,
		`await globalThis.__choysum_e2e_host__.continueRequest('x')`,
		`await globalThis.__choysum_e2e_host__.failRequest('x')`,
	} {
		raw := awaitHostErr(t, qjs, call)
		if !strings.Contains(raw, "no active page") {
			t.Fatalf("%s => %s", call, raw)
		}
	}

	raw := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fulfillRequest('x', '{bad')`)
	if !strings.Contains(raw, "fulfillRequest opts") {
		t.Fatalf("bad fulfill json: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fulfillRequest('x', JSON.stringify({bodyBase64:'!!!'}))`)
	if !strings.Contains(raw, "bodyBase64") {
		t.Fatalf("bad bodyBase64: %s", raw)
	}
	// undefined/null opts must fall back to "{}" (not the strings "undefined"/"null").
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fulfillRequest('x', undefined)`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("undefined opts should not unmarshal-fail: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fulfillRequest('x', null)`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("null opts should not unmarshal-fail: %s", raw)
	}
}

func TestInstallFetchRouteBindingsWithChrome(t *testing.T) {
	session := startPagehostChrome(t)
	defer session.Close()

	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
	}()
	if !engine.(*quickjsengine.QuickjsEngine).Ctx.BootstrapTimers() {
		t.Fatal("BootstrapTimers failed")
	}
	host, err = Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "live-body")
	}))
	defer srv.Close()

	raw := awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.enableFetch();
await h.enableFetch(); // idempotent
let navDone = false;
const nav = h.goto(`+jsonQuote(srv.URL+"/mock")+`, 'domcontentloaded').then(
  (v) => { navDone = true; return v; },
  (e) => { navDone = true; throw e; },
);
let mocked = false;
for (let i = 0; i < 40; i++) {
  const pausedRaw = await h.waitPausedRequest(500);
  if (pausedRaw == null) {
    if (navDone && mocked) break;
    continue;
  }
  const paused = JSON.parse(pausedRaw);
  if (String(paused.url).includes('/mock')) {
    await h.fulfillRequest(paused.id, JSON.stringify({
      status: 200,
      contentType: 'text/plain',
      body: 'route-mocked',
      headers: { 'X-E2E': '1', 'Content-Type': 'ignored' },
    }));
    mocked = true;
  } else {
    await h.continueRequest(paused.id);
  }
}
await nav;
const body = JSON.parse(await h.evaluate("document.body ? document.body.innerText : ''"));
const timedOut = await h.waitPausedRequest(20);
await h.disableFetch();
const afterDisable = await h.waitPausedRequest(50);
return {mocked, body, timedOut, afterDisable};
`)
	var got struct {
		Mocked       bool        `json:"mocked"`
		Body         string      `json:"body"`
		TimedOut     interface{} `json:"timedOut"`
		AfterDisable interface{} `json:"afterDisable"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if !got.Mocked || !strings.Contains(got.Body, "route-mocked") {
		t.Fatalf("got %+v", got)
	}
	if got.TimedOut != nil || got.AfterDisable != nil {
		t.Fatalf("expected null waits, got %+v", got)
	}

	// Continue path + bodyBase64 + disable while waiting.
	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.enableFetch();
let navDone = false;
let fulfilled = false;
const nav = h.goto(`+jsonQuote(srv.URL+"/live")+`, 'domcontentloaded').then(
  (v) => { navDone = true; return v; },
  (e) => { navDone = true; throw e; },
);
for (let i = 0; i < 40; i++) {
  const pausedRaw = await h.waitPausedRequest(500);
  if (pausedRaw == null) {
    if (navDone && fulfilled) break;
    continue;
  }
  const paused = JSON.parse(pausedRaw);
  if (String(paused.url).includes('/live')) {
    const b64 = 'bGl2ZS1va2F5'; // live-okay
    await h.fulfillRequest(paused.id, JSON.stringify({
      status: 200,
      contentType: 'text/plain',
      bodyBase64: b64,
    }));
    fulfilled = true;
  } else {
    await h.continueRequest(paused.id);
  }
}
await nav;
const body = JSON.parse(await h.evaluate("document.body ? document.body.innerText : ''"));
const waitP = h.waitPausedRequest(2000);
await h.disableFetch();
const disabled = await waitP;
return {body, disabled};
`)
	var got2 struct {
		Body     string      `json:"body"`
		Disabled interface{} `json:"disabled"`
	}
	if err := json.Unmarshal([]byte(raw), &got2); err != nil {
		t.Fatalf("parse2 %s: %v", raw, err)
	}
	if !strings.Contains(got2.Body, "live-okay") {
		t.Fatalf("bodyBase64 fulfill: %+v", got2)
	}
	if got2.Disabled != nil {
		t.Fatalf("disable should resolve wait as null: %+v", got2)
	}

	// continue-only navigation + fulfill/continue unknown id + marshal error on paused.
	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.enableFetch();
let navDone = false;
const nav = h.goto(`+jsonQuote(srv.URL+"/passthrough")+`, 'domcontentloaded').then(
  (v) => { navDone = true; return v; },
  (e) => { navDone = true; throw e; },
);
for (let i = 0; i < 40; i++) {
  const pausedRaw = await h.waitPausedRequest(500);
  if (pausedRaw == null) {
    if (navDone) break;
    continue;
  }
  const paused = JSON.parse(pausedRaw);
  await h.continueRequest(paused.id);
}
await nav;
const body = JSON.parse(await h.evaluate("document.body ? document.body.innerText : ''"));
await h.disableFetch();
return body;
`)
	if !strings.Contains(raw, "live-body") {
		t.Fatalf("continue passthrough: %s", raw)
	}

	raw = awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.enableFetch();
await h.fulfillRequest('missing-id', '{}');
`)
	if !strings.Contains(raw, "unknown fetch request id") {
		t.Fatalf("fulfill missing: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.continueRequest('missing-id');
`)
	if !strings.Contains(raw, "unknown fetch request id") {
		t.Fatalf("continue missing: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.failRequest('missing-id');
`)
	if !strings.Contains(raw, "unknown fetch request id") {
		t.Fatalf("fail missing: %s", raw)
	}

	// Abort/fail a paused request (network failure, not HTTP 500).
	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.enableFetch();
const nav = h.goto(`+jsonQuote(srv.URL+"/abort-me")+`, 'domcontentloaded');
let aborted = false;
for (let i = 0; i < 40; i++) {
  const pausedRaw = await h.waitPausedRequest(500);
  if (pausedRaw == null) {
    if (aborted) break;
    continue;
  }
  const paused = JSON.parse(pausedRaw);
  if (String(paused.url).includes('/abort-me')) {
    await h.failRequest(paused.id);
    aborted = true;
  } else {
    await h.continueRequest(paused.id);
  }
}
let navErr = '';
try { await nav; } catch (e) { navErr = String(e && e.message ? e.message : e); }
await h.disableFetch();
return {aborted, navErr};
`)
	if !strings.Contains(raw, `"aborted":true`) {
		t.Fatalf("failRequest abort: %s", raw)
	}

	oldMarshal := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("paused marshal boom") }
	raw = awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.enableFetch();
const nav = h.goto(`+jsonQuote(srv.URL+"/marshal")+`, 'domcontentloaded');
const pausedRaw = await h.waitPausedRequest(5000);
await h.disableFetch();
try { await nav; } catch (_) {}
return pausedRaw;
`)
	jsonMarshal = oldMarshal
	if !strings.Contains(raw, "paused marshal boom") {
		t.Fatalf("marshal paused: %s", raw)
	}
}

func TestWaitPausedMapsToNull(t *testing.T) {
	t.Parallel()
	for _, msg := range []string{
		"cdp: wait paused request: timeout",
		"cdp: fetch disabled",
		"cdp: fetch not enabled",
		"timeout",
	} {
		if !waitPausedMapsToNull(msg) {
			t.Fatalf("expected null mapping for %q", msg)
		}
	}
	if waitPausedMapsToNull("context canceled") {
		t.Fatal("unexpected null mapping")
	}
}

func TestInstallWaitPausedNullMappingBranches(t *testing.T) {
	// Chrome-free: cover every waitPausedRequest null-mapping operand and arg defaulting.
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
		pageEnableFetch, pageDisableFetch, pageWaitPaused = pageEnableFetchDefault, pageDisableFetchDefault, pageWaitPausedDefault
	}()
	if !engine.(*quickjsengine.QuickjsEngine).Ctx.BootstrapTimers() {
		t.Fatal("BootstrapTimers failed")
	}
	host, err = Install(engine, nil, "{}")
	if err != nil {
		t.Fatal(err)
	}
	// activePage only checks non-nil page; hooks avoid real CDP calls.
	host.page = &cdp.Page{}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	pageEnableFetch = func(p *cdp.Page) error { return nil }
	pageDisableFetch = func(p *cdp.Page) error { return nil }
	if raw := awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.enableFetch(); return 'ok'`); !strings.Contains(raw, "ok") {
		t.Fatalf("enable ok: %s", raw)
	}
	if raw := awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.disableFetch(); return 'ok'`); !strings.Contains(raw, "ok") {
		t.Fatalf("disable ok: %s", raw)
	}

	for _, msg := range []string{
		"cdp: wait paused request: timeout",
		"cdp: fetch disabled",
		"cdp: fetch not enabled",
	} {
		msg := msg
		pageWaitPaused = func(p *cdp.Page, timeout time.Duration) (*cdp.PausedRequest, error) {
			return nil, errors.New(msg)
		}
		// Default timeout branch (no args) and non-number arg branch.
		for _, call := range []string{
			`return await globalThis.__choysum_e2e_host__.waitPausedRequest()`,
			`return await globalThis.__choysum_e2e_host__.waitPausedRequest('nope')`,
			`return await globalThis.__choysum_e2e_host__.waitPausedRequest(25)`,
		} {
			raw := awaitHost(t, qjs, call)
			if raw != "null" {
				t.Fatalf("msg=%q call=%q => %s", msg, call, raw)
			}
		}
	}

	pageWaitPaused = func(p *cdp.Page, timeout time.Duration) (*cdp.PausedRequest, error) {
		return &cdp.PausedRequest{ID: "fetch-1", URL: "http://example.test/", Method: "GET"}, nil
	}
	raw := awaitHost(t, qjs, `return await globalThis.__choysum_e2e_host__.waitPausedRequest(10)`)
	if !strings.Contains(raw, "fetch-1") {
		t.Fatalf("paused json: %s", raw)
	}

	// fulfillRequest with only the id arg (opts default "{}").
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fulfillRequest('missing-only-id')`)
	if !strings.Contains(raw, "unknown fetch request id") && !strings.Contains(raw, "empty fetch") && !strings.Contains(raw, "cdp:") {
		t.Fatalf("single-arg fulfill: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.failRequest('missing-only-id')`)
	if !strings.Contains(raw, "unknown fetch request id") && !strings.Contains(raw, "cdp:") {
		t.Fatalf("failRequest missing: %s", raw)
	}
}

func TestInstallFetchRouteBindingHookErrorsAndClosed(t *testing.T) {
	session := startPagehostChrome(t)
	defer session.Close()

	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
	}()
	if !engine.(*quickjsengine.QuickjsEngine).Ctx.BootstrapTimers() {
		t.Fatal("BootstrapTimers failed")
	}
	host, err = Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	_ = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.newPage(); return 'ok'`)

	oldEnable, oldDisable, oldWait := pageEnableFetch, pageDisableFetch, pageWaitPaused
	defer func() {
		pageEnableFetch, pageDisableFetch, pageWaitPaused = oldEnable, oldDisable, oldWait
	}()

	pageEnableFetch = func(p *cdp.Page) error { return errors.New("enable boom") }
	raw := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.enableFetch()`)
	if !strings.Contains(raw, "enable boom") {
		t.Fatalf("enable err: %s", raw)
	}
	pageEnableFetch = oldEnable

	pageDisableFetch = func(p *cdp.Page) error { return errors.New("disable boom") }
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.disableFetch()`)
	if !strings.Contains(raw, "disable boom") {
		t.Fatalf("disable err: %s", raw)
	}
	pageDisableFetch = oldDisable

	pageWaitPaused = func(p *cdp.Page, timeout time.Duration) (*cdp.PausedRequest, error) {
		return nil, errors.New("wait boom")
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.waitPausedRequest(10)`)
	if !strings.Contains(raw, "wait boom") {
		t.Fatalf("wait reject: %s", raw)
	}

	// Closed before schedule after wait returns.
	pageWaitPaused = func(p *cdp.Page, timeout time.Duration) (*cdp.PausedRequest, error) {
		time.Sleep(30 * time.Millisecond)
		return nil, errors.New("timeout")
	}
	_ = awaitHost(t, qjs, `
globalThis.__choysum_e2e_host__.waitPausedRequest(50);
return 'armed';
`)
	host.Drain()
	time.Sleep(80 * time.Millisecond)

	// Re-install after Drain so globals point at a live host for the closed-in-job path.
	host, err = Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	_ = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.newPage(); return 'ok'`)

	oldSched := ctxSchedule
	defer func() { ctxSchedule = oldSched }()
	pageWaitPaused = func(p *cdp.Page, timeout time.Duration) (*cdp.PausedRequest, error) {
		return nil, errors.New("timeout")
	}
	// Run the schedule job synchronously with closed=true so waitPausedRequest's
	// closed-at-callback early return is covered (async Schedule may not pump in time).
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		host.closed.Store(true)
		job(ctx)
		return true
	}
	// Do not await waitPausedRequest: closed job skips resolve/reject.
	// Avoid other host async APIs here — ctxSchedule is wrapped for this case only.
	_ = awaitHost(t, qjs, `
globalThis.__choysum_e2e_host__.waitPausedRequest(10);
return 'armed';
`)
	time.Sleep(80 * time.Millisecond)
	host.closed.Store(false)
}

func TestInstallAsyncGotoReloadClosedInScheduleJob(t *testing.T) {
	session := startPagehostChrome(t)
	defer session.Close()

	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
	}()
	if !engine.(*quickjsengine.QuickjsEngine).Ctx.BootstrapTimers() {
		t.Fatal("BootstrapTimers failed")
	}
	host, err = Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<!doctype html><html><body>ok</body></html>`)
	}))
	defer srv.Close()

	_ = awaitHost(t, qjs, `
await globalThis.__choysum_e2e_host__.newPage();
await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'domcontentloaded');
return 'ready';
`)

	oldSched := ctxSchedule
	defer func() {
		ctxSchedule = oldSched
		host.closed.Store(false)
	}()
	// Run settle jobs synchronously; mark closed first to hit the in-job guard.
	armClosedSettle := func() {
		ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
			host.closed.Store(true)
			job(ctx)
			return true
		}
	}
	waitClosed := func(label string) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if host.closed.Load() {
				host.closed.Store(false)
				ctxSchedule = oldSched
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("expected %s settle to mark host closed", label)
	}

	armClosedSettle()
	_ = awaitHost(t, qjs, `
globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'domcontentloaded');
return 'armed-goto';
`)
	waitClosed("goto")

	armClosedSettle()
	_ = awaitHost(t, qjs, `
globalThis.__choysum_e2e_host__.reload('domcontentloaded');
return 'armed-reload';
`)
	waitClosed("reload")
}

func TestInstallAsyncGotoReloadErrors(t *testing.T) {
	session := startPagehostChrome(t)
	defer session.Close()

	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
	}()
	if !engine.(*quickjsengine.QuickjsEngine).Ctx.BootstrapTimers() {
		t.Fatal("BootstrapTimers failed")
	}
	host, err = Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	raw := awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto('http://127.0.0.1:1/', 'load');
`)
	if raw == "" || strings.Contains(raw, `"ok":true`) {
		t.Fatalf("expected goto error, got %s", raw)
	}

	raw = awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.reload('not-a-wait');
`)
	if !strings.Contains(raw, "unsupported waitUntil") {
		t.Fatalf("reload bad waitUntil: %s", raw)
	}

	// Successful async reload (covers resolve path).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<!doctype html><html><body>ok</body></html>`)
	}))
	defer srv.Close()
	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto(`+jsonQuote(srv.URL)+`, 'domcontentloaded');
await h.reload('domcontentloaded');
return 'reloaded';
`)
	if !strings.Contains(raw, "reloaded") {
		t.Fatalf("reload ok: %s", raw)
	}

	// Drain while async goto/reload pending (closed host schedule path).
	_ = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
h.goto('about:blank', 'domcontentloaded');
h.reload('domcontentloaded');
return 'scheduled';
`)
	host.Drain()
	time.Sleep(50 * time.Millisecond)

	// Closed mid-flight after a slow navigation completes scheduling.
	host2, err := Install(engine, session, "{}")
	if err != nil {
		t.Fatal(err)
	}
	defer host2.Drain()
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<!doctype html><html><body>slow</body></html>`)
	}))
	defer slow.Close()
	_ = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
h.goto(`+jsonQuote(slow.URL)+`, 'load');
return 'armed';
`)
	time.Sleep(20 * time.Millisecond)
	host2.Drain()
	time.Sleep(150 * time.Millisecond)
}
