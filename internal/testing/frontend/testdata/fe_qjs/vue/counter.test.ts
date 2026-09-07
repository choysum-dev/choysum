// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { mount } from '@choysum/test-utils';
import Counter from './Counter.vue';

test('mounts Counter and exposes script marker', () => {
  const w = mount(Counter);
  expect(w.find('.fe-qjs-counter').exists()).toBe(true);
  expect(w.find('.fe-qjs-counter').text()).toBe('fe-qjs-7');
  w.unmount();
});
