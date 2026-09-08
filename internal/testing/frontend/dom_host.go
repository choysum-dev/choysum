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
  // URLSearchParams must expose has/set for document binding token append.
  if (typeof g.URLSearchParams !== "function" || typeof g.URLSearchParams.prototype?.has !== "function") {
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
      this.has = function (k) {
        return this.get(k) != null;
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
  // Minimal URL: pathname/search/searchParams enough for binding preview token rewrite.
  var needURL = true;
  try {
    var probe = new g.URL("/x", "http://choysum.local");
    needURL = !(
      probe &&
      probe.searchParams &&
      typeof probe.searchParams.has === "function" &&
      typeof probe.searchParams.set === "function" &&
      typeof probe.pathname === "string"
    );
  } catch (e) {
    needURL = true;
  }
  if (needURL) {
    var prevCreateObjectURL = g.URL && g.URL.createObjectURL;
    var prevRevokeObjectURL = g.URL && g.URL.revokeObjectURL;
    g.URL = function URL(input, base) {
      var href = String(input == null ? "" : input);
      if (base && href && href.charAt(0) === "/") {
        var b = String(base);
        var slash = b.indexOf("/", b.indexOf("//") >= 0 ? b.indexOf("//") + 2 : 0);
        href = (slash >= 0 ? b.slice(0, slash) : b) + href;
      } else if (base && href && !/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(href)) {
        href = String(base).replace(/\/?$/, "/") + href.replace(/^\//, "");
      }
      var hashIdx = href.indexOf("#");
      this.hash = hashIdx >= 0 ? href.slice(hashIdx) : "";
      var noHash = hashIdx >= 0 ? href.slice(0, hashIdx) : href;
      var qIdx = noHash.indexOf("?");
      this.search = qIdx >= 0 ? noHash.slice(qIdx) : "";
      var noQuery = qIdx >= 0 ? noHash.slice(0, qIdx) : noHash;
      var protoIdx = noQuery.indexOf("://");
      var afterProto = protoIdx >= 0 ? noQuery.slice(protoIdx + 3) : noQuery;
      var pathIdx = afterProto.indexOf("/");
      this.pathname = pathIdx >= 0 ? afterProto.slice(pathIdx) : "/";
      this.origin = protoIdx >= 0 ? noQuery.slice(0, protoIdx + 3 + (pathIdx >= 0 ? pathIdx : afterProto.length)) : "";
      this.href = href;
      this.searchParams = new g.URLSearchParams(this.search);
      var self = this;
      var syncSearch = function () {
        var s = self.searchParams.toString();
        self.search = s ? "?" + s : "";
        self.href = self.origin + self.pathname + self.search + self.hash;
      };
      var origSet = this.searchParams.set;
      this.searchParams.set = function (k, v) {
        origSet.call(this, k, v);
        syncSearch();
      };
      var origAppend = this.searchParams.append;
      this.searchParams.append = function (k, v) {
        origAppend.call(this, k, v);
        syncSearch();
      };
      this.toString = function () {
        return this.origin + this.pathname + this.search + this.hash;
      };
    };
    g.URL.createObjectURL =
      prevCreateObjectURL ||
      function () {
        return "blob:choysum-fe-unit/" + Math.random().toString(36).slice(2);
      };
    g.URL.revokeObjectURL = prevRevokeObjectURL || function () {};
  }
  // Minimal Intl for dayjs timezone: formatToParts + toLocaleString(timeZone).
  // Fixed offsets for zones used by FE unit (not a full ICU database).
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
  function nthSundayUTC(year, monthIndex, n) {
    var d = new Date(Date.UTC(year, monthIndex, 1));
    var day = d.getUTCDay();
    var firstSun = 1 + ((7 - day) % 7);
    return Date.UTC(year, monthIndex, firstSun + (n - 1) * 7);
  }
  function usDstOffsetMinutes(dateMs, standard, daylight) {
    var y = new Date(dateMs).getUTCFullYear();
    // US: 2nd Sunday March 07:00Z (EST) / 1st Sunday November 06:00Z (EDT) approx for -5/-4.
    var start = nthSundayUTC(y, 2, 2) + 7 * 3600 * 1000;
    var end = nthSundayUTC(y, 10, 1) + 6 * 3600 * 1000;
    return dateMs >= start && dateMs < end ? daylight : standard;
  }
  function zoneOffsetMinutes(tz, dateMs) {
    var z = String(tz || "UTC");
    if (z === "UTC" || z === "Etc/UTC") return 0;
    if (z === "Asia/Shanghai" || z === "Asia/Chongqing") return 480;
    if (z === "Asia/Tokyo") return 540;
    if (z === "America/New_York") return usDstOffsetMinutes(dateMs, -300, -240);
    if (z === "America/Chicago") return usDstOffsetMinutes(dateMs, -360, -300);
    if (z === "Europe/Berlin") {
      var yb = new Date(dateMs).getUTCFullYear();
      // EU rough: last Sunday March 01:00Z → last Sunday October 01:00Z
      var mar = nthSundayUTC(yb, 2, 5);
      while (new Date(mar).getUTCMonth() !== 2) mar -= 7 * 86400000;
      var oct = nthSundayUTC(yb, 9, 5);
      while (new Date(oct).getUTCMonth() !== 9) oct -= 7 * 86400000;
      return dateMs >= mar + 3600000 && dateMs < oct + 3600000 ? 120 : 60;
    }
    if (z === "Europe/London") {
      var yl = new Date(dateMs).getUTCFullYear();
      var marL = nthSundayUTC(yl, 2, 5);
      while (new Date(marL).getUTCMonth() !== 2) marL -= 7 * 86400000;
      var octL = nthSundayUTC(yl, 9, 5);
      while (new Date(octL).getUTCMonth() !== 9) octL -= 7 * 86400000;
      return dateMs >= marL + 3600000 && dateMs < octL + 3600000 ? 60 : 0;
    }
    return 0;
  }
  function pad2(n) {
    return n < 10 ? "0" + n : String(n);
  }
  function wallParts(dateMs, tz) {
    var off = zoneOffsetMinutes(tz, dateMs);
    var d = new Date(dateMs + off * 60000);
    return {
      year: d.getUTCFullYear(),
      month: d.getUTCMonth() + 1,
      day: d.getUTCDate(),
      hour: d.getUTCHours(),
      minute: d.getUTCMinutes(),
      second: d.getUTCSeconds(),
      offset: off,
    };
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
    var hour12 = options.hour12 === true;
    var wantName = options.timeZoneName;
    this.resolvedOptions = function () {
      return { locale: String(locales || "en-US"), timeZone: String(tz || "UTC") };
    };
    this.formatToParts = function (date) {
      var ms = date == null ? Date.now() : +new Date(date);
      if (!isFinite(ms)) ms = Date.now();
      var w = wallParts(ms, tz);
      var hour = w.hour;
      var parts = [
        { type: "month", value: pad2(w.month) },
        { type: "literal", value: "/" },
        { type: "day", value: pad2(w.day) },
        { type: "literal", value: "/" },
        { type: "year", value: String(w.year) },
        { type: "literal", value: ", " },
      ];
      if (hour12) {
        var ap = hour >= 12 ? "PM" : "AM";
        var h12 = hour % 12;
        if (h12 === 0) h12 = 12;
        parts.push({ type: "hour", value: pad2(h12) });
        parts.push({ type: "literal", value: ":" });
        parts.push({ type: "minute", value: pad2(w.minute) });
        parts.push({ type: "literal", value: ":" });
        parts.push({ type: "second", value: pad2(w.second) });
        parts.push({ type: "literal", value: " " });
        parts.push({ type: "dayPeriod", value: ap });
      } else {
        // dayjs timezone uses hour12:false; map 24 → 24 for midnight-next semantics.
        parts.push({ type: "hour", value: pad2(hour) });
        parts.push({ type: "literal", value: ":" });
        parts.push({ type: "minute", value: pad2(w.minute) });
        parts.push({ type: "literal", value: ":" });
        parts.push({ type: "second", value: pad2(w.second) });
      }
      if (wantName) {
        parts.push({ type: "literal", value: " " });
        parts.push({ type: "timeZoneName", value: wantName === "long" ? String(tz) : "GMT" });
      }
      return parts;
    };
    this.format = function (date) {
      return this.formatToParts(date)
        .map(function (p) {
          return p.value;
        })
        .join("");
    };
  }
  g.Intl = { DateTimeFormat: DateTimeFormat };
  var origToLocaleString = Date.prototype.toLocaleString;
  Date.prototype.toLocaleString = function (locales, options) {
    if (options && options.timeZone) {
      return new DateTimeFormat(locales || "en-US", options).format(this);
    }
    return origToLocaleString ? origToLocaleString.call(this, locales, options) : String(this);
  };
  // BootstrapTimers only injects setTimeout/clearTimeout; chatter tips need setInterval.
  if (typeof g.setInterval !== "function") {
    g.setInterval = function (fn, ms) {
      var st = g.setTimeout;
      if (typeof st !== "function") throw new Error("setInterval requires setTimeout");
      var args = Array.prototype.slice.call(arguments, 2);
      var handle = { cleared: false, tid: 0 };
      function tick() {
        if (handle.cleared) return;
        try {
          fn.apply(null, args);
        } catch (e) {}
        if (!handle.cleared) handle.tid = st(tick, ms);
      }
      handle.tid = st(tick, ms);
      return handle;
    };
    g.clearInterval = function (handle) {
      if (!handle || handle.cleared) return;
      handle.cleared = true;
      if (typeof g.clearTimeout === "function" && handle.tid) g.clearTimeout(handle.tid);
    };
  }
  if (typeof g.Headers !== "function") {
    g.Headers = function Headers(init) {
      this._map = Object.create(null);
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
      this.delete = function (k) {
        delete this._map[String(k).toLowerCase()];
      };
      this.forEach = function (fn, thisArg) {
        var keys = Object.keys(this._map);
        for (var i = 0; i < keys.length; i++) {
          fn.call(thisArg, this._map[keys[i]], keys[i], this);
        }
      };
      if (init == null) return;
      var self = this;
      if (Array.isArray(init)) {
        for (var i = 0; i < init.length; i++) {
          var pair = init[i];
          if (pair && pair.length >= 2) self.append(pair[0], pair[1]);
        }
      } else if (typeof init.forEach === "function") {
        // Headers/Map forEach: (value, key)
        init.forEach(function (v, k) {
          self.append(k, v);
        });
      } else if (typeof init === "object") {
        Object.keys(init).forEach(function (k) {
          self.set(k, init[k]);
        });
      }
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
