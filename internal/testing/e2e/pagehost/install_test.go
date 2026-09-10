// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestInstallRegistersGlobalsWithoutBrowser(t *testing.T) {
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
		_ = engine.Close()
	}()

	host, err = Install(engine, nil, `{"baseURL":"http://127.0.0.1:9"}`)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	qjs := engine.(*quickjsengine.QuickjsEngine)
	val := qjs.Ctx.Eval(`(() => {
  const r = globalThis.__choysum_e2e_runtime__;
  const h = globalThis.__choysum_e2e_host__;
  if (!r || r.baseURL !== 'http://127.0.0.1:9') throw new Error('runtime missing');
  if (!h || typeof h.newPage !== 'function') throw new Error('host.newPage missing');
  if (typeof h.goto !== 'function') throw new Error('host.goto missing');
  if (typeof h.reload !== 'function') throw new Error('host.reload missing');
  if (typeof h.readTextFile !== 'function') throw new Error('host.readTextFile missing');
  if (typeof h.fetch !== 'function') throw new Error('host.fetch missing');
  if (typeof h.clearOriginStorage !== 'function') throw new Error('host.clearOriginStorage missing');
  if (typeof h.waitForResponse !== 'function') throw new Error('host.waitForResponse missing');
  if (typeof h.delay !== 'function') throw new Error('host.delay missing');
  return 'ok';
})()`)
	defer val.Free()
	if val.IsException() {
		t.Fatalf("eval: %v", qjs.Ctx.Exception())
	}
	if got := strings.TrimSpace(val.String()); got != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestInstallRequiresQuickjsEngine(t *testing.T) {
	_, err := Install(nil, nil, "{}")
	if err == nil || !strings.Contains(err.Error(), "QuickjsEngine") {
		t.Fatalf("expected QuickjsEngine error, got %v", err)
	}
}

func TestInstallEmptyAndWhitespaceRuntimeJSON(t *testing.T) {
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
		_ = engine.Close()
	}()
	host, err = Install(engine, nil, "   \n\t")
	if err != nil {
		t.Fatalf("Install whitespace: %v", err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	val := qjs.Ctx.Eval(`JSON.stringify(globalThis.__choysum_e2e_runtime__)`)
	defer val.Free()
	if val.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	if got := strings.TrimSpace(val.String()); got != "{}" {
		t.Fatalf("runtime=%q", got)
	}
}

func TestInstallInvalidRuntimeJSON(t *testing.T) {
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if host != nil {
			host.Drain()
		}
		_ = engine.Close()
	}()
	host, err = Install(engine, nil, "{not-json")
	if err == nil || !strings.Contains(err.Error(), "parse runtime json") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func awaitHost(t *testing.T, qjs *quickjsengine.QuickjsEngine, expr string) string {
	t.Helper()
	script := `(async () => { return JSON.stringify(await (async () => { ` + expr + ` })()); })()`
	val := qjs.Ctx.Eval(script)
	if val.IsException() {
		t.Fatalf("eval: %v", qjs.Ctx.Exception())
	}
	val = qjs.Ctx.Await(val)
	defer val.Free()
	if val.IsException() {
		t.Fatalf("await: %v", qjs.Ctx.Exception())
	}
	return strings.TrimSpace(val.String())
}

func awaitHostErr(t *testing.T, qjs *quickjsengine.QuickjsEngine, expr string) string {
	t.Helper()
	script := `(async () => {
  try {
    await (async () => { ` + expr + ` })();
    return JSON.stringify({ok:true});
  } catch (e) {
    return JSON.stringify({ok:false, message: String(e && e.message ? e.message : e)});
  }
})()`
	val := qjs.Ctx.Eval(script)
	if val.IsException() {
		t.Fatalf("eval: %v", qjs.Ctx.Exception())
	}
	val = qjs.Ctx.Await(val)
	defer val.Free()
	if val.IsException() {
		t.Fatalf("await: %v", qjs.Ctx.Exception())
	}
	return strings.TrimSpace(val.String())
}

func TestInstallHostMethodsErrorPathsWithoutPage(t *testing.T) {
	var host *Host
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	// Do not engine.Close(): async delay/waitForResponse may leave QuickJS promise
	// objects that abort JS_FreeRuntime (same constraint as runner_host).
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

	raw := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.newPage()`)
	if !strings.Contains(raw, "nil session") {
		t.Fatalf("newPage nil session: %s", raw)
	}

	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.goto('about:blank')`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("goto no page: %s", raw)
	}
	for _, call := range []string{
		`await globalThis.__choysum_e2e_host__.click('#x')`,
		`await globalThis.__choysum_e2e_host__.fill('#x','y')`,
		`await globalThis.__choysum_e2e_host__.evaluate('1')`,
		`await globalThis.__choysum_e2e_host__.waitForFunction('true', 10)`,
		`await globalThis.__choysum_e2e_host__.isVisible('#x')`,
		`await globalThis.__choysum_e2e_host__.isEnabled('#x')`,
		`await globalThis.__choysum_e2e_host__.count('#x')`,
		`await globalThis.__choysum_e2e_host__.url()`,
		`await globalThis.__choysum_e2e_host__.screenshot('/tmp/x.png')`,
		`await globalThis.__choysum_e2e_host__.reload('load')`,
	} {
		raw = awaitHostErr(t, qjs, call)
		if !strings.Contains(raw, "no active page") {
			t.Fatalf("%s => %s", call, raw)
		}
	}

	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.waitForResponse('{bad')`)
	if !strings.Contains(raw, "waitForResponse match") {
		t.Fatalf("bad match json: %s", raw)
	}

	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.delay(5); return 'delayed'`)
	if raw != `"delayed"` {
		t.Fatalf("delay: %s", raw)
	}
	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.delay(-1); return 'ok'`)
	if raw != `"ok"` {
		t.Fatalf("delay neg: %s", raw)
	}
	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.closePage(); return 'closed'`)
	if raw != `"closed"` {
		t.Fatalf("closePage: %s", raw)
	}
}

