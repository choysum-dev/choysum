// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';
import {
  beginNavigationLoading,
  getLoadingRegistryDebugState,
  resetLoadingRegistryForTests,
  runWithLoading,
} from '@/core/web/client/loadingRegistry';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  settleNavChromeIfNoSurface,
  useSettleNavChromeOnActivate,
} from './useSettleNavChromeOnActivate';

test('settleNavChromeIfNoSurface drops an idle navigation ticket', () => {
  resetLoadingRegistryForTests();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  settleNavChromeIfNoSurface();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  resetLoadingRegistryForTests();
});

test('settleNavChromeIfNoSurface keeps the ticket while a surface is busy', async () => {
  resetLoadingRegistryForTests();
  beginNavigationLoading();
  await runWithLoading({ layer: 'surface', target: 'keep-alive-land', mode: 'block' }, async () => {
    settleNavChromeIfNoSurface();
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  });
  resetLoadingRegistryForTests();
});

test('useSettleNavChromeOnActivate settles leftover nav chrome on KeepAlive reactivate', async () => {
  resetLoadingRegistryForTests();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);

  const showCached = ref(true);
  const CachedPage = defineComponent({
    name: 'SettleActivatePage',
    setup() {
      useSettleNavChromeOnActivate();
      return () => h('div', { 'data-test': 'cached' }, 'ok');
    },
  });
  const Host = defineComponent({
    name: 'SettleActivateHost',
    setup() {
      return () =>
        h(KeepAlive, null, {
          default: () => (showCached.value ? h(CachedPage) : h('div', { 'data-test': 'other' }, 'other')),
        });
    },
  });
  const mounted = mountApp(Host as any);
  await flushPromises();
  await nextTick();
  // First activate (mount into KeepAlive) settles idle nav chrome.
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);

  showCached.value = false;
  await nextTick();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);

  showCached.value = true;
  await flushPromises();
  await nextTick();
  await nextTick();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);

  resetLoadingRegistryForTests();
  mounted.unmount();
});
