// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { getConnectivityHooks, resetConnectivityHooksForTests } from '@/core/web/client/connectivityHooks';
import {
  TRANSPORT_OUTAGE_THRESHOLD,
  bindConnectivityLifecycle,
  isConnectivityUnreachable,
  unbindConnectivityLifecycleForTests,
  useConnectivityStore,
} from './index';

beforeEach(() => {
  setActivePinia(createPinia());
  unbindConnectivityLifecycleForTests();
  useConnectivityStore().resetForTests();
});

afterEach(() => {
  unbindConnectivityLifecycleForTests();
  resetConnectivityHooksForTests();
});

test('connectivityStore: reaches unreachable after threshold transport outages', () => {
  const store = useConnectivityStore();
  expect(store.status).toBe('reachable');
  for (let i = 0; i < TRANSPORT_OUTAGE_THRESHOLD - 1; i++) {
    store.noteTransportOutage();
    expect(store.status).toBe('reachable');
  }
  store.noteTransportOutage();
  expect(store.status).toBe('unreachable');
  expect(store.isUnreachable).toBe(true);
});

test('isConnectivityUnreachable mirrors store status and is false without Pinia', () => {
  expect(isConnectivityUnreachable()).toBe(false);
  useConnectivityStore().markUnreachable();
  expect(isConnectivityUnreachable()).toBe(true);
  setActivePinia(undefined as any);
  expect(isConnectivityUnreachable()).toBe(false);
  // Restore for describe afterEach (unbind / reset need an active Pinia).
  setActivePinia(createPinia());
});

test('connectivityStore: transport success clears streak and recovers', () => {
  const store = useConnectivityStore();
  store.noteTransportOutage();
  store.noteTransportOutage();
  expect(store.status).toBe('unreachable');
  const gen = store.recoveryGeneration;
  store.noteTransportSuccess();
  expect(store.status).toBe('reachable');
  expect(store.transportFailureStreak).toBe(0);
  expect(store.recoveryGeneration).toBe(gen + 1);
});

test('connectivityStore: probeConnectivity marks reachable on 2xx', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const ok = await store.probeConnectivity(async () => ({ ok: true } as Response));
  expect(ok).toBe(true);
  expect(store.status).toBe('reachable');
});

test('connectivityStore: probeConnectivity stays unreachable on failure', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const ok = await store.probeConnectivity(async () => {
    throw new Error('Failed to fetch');
  });
  expect(ok).toBe(false);
  expect(store.status).toBe('unreachable');
});

test('connectivityStore: navigator offline marks unreachable', () => {
  const store = useConnectivityStore();
  const listeners: Record<string, Array<() => void>> = {};
  const fakeWindow = {
    navigator: { onLine: true },
    addEventListener(type: string, fn: () => void) {
      (listeners[type] ||= []).push(fn);
    },
    removeEventListener(type: string, fn: () => void) {
      listeners[type] = (listeners[type] || []).filter(x => x !== fn);
    },
  } as unknown as Window;
  store.bindNavigator(fakeWindow);
  listeners.offline?.[0]?.();
  expect(store.status).toBe('unreachable');
  store.unbindNavigator(fakeWindow);
});

test('bindConnectivityLifecycle: hooks drive the store', () => {
  bindConnectivityLifecycle();
  const hooks = getConnectivityHooks();
  expect(typeof hooks.onTransportSuccess).toBe('function');
  expect(typeof hooks.onTransportOutage).toBe('function');
  const store = useConnectivityStore();
  hooks.onTransportOutage?.();
  hooks.onTransportOutage?.();
  expect(store.status).toBe('unreachable');
  hooks.onTransportSuccess?.();
  expect(store.status).toBe('reachable');
});
