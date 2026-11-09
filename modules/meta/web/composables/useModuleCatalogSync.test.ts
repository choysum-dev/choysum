// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useModuleCatalogSync } from './useModuleCatalogSync';

const msgs = {
  triggered: (id: string) => `triggered:${id}`,
  failed: (detail: string) => `failed:${detail}`,
};

test('useModuleCatalogSync: requestSync returns operation id from bridge response', async () => {
  const store = {
    RequestSync: async () => ({ Id: 'op-sync-1' }),
  } as any;
  const api = useModuleCatalogSync(store, msgs);
  const id = await api.requestSync();
  expect(id).toBe('op-sync-1');
});

test('useModuleCatalogSync: requestSync accepts string id and empty payload', async () => {
  const store = {
    RequestSync: async () => 'op-string',
  } as any;
  const api = useModuleCatalogSync(store, msgs);
  expect(await api.requestSync()).toBe('op-string');

  store.RequestSync = async () => ({ Id: '' });
  expect(await api.requestSync()).toBe('');
});

test('useModuleCatalogSync: requestSync returns empty string on failure and while loading', async () => {
  let resolvePending: ((value: { Id: string }) => void) | null = null;
  const store = {
    RequestSync: () =>
      new Promise<{ Id: string }>(resolve => {
        resolvePending = resolve;
      }),
  } as any;
  const api = useModuleCatalogSync(store, msgs);
  const pending = api.requestSync();
  expect(api.syncLoading.value).toBe(true);
  expect(await api.requestSync()).toBe('');
  resolvePending!({ Id: 'op-done' });
  expect(await pending).toBe('op-done');

  store.RequestSync = async () => {
    throw new Error('boom');
  };
  expect(await api.requestSync()).toBe('');

  store.RequestSync = async () => {
    throw 'string-err';
  };
  expect(await api.requestSync()).toBe('');
});

test('useModuleCatalogSync: empty operation id does not treat as success path for callers', async () => {
  const store = {
    RequestSync: async () => ({ Id: '' }),
  } as any;
  const api = useModuleCatalogSync(store, msgs);
  expect(await api.requestSync()).toBe('');
});
