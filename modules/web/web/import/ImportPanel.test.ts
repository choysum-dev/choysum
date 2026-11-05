// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp } from 'vue';
import { flushPromises } from '@/web/web/__tests__/mountApp';

test('ImportPanel smoke: mounts with minimal props', async () => {
  const { default: Comp } = await import('./ImportPanel.vue');
  expect(Comp).toBeTruthy();

  const app = createApp(Comp, {
    model: 'partner.Partner',
    companyId: 'c1',
    modelValue: true,
  });

  const el = document.createElement('div');
  document.body.appendChild(el);
  app.mount(el);
  await flushPromises();
  const content =
    el.querySelector('[data-slot="dialog-content"]') ||
    document.body.querySelector('[data-slot="dialog-content"]');
  expect(content).not.toBeNull();
  expect(content!.querySelector('[data-slot="dialog-footer"]')).not.toBeNull();
  expect((content!.textContent || '').includes('Import')).toBe(true);
  const stepper = content!.querySelector('[data-anchor="choy.stepper"]');
  expect(stepper).not.toBeNull();
  expect(stepper!.getAttribute('data-readonly')).toBe('true');
  expect(stepper!.querySelector('[data-step="upload"]')!.getAttribute('data-state')).toBe('active');
  for (const value of ['upload', 'preview', 'import']) {
    expect(stepper!.querySelector(`[data-step="${value}"]`)!.hasAttribute('data-disabled')).toBe(true);
  }
  expect(content!.querySelector('.import-panel-steps')).toBeNull();
  app.unmount();
  el.remove();
});
