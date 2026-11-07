// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, reactive } from 'vue';
import { useModuleCatalogSync } from './useModuleCatalogSync';

const msgs = {
  triggered: (id: string) => `ok:${id}`,
  triggeredEmpty: 'ok-empty',
  failed: (msg: string) => `fail:${msg}`,
};

test('useModuleCatalogSync: requestSync returns operation id from bridge response', async () => {
  const store: any = {
    state: { result: { rows: [{ Id: '1' }] } },
    RequestSync: async () => ({ Id: 'op-sync-1' }),
  };
  const api = useModuleCatalogSync(store, msgs);
  const id = await api.requestSync({ Force: true });
  expect(id).toBe('op-sync-1');
  expect(api.syncLoading.value).toBe(false);
});

test('useModuleCatalogSync: requestSync accepts string id and empty payload', async () => {
  const store: any = {
    RequestSync: async () => 'op-string',
  };
  const api = useModuleCatalogSync(store, msgs);
  expect(await api.requestSync()).toBe('op-string');

  store.RequestSync = async () => ({ Id: '' });
  expect(await api.requestSync({ Force: false, IfStale: true })).toBe('');
});

test('useModuleCatalogSync: requestSync returns empty string on failure and while loading', async () => {
  let resolveGate: (() => void) | undefined;
  const store: any = {
    RequestSync: () =>
      new Promise(resolve => {
        resolveGate = () => resolve({ Id: 'late' });
      }),
  };
  const api = useModuleCatalogSync(store, msgs);
  const pending = api.requestSync();
  expect(api.syncLoading.value).toBe(true);
  expect(await api.requestSync()).toBe('');
  resolveGate?.();
  expect(await pending).toBe('late');

  store.RequestSync = async () => {
    throw new Error('boom');
  };
  expect(await api.requestSync()).toBe('');

  store.RequestSync = async () => {
    throw 'raw';
  };
  expect(await api.requestSync()).toBe('');
});

test('useModuleCatalogSync: watchEmptyCatalogAutoSync fires once for empty loaded result', async () => {
  let syncCalls = 0;
  const state = reactive({ result: undefined as any, loading: false });
  const store: any = {
    state,
    RequestSync: async () => {
      syncCalls += 1;
      return { Id: `op-${syncCalls}` };
    },
  };
  const api = useModuleCatalogSync(store, msgs);
  api.watchEmptyCatalogAutoSync();

  state.result = null;
  await nextTick();
  expect(syncCalls).toBe(0);

  state.loading = true;
  state.result = { rows: [] };
  await nextTick();
  expect(syncCalls).toBe(0);

  state.loading = false;
  state.result = { rows: [] };
  await nextTick();
  await Promise.resolve();
  expect(syncCalls).toBe(1);

  state.result = { rows: [] };
  await nextTick();
  expect(syncCalls).toBe(1);

  state.result = { rows: [{ Id: '1' }] };
  await nextTick();
  state.result = { nope: true };
  await nextTick();
  expect(syncCalls).toBe(1);
});

test('useModuleCatalogSync: later empty search after non-empty first result does not force sync', async () => {
  let syncCalls = 0;
  const state = reactive({ result: undefined as any, loading: false });
  const store: any = {
    state,
    RequestSync: async () => {
      syncCalls += 1;
      return { Id: `op-${syncCalls}` };
    },
  };
  const api = useModuleCatalogSync(store, msgs);
  api.watchEmptyCatalogAutoSync();

  state.result = { rows: [{ Id: '1' }] };
  await nextTick();
  expect(syncCalls).toBe(0);

  // User search with no matches must not be treated as an empty catalog.
  state.result = { rows: [] };
  await nextTick();
  await Promise.resolve();
  expect(syncCalls).toBe(0);
});

test('useModuleCatalogSync: immediate watch syncs a pre-settled empty catalog once', async () => {
  let syncCalls = 0;
  const store: any = {
    state: { result: { rows: [] }, loading: false },
    RequestSync: async () => {
      syncCalls += 1;
      return { Id: `op-${syncCalls}` };
    },
  };
  const api = useModuleCatalogSync(store, msgs);
  api.watchEmptyCatalogAutoSync();
  await nextTick();
  await Promise.resolve();
  expect(syncCalls).toBe(1);
});
