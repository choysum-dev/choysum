// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@choysum/test-utils';
import {
  flushLoadingDisplayForTests,
  loadingRegistryOnRpcFinish,
  loadingRegistryOnRpcStart,
  resetLoadingRegistryForTests,
} from '@/core/web/client/loadingRegistry';
import ChoyLoadingSlowHint from './ChoyLoadingSlowHint.vue';

test('ChoyLoadingSlowHint mounts hidden then shows registry slow-hint text', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyLoadingSlowHint as any);
  await flushPromises();
  expect(document.querySelector('[data-anchor="choy.loading.slow-hint"]')).toBeNull();

  loadingRegistryOnRpcStart('slow-hint-mount');
  flushLoadingDisplayForTests();
  await flushPromises();
  const hint = document.querySelector('[data-anchor="choy.loading.slow-hint"]');
  expect(hint).toBeTruthy();
  expect(String(hint?.textContent || '')).toContain('Loading');
  expect(String(hint?.getAttribute('aria-label') || '')).toContain('Loading');

  loadingRegistryOnRpcFinish('slow-hint-mount');
  flushLoadingDisplayForTests();
  await flushPromises();
  wrapper.unmount();
  resetLoadingRegistryForTests();
});

test('ChoyLoadingSlowHint unmount clears timers while hint is visible', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyLoadingSlowHint as any);
  loadingRegistryOnRpcStart('slow-hint-unmount');
  flushLoadingDisplayForTests();
  await flushPromises();
  expect(document.querySelector('[data-anchor="choy.loading.slow-hint"]')).toBeTruthy();
  wrapper.unmount();
  loadingRegistryOnRpcFinish('slow-hint-unmount');
  flushLoadingDisplayForTests();
  resetLoadingRegistryForTests();
});
