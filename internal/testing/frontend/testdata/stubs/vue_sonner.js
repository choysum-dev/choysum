// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `vue-sonner`.
 * Real Toaster pulls browser/DOM APIs that are unreliable in QuickJS.
 */
import { defineComponent, h } from 'vue';

const entries = [];
let nextId = 1;

function push(level, message, data) {
  const id = nextId++;
  const title = typeof message === 'string' ? message : String(message?.title ?? message ?? '');
  entries.push({
    id,
    level,
    title,
    description: data?.description,
    duration: data?.duration,
  });
  return id;
}

export const toast = Object.assign(
  (message, data) => push('default', message, data),
  {
    success: (message, data) => push('success', message, data),
    error: (message, data) => push('error', message, data),
    warning: (message, data) => push('warning', message, data),
    info: (message, data) => push('info', message, data),
    dismiss: (id) => {
      if (id == null) {
        entries.length = 0;
        return;
      }
      const idx = entries.findIndex((e) => e.id === id);
      if (idx >= 0) entries.splice(idx, 1);
    },
    /** Test helper: inspect pending toasts. */
    _entries: entries,
    _clear: () => {
      entries.length = 0;
    },
  },
);

export const Toaster = defineComponent({
  name: 'SonnerToaster',
  inheritAttrs: false,
  setup(_, { attrs, slots }) {
    return () =>
      h(
        'div',
        { 'data-sonner-stub': 'Toaster', ...attrs },
        slots.default ? slots.default() : null,
      );
  },
});

export default Toaster;
