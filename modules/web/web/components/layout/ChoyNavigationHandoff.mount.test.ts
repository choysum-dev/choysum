// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@choysum/test-utils';
import {
  beginNavigationLoading,
  resetLoadingRegistryForTests,
  runWithLoading,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';
import ChoyNavigationHandoff from './ChoyNavigationHandoff.vue';

test('ChoyNavigationHandoff shows during nav ticket and clears on surface or settle', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyNavigationHandoff as any);
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();

  beginNavigationLoading();
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeTruthy();
  expect(String(document.body.textContent || '')).toContain('Loading');

  await runWithLoading({ layer: 'surface', target: 'handoff-land', mode: 'block' }, async () => {
    await flushPromises();
    expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();
  });

  beginNavigationLoading();
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeTruthy();
  settleNavigationIfActive();
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();

  wrapper.unmount();
  resetLoadingRegistryForTests();
});
