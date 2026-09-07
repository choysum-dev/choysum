// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { h } from 'vue';
import { shallowMount } from '@choysum/test-utils';
import HostWithChild from './HostWithChild.vue';

const w = shallowMount(HostWithChild, {
  stubs: {
    ChildWidget: {
      name: 'ChildWidget',
      setup() {
        return () => h('div', { class: 'stub-child' }, 'stubbed');
      },
    },
  },
});

(globalThis as any).__hostResult = {
  label: w.find('.label').text(),
  hasStub: w.find('.stub-child').exists(),
  hasRealChild: w.find('.real-child').exists(),
  ready: true,
};
