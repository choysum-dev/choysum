// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `@tiptap/vue-3`.
 * Mimics TipTap's onMounted editor creation so OHtmlCommitBridge can sync store ↔ HTML.
 */
import { defineComponent, h, onBeforeUnmount, shallowRef } from 'vue';

/** @type {null | Record<string, any>} */
export let __feStubLastEditor = null;
/** @type {null | { value: any }} */
export let __feStubEditorRef = null;

function createEditor(options) {
  var html = options && options.content != null ? String(options.content) : '';
  var listeners = Object.create(null);
  var active = Object.create(null);
  var attrs = Object.create(null);
  var dom = {
    tagName: 'DIV',
    attributes: Object.create(null),
    setAttribute: function (name, value) {
      this.attributes[name] = String(value);
    },
    removeAttribute: function (name) {
      delete this.attributes[name];
    },
    getAttribute: function (name) {
      return Object.prototype.hasOwnProperty.call(this.attributes, name)
        ? this.attributes[name]
        : null;
    },
  };
  var initialAttrs =
    options && options.editorProps && options.editorProps.attributes
      ? options.editorProps.attributes
      : {};
  Object.keys(initialAttrs || {}).forEach(function (key) {
    dom.setAttribute(key, initialAttrs[key]);
  });

  function emit(event) {
    var list = listeners[event] || [];
    for (var i = 0; i < list.length; i++) list[i]();
  }

  var chainApi = {
    focus: function () {
      return chainApi;
    },
    toggleBold: function () {
      active.bold = !active.bold;
      return chainApi;
    },
    toggleItalic: function () {
      active.italic = !active.italic;
      return chainApi;
    },
    toggleBulletList: function () {
      active.bulletList = !active.bulletList;
      return chainApi;
    },
    toggleOrderedList: function () {
      active.orderedList = !active.orderedList;
      return chainApi;
    },
    unsetLink: function () {
      active.link = false;
      attrs.link = {};
      return chainApi;
    },
    extendMarkRange: function () {
      return chainApi;
    },
    setLink: function (next) {
      active.link = true;
      attrs.link = next || {};
      return chainApi;
    },
    run: function () {
      emit('update');
      return true;
    },
  };

  var editor = {
    view: { dom: dom },
    getHTML: function () {
      return html;
    },
    isActive: function (name) {
      return !!active[name];
    },
    getAttributes: function (name) {
      return attrs[name] || {};
    },
    chain: function () {
      return chainApi;
    },
    commands: {
      setContent: function (content, emitUpdate) {
        html = content == null ? '' : String(content);
        if (emitUpdate) emit('update');
        return true;
      },
    },
    on: function (event, fn) {
      if (!listeners[event]) listeners[event] = [];
      listeners[event].push(fn);
    },
    off: function (event, fn) {
      listeners[event] = (listeners[event] || []).filter(function (f) {
        return f !== fn;
      });
    },
    destroy: function () {
      listeners = Object.create(null);
    },
    /** Test helper: set HTML and fire TipTap update. */
    __feStubSetHTML: function (next, emitUpdate) {
      html = next == null ? '' : String(next);
      if (emitUpdate !== false) emit('update');
    },
  };
  return editor;
}

export function useEditor(options) {
  // Create synchronously so v-if="editor" toolbar renders on first paint.
  // Real TipTap defers to onMounted; OHtmlCommitBridge already handles both.
  var ed = createEditor(options || {});
  if (options && typeof options.onUpdate === 'function') {
    ed.on('update', function () {
      options.onUpdate({ editor: ed });
    });
  }
  __feStubLastEditor = ed;
  var editorRef = shallowRef(ed);
  __feStubEditorRef = editorRef;
  if (options && typeof options.onCreate === 'function') {
    options.onCreate({ editor: ed });
  }
  onBeforeUnmount(function () {
    if (editorRef.value) editorRef.value.destroy();
    if (__feStubLastEditor === editorRef.value) __feStubLastEditor = null;
    if (__feStubEditorRef === editorRef) __feStubEditorRef = null;
    editorRef.value = null;
  });
  return editorRef;
}

export var EditorContent = defineComponent({
  name: 'EditorContent',
  props: {
    editor: { default: null },
  },
  setup: function (props) {
    return function () {
      var ed = props.editor;
      var data = {
        class: 'ProseMirror fe-stub-editor-content',
        contenteditable: 'true',
        ref: function (el) {
          if (!ed || !ed.view || !el) return;
          var prev = ed.view.dom;
          // Stub mock stores attrs in a plain object; real Element uses NamedNodeMap.
          var prevAttrs = prev && prev.nodeType !== 1 && prev.attributes ? prev.attributes : null;
          if (prev && prev.nodeType === 1 && prev !== el) {
            ['id', 'name', 'aria-invalid', 'aria-required', 'aria-describedby'].forEach(function (key) {
              var v = prev.getAttribute(key);
              if (v) el.setAttribute(key, v);
            });
          }
          ed.view.dom = el;
          if (prevAttrs) {
            Object.keys(prevAttrs).forEach(function (key) {
              el.setAttribute(key, prevAttrs[key]);
            });
          }
        },
      };
      var dom = ed && ed.view ? ed.view.dom : null;
      if (dom && dom.nodeType === 1) {
        ['id', 'name', 'aria-invalid', 'aria-required', 'aria-describedby'].forEach(function (key) {
          var v = dom.getAttribute(key);
          if (v) data[key] = v;
        });
      } else if (dom && dom.attributes) {
        var stubAttrs = dom.attributes;
        if (stubAttrs.id) data.id = stubAttrs.id;
        if (stubAttrs.name) data.name = stubAttrs.name;
        if (stubAttrs['aria-invalid']) data['aria-invalid'] = stubAttrs['aria-invalid'];
        if (stubAttrs['aria-required']) data['aria-required'] = stubAttrs['aria-required'];
        if (stubAttrs['aria-describedby']) data['aria-describedby'] = stubAttrs['aria-describedby'];
      }
      return h('div', data);
    };
  },
});
