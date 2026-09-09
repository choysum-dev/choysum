// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp } from 'vue';
import HelloFixture from './fixtures/HelloFixture.vue';

test('Vue SFC smoke: mounts a .vue component and renders props', () => {
  const el = document.createElement('div');
  const app = createApp(HelloFixture, { name: 'Choysum' });
  app.mount(el);
  expect(el.textContent).toContain('Hello Choysum');
  app.unmount();
});
