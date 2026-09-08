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

// minimalConsoleScript installs browser-like globals when the QuickJS host lacks them.
// Bundled FE deps (and some product modules) may touch these at import time.
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
      var utf8 = unescape(encodeURIComponent(str));
      var arr = new Uint8Array(utf8.length);
      for (var i = 0; i < utf8.length; i++) arr[i] = utf8.charCodeAt(i);
      return arr;
    };
  }
  if (typeof g.atob !== "function") {
    var b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";
    g.atob = function (input) {
      input = String(input).replace(/[=]+$/, "");
      if (input.length % 4 === 1) throw new Error("InvalidCharacterError");
      var str = "";
      for (var bc = 0, bs = 0, buffer, idx = 0; (buffer = input.charAt(idx++)); ) {
        buffer = b64.indexOf(buffer);
        if (buffer < 0) continue;
        bs = bc % 4 ? bs * 64 + buffer : buffer;
        if (bc++ % 4) {
          str += String.fromCharCode(255 & (bs >> ((-2 * bc) & 6)));
        }
      }
      return str;
    };
  }
  if (typeof g.AbortController !== "function") {
    g.AbortController = function AbortController() {
      var listeners = [];
      this.signal = {
        aborted: false,
        addEventListener: function (type, fn) {
          if (type === "abort" && typeof fn === "function") listeners.push(fn);
        },
        removeEventListener: function (type, fn) {
          if (type !== "abort") return;
          listeners = listeners.filter(function (x) { return x !== fn; });
        },
      };
      this.abort = function () {
        if (this.signal.aborted) return;
        this.signal.aborted = true;
        listeners.slice().forEach(function (fn) {
          try { fn(); } catch (e) {}
        });
      };
    };
  }
  function makeStorage() {
    var store = Object.create(null);
    return {
      getItem: function (k) { return Object.prototype.hasOwnProperty.call(store, k) ? store[k] : null; },
      setItem: function (k, v) { store[k] = String(v); },
      removeItem: function (k) { delete store[k]; },
      clear: function () { store = Object.create(null); },
      key: function (i) { return Object.keys(store)[i] || null; },
      get length() { return Object.keys(store).length; }
    };
  }
  if (typeof g.localStorage === "undefined" || g.localStorage === null) {
    g.localStorage = makeStorage();
  }
  if (typeof g.sessionStorage === "undefined" || g.sessionStorage === null) {
    g.sessionStorage = makeStorage();
  }
  if (typeof g.Blob !== "function") {
    g.Blob = function Blob(parts, options) {
      this._parts = parts || [];
      this.type = (options && options.type) || "";
      this.size = 0;
      for (var i = 0; i < this._parts.length; i++) {
        var p = this._parts[i];
        this.size += typeof p === "string" ? p.length : p && p.byteLength != null ? p.byteLength : String(p).length;
      }
    };
    g.Blob.prototype.arrayBuffer = function () {
      var parts = this._parts || [];
      var total = 0;
      for (var i = 0; i < parts.length; i++) {
        var p = parts[i];
        total += typeof p === "string" ? p.length : p && p.byteLength != null ? p.byteLength : String(p).length;
      }
      var out = new Uint8Array(total);
      var offset = 0;
      for (var j = 0; j < parts.length; j++) {
        var part = parts[j];
        if (typeof part === "string") {
          for (var k = 0; k < part.length; k++) out[offset++] = part.charCodeAt(k) & 255;
        } else if (part && part.byteLength != null) {
          var view = part instanceof Uint8Array ? part : new Uint8Array(part.buffer || part);
          out.set(view, offset);
          offset += view.length;
        } else {
          var s = String(part);
          for (var m = 0; m < s.length; m++) out[offset++] = s.charCodeAt(m) & 255;
        }
      }
      return Promise.resolve(out.buffer);
    };
  }
  if (typeof g.File !== "function") {
    g.File = function File(parts, name, options) {
      g.Blob.call(this, parts, options);
      this.name = name == null ? "" : String(name);
      this.lastModified = (options && options.lastModified) || Date.now();
    };
    g.File.prototype = Object.create(g.Blob.prototype);
    g.File.prototype.constructor = g.File;
  }
  if (!g.URL || typeof g.URL.createObjectURL !== "function") {
    var urlBase = g.URL || function URL() {};
    urlBase.createObjectURL = function () {
      return "blob:choysum-fe-unit/" + Math.random().toString(36).slice(2);
    };
    urlBase.revokeObjectURL = function () {};
    g.URL = urlBase;
  }
  if (typeof g.URLSearchParams !== "function") {
    g.URLSearchParams = function URLSearchParams(init) {
      this._pairs = [];
      var self = this;
      function appendPair(k, v) {
        self._pairs.push([String(k), String(v)]);
      }
      if (typeof init === "string") {
        var q = init.charAt(0) === "?" ? init.slice(1) : init;
        if (q) {
          q.split("&").forEach(function (part) {
            if (!part) return;
            var eq = part.indexOf("=");
            if (eq < 0) appendPair(decodeURIComponent(part.replace(/\+/g, " ")), "");
            else
              appendPair(
                decodeURIComponent(part.slice(0, eq).replace(/\+/g, " ")),
                decodeURIComponent(part.slice(eq + 1).replace(/\+/g, " "))
              );
          });
        }
      } else if (init && typeof init === "object") {
        Object.keys(init).forEach(function (k) {
          appendPair(k, init[k]);
        });
      }
      this.append = function (k, v) {
        appendPair(k, v);
      };
      this.set = function (k, v) {
        var key = String(k);
        this._pairs = this._pairs.filter(function (p) {
          return p[0] !== key;
        });
        appendPair(k, v);
      };
      this.get = function (k) {
        var key = String(k);
        for (var i = 0; i < this._pairs.length; i++) {
          if (this._pairs[i][0] === key) return this._pairs[i][1];
        }
        return null;
      };
      this.toString = function () {
        return this._pairs
          .map(function (p) {
            return encodeURIComponent(p[0]) + "=" + encodeURIComponent(p[1]);
          })
          .join("&");
      };
    };
  }
  // Minimal Intl for FE unit (dayjs timezone / request_timezone). Not a full ICU polyfill.
  // Always install: some QuickJS builds expose a hollow Intl without usable timeZone.
  var knownZones = {
    UTC: true,
    "Etc/UTC": true,
    "Asia/Shanghai": true,
    "America/New_York": true,
    "America/Chicago": true,
    "Europe/Berlin": true,
    "Europe/London": true,
    "Asia/Tokyo": true,
  };
  function isKnownZone(tz) {
    var v = String(tz || "").trim();
    if (!v) return false;
    if (knownZones[v]) return true;
    if (v === "Not/A_Zone" || v === "Also/Bad" || v === "Garbage" || v === "Local") return false;
    return /^[A-Za-z_]+\/[A-Za-z0-9_+-]+$/.test(v) || v === "UTC";
  }
  function DateTimeFormat(locales, options) {
    if (!(this instanceof DateTimeFormat)) {
      return new DateTimeFormat(locales, options);
    }
    options = options || {};
    var tz = options.timeZone || "UTC";
    if (options.timeZone != null && !isKnownZone(options.timeZone)) {
      throw new RangeError("Invalid time zone specified: " + options.timeZone);
    }
    this.resolvedOptions = function () {
      return { locale: String(locales || "en-US"), timeZone: String(tz || "UTC") };
    };
    this.format = function () {
      return "";
    };
  }
  g.Intl = { DateTimeFormat: DateTimeFormat };
  if (typeof g.Headers !== "function") {
    g.Headers = function Headers(init) {
      this._map = Object.create(null);
      var self = this;
      if (init && typeof init === "object") {
        Object.keys(init).forEach(function (k) {
          self.set(k, init[k]);
        });
      }
      this.set = function (k, v) {
        this._map[String(k).toLowerCase()] = String(v);
      };
      this.get = function (k) {
        var v = this._map[String(k).toLowerCase()];
        return v == null ? null : v;
      };
      this.has = function (k) {
        return Object.prototype.hasOwnProperty.call(this._map, String(k).toLowerCase());
      };
      this.append = function (k, v) {
        var key = String(k).toLowerCase();
        if (this._map[key] != null) this._map[key] += ", " + String(v);
        else this._map[key] = String(v);
      };
    };
  }
})();`

// InstallMinimalConsole injects browser-like console/storage helpers for QuickJS FE unit hosts.
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
