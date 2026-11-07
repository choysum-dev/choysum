// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useModuleCatalogSync } from './useModuleCatalogSync';

test('useModuleCatalogSync: requestSync returns operation id from bridge response', async () => {
  const store: any = {
    state: { result: { rows: [{ Id: '1' }] } },
    RequestSync: async () => ({ Id: 'op-sync-1' }),
  };
  const api = useModuleCatalogSync(store, {
    triggered: id => `ok:${id}`,
    triggeredEmpty: 'ok',
    failed: msg => `fail:${msg}`,
  });
  const id = await api.requestSync({ Force: true });
  expect(id).toBe('op-sync-1');
  expect(api.syncLoading.value).toBe(false);
});

test('useModuleCatalogSync: requestSync returns empty string on failure', async () => {
  const store: any = {
    RequestSync: async () => {
      throw new Error('boom');
    },
  };
  const api = useModuleCatalogSync(store, {
    triggered: id => id,
    triggeredEmpty: 'ok',
    failed: msg => msg,
  });
  const id = await api.requestSync();
  expect(id).toBe('');
});