func TestHostDrainNilSafe(t *testing.T) {
	var host *Host
	host.Drain()
}

func TestScheduleEarlyReturnAndClosed(t *testing.T) {
	var host *Host
	if host.schedule(nil, nil) {
		t.Fatal("nil host should not schedule")
	}
	host = &Host{}
	if host.schedule(nil, func(ctx *quickjs.Context) {}) {
		t.Fatal("nil ctx should not schedule")
	}
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	if host.schedule(qjs.Ctx, nil) {
		t.Fatal("nil job should not schedule")
	}
	host.closed.Store(true)
	if host.schedule(qjs.Ctx, func(ctx *quickjs.Context) {}) {
		t.Fatal("closed host should not schedule")
	}
}

func TestScheduleTimeoutPaths(t *testing.T) {
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	host := &Host{}

	oldSched := ctxSchedule
	oldT1, oldT2 := scheduleSelectTimeout, scheduleSelectTimeout2
	defer func() {
		ctxSchedule = oldSched
		scheduleSelectTimeout = oldT1
		scheduleSelectTimeout2 = oldT2
	}()

	scheduleSelectTimeout = 30 * time.Millisecond
	scheduleSelectTimeout2 = 100 * time.Millisecond

	// Block Schedule past both timeouts → return false.
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		time.Sleep(200 * time.Millisecond)
		return true
	}
	if host.schedule(qjs.Ctx, func(ctx *quickjs.Context) {}) {
		t.Fatal("expected false after full timeout")
	}

	// First timeout fires while closed → return false without waiting second window.
	host.closed.Store(false)
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		time.Sleep(150 * time.Millisecond)
		return true
	}
	go func() {
		time.Sleep(5 * time.Millisecond)
		host.closed.Store(true)
	}()
	if host.schedule(qjs.Ctx, func(ctx *quickjs.Context) {}) {
		t.Fatal("expected false when closed during first wait")
	}

	// First timeout, not closed, then Schedule completes in second window.
	host.closed.Store(false)
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		time.Sleep(50 * time.Millisecond)
		return true
	}
	if !host.schedule(qjs.Ctx, func(ctx *quickjs.Context) {}) {
		t.Fatal("expected true when schedule finishes in second window")
	}
}

