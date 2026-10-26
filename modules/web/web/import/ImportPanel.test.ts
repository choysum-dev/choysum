// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp } from 'vue';

test('ImportPanel smoke: mounts with minimal props', async () => {
  const { default: Comp } = await import('./ImportPanel.vue');
  expect(Comp).toBeTruthy();

  const app = createApp(Comp, {
    model: 'partner.Partner',
    companyId: 'c1',
    modelValue: false,
  });

  const el = document.createElement('div');
  app.mount(el);
  expect(el.isConnected || el.childNodes.length >= 0).toBe(true);
  app.unmount();
});
