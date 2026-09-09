// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `@tiptap/vue-3`.
 * Mimics TipTap's onMounted editor creation so OHtmlCommitBridge can sync store ↔ HTML.
 */
import { defineComponent, h, onBeforeUnmount, shallowRef } from 'vue';

/** @type {null | Record<string, any>} */
export let __feStubLastEditor = null;

function createEditor(options) {
  var html = options && options.content != null ? String(options.content) : '';
  var listeners = Object.create(null);
  var active = Object.create(null);
  var attrs = Object.create(null);

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
      return true;
    },
  };

  var editor = {
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
  __feStubLastEditor = ed;
  var editorRef = shallowRef(ed);
  onBeforeUnmount(function () {
    if (editorRef.value) editorRef.value.destroy();
    if (__feStubLastEditor === editorRef.value) __feStubLastEditor = null;
    editorRef.value = null;
  });
  return editorRef;
}

export var EditorContent = defineComponent({
  name: 'EditorContent',
  props: {
    editor: { default: null },
  },
  setup: function () {
    return function () {
      return h('div', { class: 'ProseMirror fe-stub-editor-content', contenteditable: 'true' });
    };
  },
});
