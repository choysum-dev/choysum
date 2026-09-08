// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** Passthrough slot stub used by FE unit Element Plus / icon aliases. */
import { h, defineComponent } from 'vue';

export function makeSlotStub(name) {
  return defineComponent({
    name: String(name || 'Stub'),
    inheritAttrs: false,
    setup(_props, ctx) {
      return function () {
        var children = [];
        if (ctx.slots && typeof ctx.slots.default === 'function') {
          children = children.concat(ctx.slots.default());
        }
        var stubClass = 'fe-stub-' + String(name || 'x');
        var attrs = Object.assign({}, ctx.attrs || {});
        var mergedClass = attrs.class ? stubClass + ' ' + attrs.class : stubClass;
        attrs.class = mergedClass;
        return h('div', attrs, children);
      };
    },
  });
}

export function makeIconStub(name) {
  return defineComponent({
    name: String(name || 'Icon'),
    setup: function () {
      return function () {
        return h('span', { class: 'fe-stub-icon', 'data-icon': String(name || '') });
      };
    },
  });
}
