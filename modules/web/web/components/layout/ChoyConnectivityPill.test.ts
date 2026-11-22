// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { mountApp } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '../../stores/connectivityStore';
import ChoyConnectivityPill from './ChoyConnectivityPill.vue';

beforeEach(() => {
  setActivePinia(createPinia());
  useConnectivityStore().resetForTests();
});

test('ChoyConnectivityPill: hidden while reachable', () => {
  const mounted = mountApp(ChoyConnectivityPill as any);
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  mounted.unmount();
});

test('ChoyConnectivityPill: shows when unreachable and hides after recover', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const mounted = mountApp(ChoyConnectivityPill as any);
  await nextTick();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeTruthy();
  store.markReachable();
  await nextTick();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  mounted.unmount();
});
