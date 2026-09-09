// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `vue-echarts`. */
import { defineComponent, h } from 'vue';

export default defineComponent({
  name: 'VChartStub',
  setup: function () {
    return function () {
      return h('div', { class: 'fe-stub-vchart' });
    };
  },
});
