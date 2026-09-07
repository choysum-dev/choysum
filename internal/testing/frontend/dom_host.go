// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysummount"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Test seam for QuickJS timer bootstrap (overridden in unit tests).
var bootstrapVueHostTimers = func(engine *quickjsengine.QuickjsEngine) bool {
	return engine.Ctx.BootstrapTimers()
}

// InstallMinimalDOM injects the FE unit minimal DOM polyfill into a QuickJS engine.
// Scope is frozen for PR-unit-vue-host: document/window/Element enough for Vue mount,
// querySelector, and dispatchEvent — not a full happy-dom/jsdom.
func InstallMinimalDOM(engine jsengine.JsEngine) error {
	if engine == nil {
		return xfmt.Errorf("frontend host: nil engine")
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysummount/dom.js", Content: choysummount.MinimalDOMScript},
	}); err != nil {
		return xfmt.Errorf("frontend host: InstallMinimalDOM: %w", err)
	}
	return nil
}

// minimalConsoleScript installs a no-op console when the QuickJS host lacks one.
// Bundled FE deps (and some product modules) may touch console at import time.
const minimalConsoleScript = `(function () {
  var g = globalThis;
  if (!g.console || typeof g.console.error !== "function") {
    var noop = function () {};
    g.console = { log: noop, info: noop, warn: noop, error: noop, debug: noop, trace: noop };
  }
  if (typeof g.isSecureContext === "undefined") g.isSecureContext = false;
  if (typeof g.TextEncoder !== "function") {
    g.TextEncoder = function TextEncoder() {};
    g.TextEncoder.prototype.encode = function (str) {
      str = String(str == null ? "" : str);
      var arr = new Uint8Array(str.length);
      for (var i = 0; i < str.length; i++) arr[i] = str.charCodeAt(i) & 0xff;
      return arr;
    };
  }
  if (typeof g.atob !== "function") {
    g.atob = function () {
      throw new Error("atob not available in FE unit host");
    };
  }
  if (typeof g.localStorage === "undefined" || g.localStorage === null) {
    var store = Object.create(null);
    g.localStorage = {
      getItem: function (k) { return Object.prototype.hasOwnProperty.call(store, k) ? store[k] : null; },
      setItem: function (k, v) { store[k] = String(v); },
      removeItem: function (k) { delete store[k]; },
      clear: function () { store = Object.create(null); },
      key: function (i) { return Object.keys(store)[i] || null; },
      get length() { return Object.keys(store).length; }
    };
  }
  if (typeof g.sessionStorage === "undefined" || g.sessionStorage === null) {
    g.sessionStorage = g.localStorage;
  }
})();`

// InstallMinimalConsole injects a no-op console for QuickJS FE unit hosts.
func InstallMinimalConsole(engine jsengine.JsEngine) error {
	if engine == nil {
		return xfmt.Errorf("frontend host: nil engine")
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysummount/console.js", Content: minimalConsoleScript},
	}); err != nil {
		return xfmt.Errorf("frontend host: InstallMinimalConsole: %w", err)
	}
	return nil
}

// PrepareVueHostEngine installs minimal DOM and bootstraps QuickJS timers (setTimeout).
// Call before loading a Vue host bundle that uses flushPromises / async updates.
func PrepareVueHostEngine(engine jsengine.JsEngine) error {
	if err := InstallMinimalConsole(engine); err != nil {
		return err
	}
	if err := InstallMinimalDOM(engine); err != nil {
		return err
	}
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		return nil
	}
	if !bootstrapVueHostTimers(qjs) {
		return xfmt.Errorf("frontend host: BootstrapTimers failed")
	}
	return nil
}
