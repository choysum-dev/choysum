// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { h, ref } from 'vue';
import { mount, flushPromises } from '@choysum/test-utils';

const AsyncProbe = {
  name: 'AsyncProbe',
  setup() {
    const label = ref('pending');
    Promise.resolve().then(() => {
      label.value = 'done';
    });
    return () => h('div', { class: 'async-label' }, label.value);
  },
};

const w = mount(AsyncProbe);
(globalThis as any).__hostResult = { before: w.find('.async-label').text() };

flushPromises().then(() => {
  (globalThis as any).__hostResult.after = w.find('.async-label').text();
  (globalThis as any).__hostResult.ready = true;
  w.unmount();
});