func TestInstallWaitForResponseTimeoutAndClosedDelay(t *testing.T) {
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
		_, _ = w.Write([]byte(`<!doctype html><html><body>ok</body></html>`))
	}))
	defer srv.Close()

	_ = awaitHost(t, qjs, `
await globalThis.__choysum_e2e_host__.newPage();
await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'load');
return 'ready';
`)
	// Timeout → reject(waitErr) via schedule callback.
	raw := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.waitForResponse(JSON.stringify({urlIncludes:'/never'}), 80)`)
	if !strings.Contains(raw, `"ok":false`) {
		t.Fatalf("waitForResponse timeout: %s", raw)
	}

	// Mark closed inside schedule and run the job synchronously so the
	// closed-at-callback branches execute without needing a QJS job pump.
	oldSched := ctxSchedule
	defer func() { ctxSchedule = oldSched }()
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		host.closed.Store(true)
		job(ctx)
		return true
	}
	val := qjs.Ctx.Eval(`globalThis.__choysum_e2e_host__.delay(5)`)
	if val.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	val.Free()
	time.Sleep(30 * time.Millisecond)
	// Also exercise waitForResponse post-wait closed-in-callback (short timeout).
	host.closed.Store(false)
	val = qjs.Ctx.Eval(`globalThis.__choysum_e2e_host__.waitForResponse(JSON.stringify({urlIncludes:'/never2'}), 40)`)
	if val.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	val.Free()
	time.Sleep(200 * time.Millisecond)
	host.Drain()
}

func TestInstallWaitForResponseScheduleFailAfterTimeout(t *testing.T) {
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
		_, _ = w.Write([]byte(`<!doctype html><html><body>ok</body></html>`))
	}))
	defer srv.Close()
	_ = awaitHost(t, qjs, `
