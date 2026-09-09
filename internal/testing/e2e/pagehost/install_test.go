// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestInstallDriveHostWithChrome(t *testing.T) {
	path, err := cdp.ResolveChromiumPath()
	if err != nil {
		t.Skipf("chromium unavailable: %v", err)
	}
	headless := true
	session, err := cdp.Start(nil, cdp.StartOptions{ExecPath: path, Headless: &headless})
	if err != nil {
		t.Fatalf("Start: %v", err)
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

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
