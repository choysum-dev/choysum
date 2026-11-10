// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogOpProgressStrip.vue renders locked overlay with spinner and text', async () => {
  const wrapper = mount(ModuleCatalogOpProgressStrip as any, {
    props: {
      busy: true,
      title: 'Installing partner',
      headline: 'Working on it, please wait…',
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  const lock = document.body.querySelector('[data-testid="module-op-progress-strip"]');
  expect(lock != null).toBe(true);
  expect(lock?.getAttribute('aria-modal')).toBe('true');
  expect(lock?.getAttribute('tabindex')).toBe('-1');
  const spinner = document.body.querySelector('[data-testid="choy-spinner"]');
  expect(spinner != null).toBe(true);
  expect(String(spinner?.getAttribute('class') || '')).toContain('text-primary');
  expect(document.body.querySelector('[data-testid="module-op-progress-bar"]')).toBeNull();
  expect(
    document.body.querySelector('[data-testid="module-op-progress-headline"]')?.textContent || '',
  ).toContain('Working on it');
  const panel = document.body.querySelector('[data-testid="module-op-progress-panel"]');
  expect(panel?.className || '').not.toContain('border');
  expect(panel?.className || '').not.toContain('shadow');
  wrapper.unmount();
});

test('ModuleCatalogOpProgressStrip.vue hides when not busy', async () => {
  const wrapper = mount(ModuleCatalogOpProgressStrip as any, {
    props: { busy: false, title: 'x', headline: 'y' },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  expect(document.body.querySelector('[data-testid="module-op-progress-strip"]')).toBeNull();
  wrapper.unmount();
});