await globalThis.__choysum_e2e_host__.newPage();
await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'load');
return 'ready';
`)

	oldSched := ctxSchedule
	oldT1, oldT2 := scheduleSelectTimeout, scheduleSelectTimeout2
	defer func() {
		ctxSchedule = oldSched
		scheduleSelectTimeout = oldT1
		scheduleSelectTimeout2 = oldT2
	}()
	scheduleSelectTimeout = 15 * time.Millisecond
	scheduleSelectTimeout2 = 15 * time.Millisecond
	// Block schedule so waitForResponse timeout cannot deliver reject via Schedule.
	ctxSchedule = func(ctx *quickjs.Context, job func(*quickjs.Context)) bool {
		time.Sleep(200 * time.Millisecond)
		return false
	}

	// Fire-and-forget: promise may never settle; Drain after short wait.
	val := qjs.Ctx.Eval(`globalThis.__choysum_e2e_host__.waitForResponse(JSON.stringify({urlIncludes:'/never'}), 50)`)
	if val.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	val.Free()
	time.Sleep(400 * time.Millisecond)
	host.Drain()
}

func TestInstallHostOpFailuresWithChrome(t *testing.T) {
	session := startPagehostChrome(t)
	t.Cleanup(session.Close)

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

	raw := awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.newPage(); return 'ok'`)
	if raw != `"ok"` {
		t.Fatalf("newPage: %s", raw)
	}
	// Replace existing page (closes prior tab).
	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.newPage(); return 'ok2'`)
	if raw != `"ok2"` {
		t.Fatalf("newPage replace: %s", raw)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><html><body><div id="x">hi</div></body></html>`))
	}))
	defer srv.Close()

	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'bogus')`)
	if !strings.Contains(raw, `"ok":false`) {
		t.Fatalf("goto bad waitUntil: %s", raw)
	}
	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'load'); return 'nav'`)
	if raw != `"nav"` {
		t.Fatalf("goto: %s", raw)
	}

	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.evaluate('throw new Error("x")')`)
	if !strings.Contains(raw, `"ok":false`) {
		t.Fatalf("evaluate throw: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.waitForFunction('false', 50)`)
	if !strings.Contains(raw, `"ok":false`) {
		t.Fatalf("waitForFunction timeout: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.screenshot('')`)
	if !strings.Contains(raw, `"ok":false`) {
		t.Fatalf("screenshot empty: %s", raw)
	}

	// Kill browser under the host so subsequent CDP ops fail quickly (avoid long WaitVisible).
	// Keep the active page pointer so click/fill/isVisible hit CDP errors (not "no active page").
	session.Close()
	for _, call := range []string{
		`await globalThis.__choysum_e2e_host__.click('#x')`,
		`await globalThis.__choysum_e2e_host__.fill('#x','y')`,
		`await globalThis.__choysum_e2e_host__.isVisible('#x')`,
		`await globalThis.__choysum_e2e_host__.isEnabled('#x')`,
		`await globalThis.__choysum_e2e_host__.count('#x')`,
		`await globalThis.__choysum_e2e_host__.url()`,
		`await globalThis.__choysum_e2e_host__.screenshot(` + jsonQuote(filepath.Join(t.TempDir(), "dead.png")) + `)`,
		`await globalThis.__choysum_e2e_host__.goto('about:blank','load')`,
		`await globalThis.__choysum_e2e_host__.newPage()`,
	} {
		raw = awaitHostErr(t, qjs, call)
		if !strings.Contains(raw, `"ok":false`) {
			t.Fatalf("%s => %s", call, raw)
		}
	}

	raw = awaitHost(t, qjs, `await globalThis.__choysum_e2e_host__.closePage(); return 'c'`)
	if raw != `"c"` {
		t.Fatalf("close: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.waitForResponse('{"urlIncludes":"/x"}', 100)`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("waitForResponse no page: %s", raw)
	}
	raw = awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.isVisible()`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("isVisible no args: %s", raw)
	}
}

func TestInstallDelayDrainBeforeResolve(t *testing.T) {
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
	// Fire a long delay then Drain immediately so the goroutine sees closed before resolve.
	val := qjs.Ctx.Eval(`globalThis.__choysum_e2e_host__.delay(5000)`)
	if val.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	val.Free()
	host.Drain()
}

func TestInstallWaitForResponseDrainWhilePending(t *testing.T) {
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
		_, _ = w.Write([]byte(`<!doctype html><html><body>ok</body></html>`))
	}))
	defer srv.Close()

	_ = awaitHost(t, qjs, `
await globalThis.__choysum_e2e_host__.newPage();
await globalThis.__choysum_e2e_host__.goto(`+jsonQuote(srv.URL)+`, 'load');
globalThis.__choysum_e2e_host__.waitForResponse(JSON.stringify({urlIncludes:'/never'}), 5000);
await globalThis.__choysum_e2e_host__.delay(30);
return 'armed';
`)
	host.Drain() // closed while waitForResponse / delay may still be pending
}

func startPagehostChrome(t *testing.T) *cdp.Session {
	t.Helper()
	cands := []string{}
	if p, err := cdp.ResolveChromiumPath(); err == nil {
		cands = append(cands, p)
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	headless := true
	var session *cdp.Session
	var lastErr error
	for _, path := range cands {
		session, lastErr = cdp.Start(nil, cdp.StartOptions{ExecPath: path, Headless: &headless})
		if lastErr == nil {
			t.Cleanup(session.Close)
			return session
		}
	}
	t.Skipf("chromium start failed: %v", lastErr)
	return nil
}

func TestInstallDriveHostWithChrome(t *testing.T) {
	cands := []string{}
	if p, err := cdp.ResolveChromiumPath(); err == nil {
		cands = append(cands, p)
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	headless := true
	var session *cdp.Session
	var lastErr error
	for _, path := range cands {
		session, lastErr = cdp.Start(nil, cdp.StartOptions{ExecPath: path, Headless: &headless})
		if lastErr == nil {
			break
		}
	}
	if session == nil {
		t.Skipf("chromium start failed: %v", lastErr)
	}
	defer session.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/echo":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"hi":1}`))
		default:
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<input id="name" />
<button id="btn">ok</button>
<button id="post">post</button>
<div class="n">a</div><div class="n">b</div>
<script>
document.getElementById('post').onclick = () => fetch('/api/echo', {method:'GET'});
</script></body></html>`))
		}
	}))
	defer srv.Close()

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
	runtimeJSON, _ := json.Marshal(map[string]string{"baseURL": srv.URL})
	host, err = Install(engine, session, string(runtimeJSON))
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	shot := filepath.Join(t.TempDir(), "host.png")
	script := `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto(` + jsonQuote(srv.URL+"/") + `, 'load');
await h.fill('#name', 'bob');
const v = JSON.parse(await h.evaluate("document.querySelector('#name').value"));
if (v !== 'bob') throw new Error('fill failed: ' + v);
await h.click('#btn');
await h.waitForFunction('true', 1000);
const visible = await h.isVisible('#btn');
const enabled = await h.isEnabled('#btn');
const n = await h.count('.n');
const u = await h.url();
await h.screenshot(` + jsonQuote(shot) + `);
const waitP = h.waitForResponse(JSON.stringify({urlIncludes:'/api/echo', method:'GET'}), 10000);
await h.delay(20);
await h.click('#post');
const respRaw = await waitP;
const resp = JSON.parse(respRaw);
await h.closePage();
await h.newPage();
await h.closePage();
return {visible, enabled, n, u, status: resp.status, hasBody: !!resp.bodyBase64};
`
	raw := awaitHost(t, qjs, script)
	var got struct {
		Visible bool   `json:"visible"`
		Enabled bool   `json:"enabled"`
		N       int    `json:"n"`
		U       string `json:"u"`
		Status  int64  `json:"status"`
		HasBody bool   `json:"hasBody"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if !got.Visible || !got.Enabled || got.N != 2 || !strings.HasPrefix(got.U, srv.URL) || got.Status != 200 {
		t.Fatalf("unexpected host result: %+v raw=%s", got, raw)
	}
	if st, err := os.Stat(shot); err != nil || st.Size() == 0 {
		t.Fatalf("screenshot: %v", err)
	}
}

