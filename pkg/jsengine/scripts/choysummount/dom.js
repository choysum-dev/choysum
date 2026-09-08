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
    this.style = createStyle();
    this.className = '';
    this.id = '';
    this._listeners = Object.create(null);
    this.ownerDocument = null;
  }

  function cssCamel(name) {
    return String(name || '').replace(/-([a-z])/g, function (_m, c) {
      return c.toUpperCase();
    });
  }

  function createStyle() {
    var style = {
      getPropertyValue: function (name) {
        var key = cssCamel(name);
        var v = style[key];
        return v == null ? '' : String(v);
      },
      setProperty: function (name, value) {
        style[cssCamel(name)] = String(value == null ? '' : value);
      },
      removeProperty: function (name) {
        var key = cssCamel(name);
        var prev = style[key] == null ? '' : String(style[key]);
        style[key] = '';
        return prev;
      },
    };
    return style;
  }

  Element.prototype.appendChild = function (child) {
    if (!child) return child;
    if (child.parentNode) {
      child.parentNode.removeChild(child);
    }
    this._html = '';
    child.parentNode = this;
    this._children.push(child);
    syncChildNodes(this);
    return child;
  };

  Element.prototype.removeChild = function (child) {
    var i = this._children.indexOf(child);
    if (i >= 0) {
      this._html = '';
      this._children.splice(i, 1);
      child.parentNode = null;
      syncChildNodes(this);
    }
    return child;
  };

  Element.prototype.remove = function () {
    if (this.parentNode && typeof this.parentNode.removeChild === 'function') {
      this.parentNode.removeChild(this);
    }
  };

  Element.prototype.insertBefore = function (newNode, ref) {
    if (!ref) return this.appendChild(newNode);
    // DOM: insertBefore(node, node) is a no-op that returns the node.
    if (newNode === ref) return newNode;
    // Remove first so indexOf(ref) stays valid when newNode was already a sibling.
    if (newNode && newNode.parentNode) newNode.parentNode.removeChild(newNode);
    var i = this._children.indexOf(ref);
    if (i < 0) return this.appendChild(newNode);
    this._html = '';
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
    if (key === 'value' && this._formValue !== undefined) this._formValue = val;
  };

  Element.prototype.removeAttribute = function (name) {
    var key = String(name);
    delete this.attrs[key];
    if (key === 'class') this.className = '';
    if (key === 'id') this.id = '';
    // Live IDL state (_formValue / _checked) is independent of content attributes.
  };

  Object.defineProperty(Element.prototype, 'classList', {
    get: function () {
      var el = this;
      function tokens() {
        return String(el.className || '')
          .replace(/\s+/g, ' ')
          .trim()
          .split(' ')
          .filter(Boolean);
      }
      function write(list) {
        el.className = list.join(' ');
        el.attrs.class = el.className;
      }
      return {
        add: function () {
          var list = tokens();
          for (var i = 0; i < arguments.length; i++) {
            var t = String(arguments[i] || '');
            if (t && list.indexOf(t) < 0) list.push(t);
          }
          write(list);
        },
        remove: function () {
          var list = tokens();
          for (var i = 0; i < arguments.length; i++) {
            var t = String(arguments[i] || '');
            var idx = list.indexOf(t);
            if (idx >= 0) list.splice(idx, 1);
          }
          write(list);
        },
        toggle: function (token, force) {
          var t = String(token || '').trim();
          if (!t) return false;
          var list = tokens();
          var idx = list.indexOf(t);
          var shouldAdd = force === undefined ? idx < 0 : !!force;
          if (shouldAdd && idx < 0) list.push(t);
          if (!shouldAdd && idx >= 0) list.splice(idx, 1);
          write(list);
          return shouldAdd;
        },
        contains: function (token) {
          return tokens().indexOf(String(token || '')) >= 0;
        },
        toString: function () {
          return tokens().join(' ');
        },
      };
    },
  });

  Object.defineProperty(Element.prototype, 'dataset', {
    get: function () {
      var el = this;
      var out = {};
      Object.keys(el.attrs).forEach(function (key) {
        if (key.slice(0, 5) !== 'data-') return;
        var raw = key.slice(5);
        var camel = raw.replace(/-([a-z])/g, function (_m, c) {
          return c.toUpperCase();
        });
        out[camel] = el.attrs[key];
      });
      return out;
    },
  });

  Object.defineProperty(Element.prototype, 'value', {
    get: function () {
      if (this._formValue !== undefined) return this._formValue;
      var attr = this.getAttribute('value');
      return attr == null ? '' : attr;
    },
    set: function (v) {
      this._formValue = String(v == null ? '' : v);
      this.attrs.value = this._formValue;
    },
  });

  Object.defineProperty(Element.prototype, 'checked', {
    get: function () {
      if (this._checked !== undefined) return !!this._checked;
      return this.hasAttribute('checked');
    },
    set: function (v) {
      this._checked = !!v;
      if (this._checked) this.attrs.checked = '';
      else delete this.attrs.checked;
    },
  });

  Element.prototype.focus = function () {
    var doc = this.ownerDocument;
    if (doc && doc.activeElement && doc.activeElement !== this && typeof doc.activeElement.blur === 'function') {
      doc.activeElement.blur();
    }
    if (doc) doc.activeElement = this;
    this.dispatchEvent(new Event('focus', { bubbles: false }));
  };
  Element.prototype.blur = function () {
    var doc = this.ownerDocument;
    if (doc && doc.activeElement === this) doc.activeElement = null;
    this.dispatchEvent(new Event('blur', { bubbles: false }));
  };
  Element.prototype.click = function () {
    var tag = String(this.tagName || '').toLowerCase();
    var typ = String(this.getAttribute('type') || this.type || '').toLowerCase();
    var disabled = this.getAttribute('disabled') != null || this.disabled === true;
    if (disabled) return;
    var checkedChanged = false;
    if (tag === 'input' && typ === 'checkbox') {
      this.checked = !this.checked;
      checkedChanged = true;
    }
    if (tag === 'input' && typ === 'radio') {
      var name = String(this.getAttribute('name') || this.name || '');
      var wasChecked = !!this.checked;
      this.checked = true;
      if (name && this.ownerDocument && typeof this.ownerDocument.querySelectorAll === 'function') {
        var peers = this.ownerDocument.querySelectorAll('input');
        for (var i = 0; i < peers.length; i++) {
          var peer = peers[i];
          if (peer === this) continue;
          var peerTyp = String(peer.getAttribute('type') || peer.type || '').toLowerCase();
          if (peerTyp !== 'radio') continue;
          var peerName = String(peer.getAttribute('name') || peer.name || '');
          if (peerName === name) peer.checked = false;
        }
      }
      checkedChanged = !wasChecked;
    }
    if (checkedChanged) {
      this.dispatchEvent(new Event('input', { bubbles: true }));
      this.dispatchEvent(new Event('change', { bubbles: true }));
    }
    this.dispatchEvent(new Event('click', { bubbles: true }));
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
    if (this._listeners[t].indexOf(fn) === -1) {
      this._listeners[t].push(fn);
    }
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
    if (!evt.target) evt.target = this;
    var curr = this;
    var firstErr = null;
    while (curr) {
      evt.currentTarget = curr;
      var list = (curr._listeners && curr._listeners[evt.type] ? curr._listeners[evt.type] : []).slice();
      for (var i = 0; i < list.length; i++) {
        if (evt._stopImmediate) break;
        try {
          list[i].call(curr, evt);
        } catch (e) {
          if (!firstErr) firstErr = e;
        }
      }
      if (!evt.bubbles || evt.cancelBubble || evt._stopImmediate) break;
      var parent = curr.parentNode;
      curr = parent && parent.nodeType === NODE_ELEMENT ? parent : null;
    }
    if (firstErr) throw firstErr;
    return true;
  };

  Element.prototype.querySelector = function (sel) {
    var all = this.querySelectorAll(sel);
    return all.length ? all[0] : null;
  };

  // Shared with Element.prototype.matches so closest/matches honor the same
  // selector contract as querySelectorAll (class/id/tag/attr/tag.class/tag#id).
  function matchSelector(el, s) {
    s = String(s || '').trim();
    if (!s) return false;
    // Fail loud on forms this host does not implement (avoid false-negative finds).
    if (/[\s,>+~]/.test(s)) {
      throw new Error('choysum minimal DOM: unsupported selector: ' + s);
    }
    if (s.charAt(0) === '.') {
      // Reject .a.b / .a#id / .a[attr] (but allow simple .class-name).
      if (/[.#\[]/.test(s.slice(1))) {
        throw new Error('choysum minimal DOM: unsupported selector: ' + s);
      }
      var cls = s.slice(1);
      var cn = (el.className || el.getAttribute('class') || '').replace(/\s+/g, ' ').trim();
      return (' ' + cn + ' ').indexOf(' ' + cls + ' ') >= 0;
    }
    if (s.charAt(0) === '#') {
      if (/[.#\[]/.test(s.slice(1))) {
        throw new Error('choysum minimal DOM: unsupported selector: ' + s);
      }
      return el.id === s.slice(1) || el.getAttribute('id') === s.slice(1);
    }
    if (s.charAt(0) === '[') {
      var m = /^\[([^=\]]+)(?:=["']?([^"'\]]*)["']?)?\]$/.exec(s);
      if (!m) {
        throw new Error('choysum minimal DOM: unsupported attribute selector: ' + s);
      }
      var got = el.getAttribute(m[1]);
      if (m[2] === undefined) return got != null;
      return got === m[2];
    }
    // tag, tag.class, or tag#id (single class / id only).
    var tagClass = /^([a-zA-Z][\w-]*)\.([^\s.#\[]+)$/.exec(s);
    if (tagClass) {
      if (el.tagName !== tagClass[1].toUpperCase()) return false;
      var cls2 = tagClass[2];
      var cn2 = (el.className || el.getAttribute('class') || '').replace(/\s+/g, ' ').trim();
      return (' ' + cn2 + ' ').indexOf(' ' + cls2 + ' ') >= 0;
    }
    var tagId = /^([a-zA-Z][\w-]*)#([^\s.#\[]+)$/.exec(s);
    if (tagId) {
      if (el.tagName !== tagId[1].toUpperCase()) return false;
      return el.id === tagId[2] || el.getAttribute('id') === tagId[2];
    }
    // Tag name only — reject unsupported compound selectors.
    if (!/^[a-zA-Z][\w-]*$/.test(s)) {
      throw new Error('choysum minimal DOM: unsupported selector: ' + s);
    }
    return el.tagName === s.toUpperCase();
  }

  Element.prototype.querySelectorAll = function (sel) {
    var out = [];
    var selector = String(sel || '').trim();
    function walk(node) {
      if (node.nodeType !== NODE_ELEMENT) return;
      if (matchSelector(node, selector)) out.push(node);
      for (var i = 0; i < node._children.length; i++) {
        walk(node._children[i]);
      }
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
      for (var i = 0; i < this._children.length; i++) {
        this._children[i].parentNode = null;
      }
      this._children = [];
      this._html = '';
      syncChildNodes(this);
      this._text = String(v == null ? '' : v);
      if (this._text) {
        var doc = this.ownerDocument || global.document;
        var t = doc.createTextNode(this._text);
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
      // Test-host plaintext projection only (not a sanitizer): drop tags so blank
      // markup like <p></p> yields empty textContent for normalizeHtmlForStore.
      this.textContent = this._html.replace(/<[^>]*>/g, '');
    },
  });

  Object.defineProperty(Element.prototype, 'nextSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i >= 0 && i + 1 < sibs.length ? sibs[i + 1] : null;
    },
  });

  Object.defineProperty(Element.prototype, 'previousSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i > 0 ? sibs[i - 1] : null;
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

  Element.prototype.getBoundingClientRect = function () {
    return { top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0, x: 0, y: 0 };
  };

  Element.prototype.closest = function (sel) {
    var el = this;
    var selector = String(sel || '');
    while (el && el.nodeType === NODE_ELEMENT) {
      if (typeof el.matches === 'function' && el.matches(selector)) return el;
      el = el.parentElement || el.parentNode;
    }
    return null;
  };

  Element.prototype.matches = function (sel) {
    return matchSelector(this, sel);
  };

  function TextNode(data) {
    this.nodeType = NODE_TEXT;
    this.nodeName = '#text';
    this.data = String(data == null ? '' : data);
    this.parentNode = null;
    this.ownerDocument = null;
  }
  Object.defineProperty(TextNode.prototype, 'nodeValue', {
    get: function () {
      return this.data;
    },
    set: function (v) {
      this.data = String(v == null ? '' : v);
    },
  });
  Object.defineProperty(TextNode.prototype, 'textContent', {
    get: function () {
      return this.data;
    },
    set: function (v) {
      this.data = String(v == null ? '' : v);
    },
  });
  Object.defineProperty(TextNode.prototype, 'nextSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i >= 0 && i + 1 < sibs.length ? sibs[i + 1] : null;
    },
  });
  Object.defineProperty(TextNode.prototype, 'previousSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i > 0 ? sibs[i - 1] : null;
    },
  });

  function CommentNode(data) {
    this.nodeType = NODE_COMMENT;
    this.nodeName = '#comment';
    this.data = String(data == null ? '' : data);
    this.parentNode = null;
    this.ownerDocument = null;
  }
  Object.defineProperty(CommentNode.prototype, 'nextSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i >= 0 && i + 1 < sibs.length ? sibs[i + 1] : null;
    },
  });
  Object.defineProperty(CommentNode.prototype, 'previousSibling', {
    get: function () {
      if (!this.parentNode || !this.parentNode._children) return null;
      var sibs = this.parentNode._children;
      var i = sibs.indexOf(this);
      return i > 0 ? sibs[i - 1] : null;
    },
  });
  Object.defineProperty(CommentNode.prototype, 'textContent', {
    get: function () {
      return '';
    },
    set: function () {},
  });

  function Document() {
    this.nodeType = NODE_DOCUMENT;
    this.nodeName = '#document';
    this.__choysumMinimalDOM = true;
    this.activeElement = null;
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
    this.cancelBubble = false;
    this._stopImmediate = false;
    this.target = null;
    this.currentTarget = null;
  }
  Event.prototype.preventDefault = function () {
    this.defaultPrevented = true;
  };
  Event.prototype.stopPropagation = function () {
    this.cancelBubble = true;
  };
  Event.prototype.stopImmediatePropagation = function () {
    this.cancelBubble = true;
    this._stopImmediate = true;
  };

  var doc = new Document();
  global.document = doc;
  global.window = global;
  global.self = global;
  global.Node = Node;
  global.Element = Element;
  global.HTMLElement = Element;
  global.HTMLButtonElement = Element;
  global.HTMLInputElement = Element;
  global.SVGElement = Element;
  global.Text = TextNode;
  global.Comment = CommentNode;
  global.Document = Document;
  global.Event = Event;
  global.CustomEvent = Event;
  global.navigator = global.navigator || { userAgent: 'choysum-minimal-dom' };
  global.location = global.location || { href: 'http://localhost/', protocol: 'http:' };
  if (typeof global.innerHeight !== 'number') global.innerHeight = 768;
  if (typeof global.innerWidth !== 'number') global.innerWidth = 1024;
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
