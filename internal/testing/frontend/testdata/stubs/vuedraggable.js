// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `vuedraggable`. */
import { defineComponent, h } from 'vue';

export default defineComponent({
  name: 'DraggableStub',
  setup: function (_props, ctx) {
    return function () {
      return h('div', { class: 'fe-stub-draggable' }, ctx.slots.default ? ctx.slots.default() : []);
    };
  },
});