func TestWaitForResponseMarshalError(t *testing.T) {
	session := startPagehostChrome(t)
	page, err := session.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/m":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":1}`))
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<button id="go">go</button>
<script>document.getElementById('go').onclick=()=>fetch('/api/m');</script>
</body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("marshal boom") }
	t.Cleanup(func() { jsonMarshal = old })

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
	runtimeJSON, _ := json.Marshal(map[string]string{"baseURL": srv.URL})
	host, err = Install(engine, session, string(runtimeJSON))
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	if err := page.Goto(srv.URL+"/", "load"); err != nil {
		// Host opens its own page; navigate via host.
		_ = err
	}
	raw := awaitHostErr(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto(`+jsonQuote(srv.URL+"/")+`, 'load');
const waitP = h.waitForResponse(JSON.stringify({urlIncludes:'/api/m'}), 10000);
await h.delay(30);
await h.click('#go');
await waitP;
`)
	if !strings.Contains(raw, "marshal boom") {
		t.Fatalf("expected marshal boom, got %s", raw)
	}
}

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func TestInstallFetchAndReadTextFile(t *testing.T) {
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Test") != "1" {
			http.Error(w, "header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Echo", string(body))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	raw := awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
const init = JSON.stringify({
  method: 'POST',
  headers: {'X-Test': '1', 'Content-Type': 'text/plain'},
  body: 'hello-body'
});
const respRaw = await h.fetch(`+jsonQuote(srv.URL)+`, init);
return JSON.parse(respRaw);
`)
	var got struct {
		Status     int               `json:"status"`
		Headers    map[string]string `json:"headers"`
		BodyBase64 string            `json:"bodyBase64"`
		URL        string            `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if got.Status != 200 {
		t.Fatalf("status=%d raw=%s", got.Status, raw)
	}
	if got.Headers["X-Echo"] != "hello-body" && got.Headers["x-echo"] != "hello-body" {
		// httptest canonicalizes header keys; Go map keeps canonical form.
		found := false
		for k, v := range got.Headers {
			if strings.EqualFold(k, "X-Echo") && v == "hello-body" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing X-Echo header: %#v", got.Headers)
		}
	}
	body, err := base64.StdEncoding.DecodeString(got.BodyBase64)
	if err != nil || string(body) != `{"ok":true}` {
		t.Fatalf("body=%q err=%v", body, err)
	}

	tmp := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(tmp, []byte("log-line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	textRaw := awaitHost(t, qjs, `return await globalThis.__choysum_e2e_host__.readTextFile(`+jsonQuote(tmp)+`)`)
	var text string
	if err := json.Unmarshal([]byte(textRaw), &text); err != nil {
		t.Fatalf("parse text %s: %v", textRaw, err)
	}
	if text != "log-line\n" {
		t.Fatalf("readTextFile=%q", text)
	}

	bad := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fetch('http://127.0.0.1:9', '{bad')`)
	if !strings.Contains(bad, "fetch init") {
		t.Fatalf("bad init: %s", bad)
	}
	missing := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.readTextFile(`+jsonQuote(filepath.Join(t.TempDir(), "missing.txt"))+`)`)
	if !strings.Contains(missing, `"ok":false`) {
		t.Fatalf("missing file: %s", missing)
	}
}

func TestInstallReloadWithChrome(t *testing.T) {
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

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body><div id="n">` + strconv.Itoa(hits) + `</div></body></html>`))
	}))
	defer srv.Close()

	raw := awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto(`+jsonQuote(srv.URL)+`, 'load');
const before = JSON.parse(await h.evaluate("document.querySelector('#n').textContent"));
await h.reload('domcontentloaded');
const after = JSON.parse(await h.evaluate("document.querySelector('#n').textContent"));
return {before: Number(before), after: Number(after)};
`)
	var got struct {
		Before int `json:"before"`
		After  int `json:"after"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if got.After <= got.Before {
		t.Fatalf("expected reload to increase hit counter: %+v hits=%d", got, hits)
	}
	if hits < 2 {
		t.Fatalf("expected at least 2 hits, got %d", hits)
	}
}

func TestInstallReloadErrorAndClearOriginStorage(t *testing.T) {
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
		_, _ = w.Write([]byte(`<!doctype html><html><body><div id="x">ok</div></body></html>`))
	}))
	defer srv.Close()

	raw := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.clearOriginStorage('http://127.0.0.1:9')`)
	if !strings.Contains(raw, "no active page") {
		t.Fatalf("clearOriginStorage no page: %s", raw)
	}

	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
await h.newPage();
await h.goto(`+jsonQuote(srv.URL)+`, 'load');
await h.clearOriginStorage(`+jsonQuote(srv.URL)+`);
await h.evaluate("(() => { localStorage.setItem('k','v'); sessionStorage.setItem('s','1'); return 'set'; })()");
await h.clearOriginStorage(`+jsonQuote(srv.URL+"/path")+`);
const cleared = JSON.parse(await h.evaluate("({ls: localStorage.getItem('k'), ss: sessionStorage.getItem('s')})"));
return cleared;
`)
	var cleared struct {
		LS *string `json:"ls"`
		SS *string `json:"ss"`
	}
	if err := json.Unmarshal([]byte(raw), &cleared); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if cleared.LS != nil || cleared.SS != nil {
		t.Fatalf("expected storage cleared, got %+v", cleared)
	}

	badReload := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.reload('bogus')`)
	if !strings.Contains(badReload, "unsupported waitUntil") {
		t.Fatalf("reload bad waitUntil: %s", badReload)
	}

	badOrigin := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.clearOriginStorage('')`)
	if !strings.Contains(badOrigin, "empty origin") {
		t.Fatalf("clearOriginStorage empty: %s", badOrigin)
	}
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (errReadCloser) Close() error             { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInstallFetchBranches(t *testing.T) {
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Method", r.Method)
		_, _ = w.Write([]byte("echo:" + string(body)))
	}))
	defer srv.Close()

	// Empty init JSON + default GET method.
	raw := awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
const respRaw = await h.fetch(`+jsonQuote(srv.URL)+`, '');
return JSON.parse(respRaw);
`)
	var got struct {
		Status     int    `json:"status"`
		BodyBase64 string `json:"bodyBase64"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	if got.Status != 200 {
		t.Fatalf("empty init GET status=%d", got.Status)
	}

	// Default method when omitted + bodyBase64 payload.
	b64 := base64.StdEncoding.EncodeToString([]byte("via-b64"))
	raw = awaitHost(t, qjs, `
const h = globalThis.__choysum_e2e_host__;
const init = JSON.stringify({headers: {'X-T': '1'}, bodyBase64: `+jsonQuote(b64)+`});
const respRaw = await h.fetch(`+jsonQuote(srv.URL)+`, init);
return JSON.parse(respRaw);
`)
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("parse b64 %s: %v", raw, err)
	}
	body, err := base64.StdEncoding.DecodeString(got.BodyBase64)
	if err != nil || !strings.Contains(string(body), "via-b64") {
		t.Fatalf("bodyBase64 echo=%q err=%v", body, err)
	}

	badB64 := awaitHostErr(t, qjs, `
await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, JSON.stringify({bodyBase64: '!!!'}))
`)
	if !strings.Contains(badB64, "bodyBase64") {
		t.Fatalf("bad bodyBase64: %s", badB64)
	}

	badURL := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fetch('', '{}')`)
	if !strings.Contains(badURL, `"ok":false`) {
		t.Fatalf("empty url: %s", badURL)
	}

	badMethod := awaitHostErr(t, qjs, `
await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, JSON.stringify({method: 'BAD METHOD'}))
`)
	if !strings.Contains(badMethod, `"ok":false`) {
		t.Fatalf("bad method: %s", badMethod)
	}

	oldClient := hostHTTPClient
	hostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("do boom")
		}),
	}
	doErr := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, '{}')`)
	hostHTTPClient = oldClient
	if !strings.Contains(doErr, "do boom") {
		t.Fatalf("Do error: %s", doErr)
	}

	hostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       errReadCloser{},
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}
	readErr := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, '{}')`)
	hostHTTPClient = oldClient
	if !strings.Contains(readErr, "read boom") {
		t.Fatalf("ReadAll error: %s", readErr)
	}

	hostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 204,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
				// nil Request exercises fallback URL path
			}, nil
		}),
	}
	oldMarshal := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("fetch marshal boom") }
	marshalErr := awaitHostErr(t, qjs, `await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, '{}')`)
	jsonMarshal = oldMarshal
	hostHTTPClient = oldClient
	if !strings.Contains(marshalErr, "fetch marshal boom") {
		t.Fatalf("marshal error: %s", marshalErr)
	}

	// Successful response with nil Request still returns the requested URL.
	hostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("nil-req")),
				Header:     http.Header{"X-Ok": []string{"1"}},
			}, nil
		}),
	}
	raw = awaitHost(t, qjs, `
const respRaw = await globalThis.__choysum_e2e_host__.fetch(`+jsonQuote(srv.URL)+`, '{}');
return JSON.parse(respRaw);
`)
	hostHTTPClient = oldClient
	var nilReq struct {
		Status int    `json:"status"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &nilReq); err != nil {
		t.Fatalf("parse nil req %s: %v", raw, err)
	}
	if nilReq.Status != 200 || nilReq.URL != srv.URL {
		t.Fatalf("nil request fallback: %+v", nilReq)
	}
}
