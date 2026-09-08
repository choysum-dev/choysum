// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, h, type Component } from 'vue';

function stub(name: string): Component {
  return {
    name,
    setup(_, { slots }) {
      return () => h('div', { 'data-stub': name }, slots.default?.());
    },
  };
}

test('OChatterMessageItem smoke: mounts and shows author/body', async () => {
  const { default: Comp } = await import('./OChatterMessageItem.vue');
  expect(Comp).toBeTruthy();

  const at = Date.parse('2024-01-01T12:00:00.000Z');
  const app = createApp(Comp, {
    authorLabel: 'Tester',
    entry: {
      kind: 'message',
      id: 'm1',
      at,
      type: 'comment',
      body: 'hello world',
      authorUid: 'u1',
    },
  });
  for (const name of ['ElAvatar', 'ElIcon']) app.component(name, stub(name));

  const el = document.createElement('div');
  app.mount(el);
  expect(el.textContent).toContain('Tester');
  expect(el.textContent).toContain('hello world');
  app.unmount();
});
