// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `@codemirror/view`.
 * EditorView mounts a textarea so Json field tests can read/write without a real CM6.
 */

/** @type {null | { getDoc: () => string, setDoc: (s: string) => void, destroy: () => void, view: any }} */
export let __feStubLastJsonEditor = null;

function collectExt(extensions, key) {
  var out = [];
  var list = Array.isArray(extensions) ? extensions : [];
  for (var i = 0; i < list.length; i++) {
    var ext = list[i];
    if (!ext) continue;
    if (Array.isArray(ext)) {
      out = out.concat(collectExt(ext, key));
      continue;
    }
    if (ext[key]) out.push(ext[key]);
  }
  return out;
}

export class EditorView {
  /**
   * @param {{ parent?: HTMLElement, state?: any }} config
   */
  constructor(config) {
    var self = this;
    this._destroyed = false;
    this._updateFns = [];
    this._blurFns = [];
    var state = (config && config.state) || {};
    var initial =
      typeof state.__feStubDoc === 'string'
        ? state.__feStubDoc
        : state.doc && typeof state.doc.toString === 'function'
          ? state.doc.toString()
          : '';
    this._doc = initial == null ? '' : String(initial);

    var updateFns = collectExt(state.extensions, '__feStubUpdateListener');
    this._updateFns = updateFns;
    var handlersList = collectExt(state.extensions, '__feStubDomHandlers');
    for (var h = 0; h < handlersList.length; h++) {
      var handlers = handlersList[h];
      if (handlers && typeof handlers.blur === 'function') {
        this._blurFns.push(handlers.blur);
      }
    }
    var contentAttrsList = collectExt(state.extensions, '__feStubContentAttributes');
    var contentAttrs = {};
    for (var c = 0; c < contentAttrsList.length; c++) {
      var ca = contentAttrsList[c];
      if (ca && typeof ca === 'object') {
        Object.keys(ca).forEach(function (k) {
          contentAttrs[k] = ca[k];
        });
      }
    }

    var ta = document.createElement('textarea');
    ta.className = 'cm-fe-stub-input choy-json-input';
    ta.value = this._doc;
    ta.setAttribute('data-testid', 'choy-json-cm');
    Object.keys(contentAttrs).forEach(function (k) {
      if (contentAttrs[k] != null && contentAttrs[k] !== '') {
        ta.setAttribute(k, String(contentAttrs[k]));
      }
    });
    ta.addEventListener('input', function () {
      self._doc = ta.value;
      self._emitUpdate();
    });
    ta.addEventListener('blur', function (ev) {
      for (var i = 0; i < self._blurFns.length; i++) {
        self._blurFns[i](ev, self);
      }
    });
    this.dom = document.createElement('div');
    this.dom.className = 'cm-editor cm-fe-stub';
    this.dom.appendChild(ta);
    this.contentDOM = ta;
    this._ta = ta;

    if (config && config.parent) {
      config.parent.appendChild(this.dom);
    }

    this.state = {
      doc: {
        toString: function () {
          return self._doc;
        },
        get length() {
          return self._doc.length;
        },
      },
    };

    __feStubLastJsonEditor = {
      getDoc: function () {
        return self._doc;
      },
      setDoc: function (s) {
        self._replaceDoc(s);
      },
      destroy: function () {
        self.destroy();
      },
      view: self,
    };
  }

  _emitUpdate() {
    var update = {
      docChanged: true,
      state: this.state,
      view: this,
    };
    for (var i = 0; i < this._updateFns.length; i++) {
      this._updateFns[i](update);
    }
  }

  _replaceDoc(s) {
    this._doc = s == null ? '' : String(s);
    this._ta.value = this._doc;
  }

  dispatch(tr) {
    if (!tr || !tr.changes) return;
    var ch = tr.changes;
    if (typeof ch.insert === 'string' && ch.from === 0) {
      this._replaceDoc(ch.insert);
      this._emitUpdate();
      return;
    }
    if (ch.__feStubReplace != null) {
      this._replaceDoc(ch.__feStubReplace);
      this._emitUpdate();
    }
  }

  focus() {
    this._ta.focus();
  }

  destroy() {
    if (this._destroyed) return;
    this._destroyed = true;
    if (this.dom && this.dom.parentNode) this.dom.parentNode.removeChild(this.dom);
    if (__feStubLastJsonEditor && __feStubLastJsonEditor.view === this) {
      __feStubLastJsonEditor = null;
    }
  }
}

EditorView.updateListener = {
  of: function (fn) {
    return { __feStubUpdateListener: fn };
  },
};

EditorView.domEventHandlers = function (handlers) {
  return { __feStubDomHandlers: handlers || {} };
};

EditorView.theme = function () {
  return { __feStubExt: 'theme' };
};

EditorView.editable = {
  of: function () {
    return { __feStubExt: 'editable' };
  },
};

EditorView.contentAttributes = {
  of: function (attrs) {
    return { __feStubContentAttributes: attrs || {} };
  },
};
