// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

// documentMinPolyfill installs a minimal document.createElement("div") used by
// @vue/shared decodeHtmlBrowser inside language-core (innerHTML / textContent /
// getAttribute). QuickJS has no DOM.
const documentMinPolyfill = `(function () {
  if (typeof globalThis.document !== "undefined" &&
      typeof globalThis.document.createElement === "function") {
    return;
  }

  var NAMED_ENTITIES = {
    nbsp: "\u00A0",
    quot: '"',
    apos: "'",
    lt: "<",
    gt: ">",
    amp: "&"
  };

  // Single-pass decode avoids secondary expansion (e.g. &#38;amp; → &amp; → &).
  function decodeEntities(s) {
    return String(s).replace(/&(?:(nbsp|quot|apos|lt|gt|amp)|#39|#x([0-9a-fA-F]+)|#(\d+));/gi, function (m, named, hex, dec) {
      if (named) return NAMED_ENTITIES[named.toLowerCase()];
      if (m.toLowerCase() === "&#39;") return "'";
      if (hex) {
        var h = parseInt(hex, 16);
        return isFinite(h) && h >= 0 && h <= 0x10FFFF ? String.fromCodePoint(h) : "";
      }
      if (dec) {
        var d = +dec;
        return isFinite(d) && d >= 0 && d <= 0x10FFFF ? String.fromCodePoint(d) : "";
      }
      return m;
    });
  }

  function Element(tag) {
    this.tagName = String(tag).toUpperCase();
    this.attrs = Object.create(null);
    this.children = [];
    this.childNodes = [];
    this._html = "";
    this._text = "";
  }

  Object.defineProperty(Element.prototype, "innerHTML", {
    get: function () { return this._html; },
    set: function (v) {
      this._html = String(v);
      this.children = [];
      this.childNodes = [];
      // Attribute decode path: <div foo="..."> or self-closing <div foo="..." />
      var attr = /^<div\s+foo="([^"]*)"\s*\/?>/i.exec(this._html);
      if (attr) {
        var child = new Element("div");
        child.attrs.foo = decodeEntities(attr[1]);
        this.children = [child];
        this.childNodes = [child];
        this._text = "";
        return;
      }
      // Text decode path: strip tags then unescape entities.
      this._text = decodeEntities(this._html.replace(/<[^>]*>/g, ""));
    }
  });

  Object.defineProperty(Element.prototype, "textContent", {
    get: function () { return this._text; },
    set: function (v) { this._text = String(v); }
  });

  Element.prototype.getAttribute = function (name) {
    var key = String(name);
    return Object.prototype.hasOwnProperty.call(this.attrs, key) ? this.attrs[key] : null;
  };
  Element.prototype.setAttribute = function (name, value) {
    this.attrs[String(name)] = String(value);
  };

  globalThis.document = {
    createElement: function (tag) { return new Element(tag); }
  };
})();
`
