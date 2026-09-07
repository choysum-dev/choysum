// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Minimal DOM for QuickJS Vue unit host (not happy-dom / jsdom).
 * Enough for createApp().mount(el), querySelector, and basic events.
 */
(function (global) {
  'use strict';
  if (global.document && global.document.__choysumMinimalDOM) {
    return;
  }

  var NODE_ELEMENT = 1;
  var NODE_TEXT = 3;
  var NODE_COMMENT = 8;
  var NODE_DOCUMENT = 9;

  function Node() {}
  Node.ELEMENT_NODE = NODE_ELEMENT;
  Node.TEXT_NODE = NODE_TEXT;
  Node.COMMENT_NODE = NODE_COMMENT;
  Node.DOCUMENT_NODE = NODE_DOCUMENT;

  function syncChildNodes(el) {
    el.childNodes = el._children.slice();
    el.children = el._children.filter(function (c) {
      return c.nodeType === NODE_ELEMENT;
    });
  }

  function Element(tagName) {
    this.nodeType = NODE_ELEMENT;
    this.tagName = String(tagName || 'DIV').toUpperCase();
    this.nodeName = this.tagName;
    this.namespaceURI = null;
    this.attrs = Object.create(null);
    this._children = [];
    this.childNodes = [];
    this.children = [];
    this.parentNode = null;
    this._text = '';
    this._html = '';
    this.style = {};
    this.className = '';
    this.id = '';
    this._listeners = Object.create(null);
    this.ownerDocument = null;
  }

  Element.prototype.appendChild = function (child) {
    if (!child) return child;
    if (child.parentNode) {
      child.parentNode.removeChild(child);
    }
    child.parentNode = this;
    this._children.push(child);
    syncChildNodes(this);
    return child;
  };

  Element.prototype.removeChild = function (child) {
    var i = this._children.indexOf(child);
    if (i >= 0) {
      this._children.splice(i, 1);
      child.parentNode = null;
      syncChildNodes(this);
    }
    return child;
  };

  Element.prototype.insertBefore = function (newNode, ref) {
    if (!ref) return this.appendChild(newNode);
    var i = this._children.indexOf(ref);
    if (i < 0) return this.appendChild(newNode);
    if (newNode.parentNode) newNode.parentNode.removeChild(newNode);
    newNode.parentNode = this;
    this._children.splice(i, 0, newNode);
    syncChildNodes(this);
    return newNode;
  };

  Element.prototype.setAttribute = function (name, value) {
    var key = String(name);
    var val = String(value);
    this.attrs[key] = val;
    if (key === 'class') this.className = val;
    if (key === 'id') this.id = val;
  };

  Element.prototype.removeAttribute = function (name) {
    var key = String(name);
    delete this.attrs[key];
    if (key === 'class') this.className = '';
    if (key === 'id') this.id = '';
  };

  Element.prototype.getAttribute = function (name) {
    var key = String(name);
    return Object.prototype.hasOwnProperty.call(this.attrs, key) ? this.attrs[key] : null;
  };

  Element.prototype.hasAttribute = function (name) {
    return Object.prototype.hasOwnProperty.call(this.attrs, String(name));
  };

  Element.prototype.addEventListener = function (type, fn) {
    var t = String(type);
    if (!this._listeners[t]) this._listeners[t] = [];
    this._listeners[t].push(fn);
  };

  Element.prototype.removeEventListener = function (type, fn) {
    var t = String(type);
    var list = this._listeners[t];
    if (!list) return;
    var i = list.indexOf(fn);
    if (i >= 0) list.splice(i, 1);
  };

  Element.prototype.dispatchEvent = function (evt) {
    if (!evt || !evt.type) return false;
    var list = (this._listeners[evt.type] || []).slice();
    for (var i = 0; i < list.length; i++) {
      try {
        list[i].call(this, evt);
      } catch (_) {}
    }
    return true;
  };

  Element.prototype.querySelector = function (sel) {
    var all = this.querySelectorAll(sel);
    return all.length ? all[0] : null;
  };

  Element.prototype.querySelectorAll = function (sel) {
    var out = [];
    var selector = String(sel || '').trim();
    function walk(node) {
      if (node.nodeType !== NODE_ELEMENT) return;
      if (match(node, selector)) out.push(node);
      for (var i = 0; i < node._children.length; i++) {
        walk(node._children[i]);
      }
    }
    function match(el, s) {
      if (!s) return false;
      if (s.charAt(0) === '.') {
        var cls = s.slice(1);
        var cn = el.className || el.getAttribute('class') || '';
        return (' ' + cn + ' ').indexOf(' ' + cls + ' ') >= 0;
      }
      if (s.charAt(0) === '#') {
        return el.id === s.slice(1) || el.getAttribute('id') === s.slice(1);
      }
      if (s.charAt(0) === '[') {
        var m = /^\[([^=\]]+)(?:=["']?([^"'\]]*)["']?)?\]$/.exec(s);
        if (!m) return false;
        var got = el.getAttribute(m[1]);
        if (m[2] === undefined) return got != null;
        return got === m[2];
      }
      return el.tagName === s.toUpperCase();
    }
    for (var i = 0; i < this._children.length; i++) {
      walk(this._children[i]);
    }
    return out;
  };

  Object.defineProperty(Element.prototype, 'textContent', {
    get: function () {
      if (this._children.length === 0) return this._text;
      var parts = [];
      for (var i = 0; i < this._children.length; i++) {
        var c = this._children[i];
        if (c.nodeType === NODE_TEXT) parts.push(c.data || '');
        else if (c.nodeType === NODE_ELEMENT) parts.push(c.textContent || '');
      }
      return parts.join('');
    },
    set: function (v) {
      this._children = [];
      syncChildNodes(this);
      this._text = String(v == null ? '' : v);
      if (this._text) {
        var t = this.ownerDocument
          ? this.ownerDocument.createTextNode(this._text)
          : { nodeType: NODE_TEXT, data: this._text, parentNode: null };
        t.parentNode = this;
        this._children.push(t);
        syncChildNodes(this);
      }
    },
  });

  Object.defineProperty(Element.prototype, 'innerHTML', {
    get: function () {
      return this._html || this.textContent;
    },
    set: function (v) {
      this._html = String(v == null ? '' : v);
      // Host tests use Vue renderer; keep a text fallback for polyfill-only paths.
      this.textContent = this._html.replace(/<[^>]*>/g, '');
    },
  });

  Object.defineProperty(Element.prototype, 'nextSibling', {
    get: function () {
      if (!this.parentNode) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i >= 0 && i + 1 < sibs.length ? sibs[i + 1] : null;
    },
  });

  Object.defineProperty(Element.prototype, 'firstChild', {
    get: function () {
      return this._children.length ? this._children[0] : null;
    },
  });

  Object.defineProperty(Element.prototype, 'parentElement', {
    get: function () {
      return this.parentNode && this.parentNode.nodeType === NODE_ELEMENT ? this.parentNode : null;
    },
  });

  function TextNode(data) {
    this.nodeType = NODE_TEXT;
    this.nodeName = '#text';
    this.data = String(data == null ? '' : data);
    this.parentNode = null;
    this.ownerDocument = null;
  }
  Object.defineProperty(TextNode.prototype, 'textContent', {
    get: function () {
      return this.data;
    },
    set: function (v) {
      this.data = String(v == null ? '' : v);
    },
  });

  function CommentNode(data) {
    this.nodeType = NODE_COMMENT;
    this.nodeName = '#comment';
    this.data = String(data == null ? '' : data);
    this.parentNode = null;
    this.ownerDocument = null;
  }

  function Document() {
    this.nodeType = NODE_DOCUMENT;
    this.nodeName = '#document';
    this.__choysumMinimalDOM = true;
    this.body = new Element('body');
    this.body.ownerDocument = this;
    this.documentElement = new Element('html');
    this.documentElement.ownerDocument = this;
    this.documentElement.appendChild(this.body);
  }

  Document.prototype.createElement = function (tag) {
    var el = new Element(tag);
    el.ownerDocument = this;
    return el;
  };

  Document.prototype.createElementNS = function (_ns, tag) {
    return this.createElement(tag);
  };

  Document.prototype.createTextNode = function (data) {
    var t = new TextNode(data);
    t.ownerDocument = this;
    return t;
  };

  Document.prototype.createComment = function (data) {
    var c = new CommentNode(data);
    c.ownerDocument = this;
    return c;
  };

  Document.prototype.createDocumentFragment = function () {
    return this.createElement('fragment');
  };

  Document.prototype.querySelector = function (sel) {
    return this.documentElement.querySelector(sel);
  };

  Document.prototype.querySelectorAll = function (sel) {
    return this.documentElement.querySelectorAll(sel);
  };

  Document.prototype.getElementById = function (id) {
    var all = this.querySelectorAll('#' + id);
    return all.length ? all[0] : null;
  };

  function Event(type, init) {
    this.type = String(type);
    this.bubbles = !!(init && init.bubbles);
    this.cancelable = !!(init && init.cancelable);
    this.defaultPrevented = false;
    this.target = null;
    this.currentTarget = null;
  }
  Event.prototype.preventDefault = function () {
    this.defaultPrevented = true;
  };
  Event.prototype.stopPropagation = function () {};

  var doc = new Document();
  global.document = doc;
  global.window = global;
  global.self = global;
  global.Node = Node;
  global.Element = Element;
  global.HTMLElement = Element;
  global.SVGElement = Element;
  global.Text = TextNode;
  global.Comment = CommentNode;
  global.Document = Document;
  global.Event = Event;
  global.CustomEvent = Event;
  global.navigator = global.navigator || { userAgent: 'choysum-minimal-dom' };
  global.location = global.location || { href: 'http://localhost/', protocol: 'http:' };
  global.getComputedStyle =
    global.getComputedStyle ||
    function () {
      return {};
    };
  global.requestAnimationFrame =
    global.requestAnimationFrame ||
    function (cb) {
      return setTimeout(function () {
        cb(Date.now());
      }, 0);
    };
  global.cancelAnimationFrame =
    global.cancelAnimationFrame ||
    function (id) {
      clearTimeout(id);
    };
  if (typeof global.queueMicrotask !== 'function') {
    global.queueMicrotask = function (fn) {
      Promise.resolve().then(fn);
    };
  }
})(typeof globalThis !== 'undefined' ? globalThis : this);
