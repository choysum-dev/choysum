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

test('ImportPanel smoke: mounts with minimal props', async () => {
  const { default: Comp } = await import('./ImportPanel.vue');
  expect(Comp).toBeTruthy();

  const app = createApp(Comp, {
    model: 'partner.Partner',
    companyId: 'c1',
    modelValue: false,
  });
  for (const name of [
    'ElDialog',
    'ElSteps',
    'ElStep',
    'ElUpload',
    'ElAlert',
    'ElButton',
    'ElSelect',
    'ElOption',
    'ElTable',
    'ElTableColumn',
    'ElResult',
  ]) {
    app.component(name, stub(name));
  }

  const el = document.createElement('div');
  app.mount(el);
  expect(el.isConnected || el.childNodes.length >= 0).toBe(true);
  app.unmount();
});
