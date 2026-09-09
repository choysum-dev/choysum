// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestInstallRegistersGlobalsWithoutBrowser(t *testing.T) {
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// session may be nil: Install only registers bindings; newPage fails later.
	if err := Install(engine, nil, `{"baseURL":"http://127.0.0.1:9"}`); err != nil {
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
	err := Install(nil, nil, "{}")
	if err == nil || !strings.Contains(err.Error(), "QuickjsEngine") {
		t.Fatalf("expected QuickjsEngine error, got %v", err)
	}
}
