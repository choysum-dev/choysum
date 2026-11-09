// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useModuleCatalogSync } from './useModuleCatalogSync';

const msgs = {
  failed: (detail: string) => `failed:${detail}`,
  settledOk: (summary: string) => (summary ? `ok:${summary}` : 'ok'),
  settledPartial: (summary: string) => `partial:${summary}`,
  settledFailed: (message: string) => `err:${message}`,
  jobStillRunning: 'still-running',
  serviceRestarting: 'restarting',
  failedToGetStatus: 'no-status',
  formatSummaryOk: (sync: { total: number; catalogRows: number }) =>
    `synced ${sync.total} / catalog ${sync.catalogRows}`,
  formatSummaryPartial: (sync: { success: number; total: number; failed: number; catalogRows: number }) =>
    `synced ${sync.success}/${sync.total} (${sync.failed} failed) / catalog ${sync.catalogRows}`,
};

test('useModuleCatalogSync: onSync watches operation and settles with summary toast path', async () => {
  let refreshed = 0;
  const catalogStore = {
    RequestSync: async () => ({ Id: 'op-sync-1' }),
  } as any;
  const operationStore = {
    Browse: async () => ({
      Action: 'sync',
      Status: 'succeeded',
      PlanJson: JSON.stringify({
        syncResult: { total: 2, success: 2, failed: 0, catalogRows: 5 },
      }),
    }),
  } as any;
  const api = useModuleCatalogSync(catalogStore, operationStore, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await api.onSync();
  expect(refreshed).toBe(1);
  expect(api.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: onSync ignores second click while busy', async () => {
  let calls = 0;
  let resolveBrowse: ((row: any) => void) | null = null;
  const catalogStore = {
    RequestSync: async () => {
      calls += 1;
      return { Id: 'op-busy' };
    },
  } as any;
  const operationStore = {
    Browse: () =>
      new Promise(resolve => {
        resolveBrowse = resolve;
      }),
  } as any;
  const api = useModuleCatalogSync(catalogStore, operationStore, msgs);
  const pending = api.onSync();
  await Promise.resolve();
  expect(api.syncBusy.value).toBe(true);
  await api.onSync();
  expect(calls).toBe(1);
  resolveBrowse!({ Action: 'sync', Status: 'succeeded', PlanJson: '{}' });
  await pending;
  expect(api.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: enqueue failure surfaces failed toast path without watch', async () => {
  const catalogStore = {
    RequestSync: async () => {
      throw new Error('boom');
    },
  } as any;
  const api = useModuleCatalogSync(catalogStore, null, msgs);
  await api.onSync();
  expect(api.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: empty operation id does not start watch', async () => {
  let browsed = 0;
  const catalogStore = {
    RequestSync: async () => ({ Id: '' }),
  } as any;
  const operationStore = {
    Browse: async () => {
      browsed += 1;
      return { Status: 'succeeded' };
    },
  } as any;
  const api = useModuleCatalogSync(catalogStore, operationStore, msgs);
  await api.onSync();
  expect(browsed).toBe(0);
});

test('useModuleCatalogSync: missing operation store refreshes without Browse', async () => {
  let refreshed = 0;
  const catalogStore = {
    RequestSync: async () => ({ Id: 'op-no-op-store' }),
  } as any;
  const api = useModuleCatalogSync(catalogStore, null, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await api.onSync();
  expect(refreshed).toBe(1);
});
