// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@choysum/test-utils';
import Xpath from './xpath.vue';

test('xpath mount: mounts empty Xpath and runs script', () => {
  const w = mount(Xpath);
  // choysumMount wrapper exposes element/vm/find/text/trigger/unmount (no wrapper.exists).
  expect(w.element).toBeTruthy();
  expect(w.vm).toBeTruthy();
  w.unmount();
});
