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

test('OChatterFieldChangeItem smoke: mounts with create entry', async () => {
  const { default: Comp } = await import('./OChatterFieldChangeItem.vue');
  expect(Comp).toBeTruthy();

  const app = createApp(Comp, {
    authorLabel: 'Tester',
    entry: {
      kind: 'fieldChange',
      id: 'f1',
      at: Date.parse('2024-01-01T12:00:00.000Z'),
      field: null,
      changeKind: 'create',
      oldValue: null,
      newValue: null,
      actorUid: 'u1',
    },
  });
  for (const name of ['ElIcon', 'ElAvatar']) app.component(name, stub(name));

  const el = document.createElement('div');
  app.mount(el);
  expect(el.childNodes.length >= 0).toBe(true);
  expect(el.textContent).toContain('Tester');
  app.unmount();
});
