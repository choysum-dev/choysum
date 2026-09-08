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

test('OPage smoke: mounts with title text and region role', async () => {
  const { default: Comp } = await import('./OPage.vue');
  expect(Comp).toBeTruthy();

  const app = createApp(Comp, {
    title: 'Test Title',
    showBreadcrumb: false,
  });
  for (const name of ['OBreadcrumb', 'OPageIoMenu', 'ElIcon', 'Loading']) {
    app.component(name, stub(name));
  }

  const el = document.createElement('div');
  app.mount(el);

  const titleEl = el.querySelector('h1.o-page__title');
  expect(titleEl).toBeTruthy();
  expect(titleEl?.textContent).toBe('Test Title');

  const region = el.querySelector('.o-page');
  expect(region?.getAttribute('role')).toBe('region');
  expect(region?.getAttribute('aria-labelledby')).toBeTruthy();

  app.unmount();
});
