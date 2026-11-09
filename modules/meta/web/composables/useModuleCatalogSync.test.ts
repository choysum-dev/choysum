// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope, ref } from 'vue';
import { ChoyMessage } from '@/web';
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

test('useModuleCatalogSync: missing operation store warns without refresh or Browse', async () => {
  let refreshed = 0;
  const warnings: string[] = [];
  const orig = ChoyMessage.warning;
  ChoyMessage.warning = (m: string) => {
    warnings.push(m);
  };
  const catalogStore = {
    RequestSync: async () => ({ Id: 'op-no-op-store' }),
  } as any;
  const api = useModuleCatalogSync(catalogStore, null, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await api.onSync();
  expect(refreshed).toBe(0);
  expect(warnings).toEqual([msgs.jobStillRunning]);
  ChoyMessage.warning = orig;
});

test('useModuleCatalogSync: partial and failed terminals refresh and clear busy', async () => {
  let refreshed = 0;
  const catalogStore = { RequestSync: async () => ({ Id: 'op-partial' }) } as any;
  const partialStore = {
    Browse: async () => ({
      Action: 'sync',
      Status: 'succeeded',
      PlanJson: JSON.stringify({
        syncResult: { total: 3, success: 2, failed: 1, catalogRows: 8 },
      }),
    }),
  } as any;
  const partial = useModuleCatalogSync(catalogStore, partialStore, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await partial.onSync();
  expect(refreshed).toBe(1);
  expect(partial.syncBusy.value).toBe(false);

  const failedStore = {
    Browse: async () => ({
      Action: 'sync',
      Status: 'failed',
      ErrorMessage: 'sync blew up',
    }),
  } as any;
  const failed = useModuleCatalogSync(catalogStore, failedStore, msgs);
  await failed.onSync();
  expect(failed.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: stop resets watch state', async () => {
  let resolveBrowse: ((row: any) => void) | undefined;
  const catalogStore = { RequestSync: async () => ({ Id: 'op-stop' }) } as any;
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
  api.stop();
  expect(api.syncBusy.value).toBe(false);
  resolveBrowse?.({ Action: 'sync', Status: 'succeeded', PlanJson: '{}' });
  await pending;
});

test('useModuleCatalogSync: ok path without custom formatters uses formatModuleOpSummary', async () => {
  const catalogStore = { RequestSync: async () => ({ Id: 'op-plain' }) } as any;
  const operationStore = {
    Browse: async () => ({
      Action: 'install',
      Status: 'succeeded',
      PlanJson: '{}',
    }),
  } as any;
  const api = useModuleCatalogSync(catalogStore, operationStore, {
    ...msgs,
    formatSummaryOk: undefined,
    formatSummaryPartial: undefined,
  });
  await api.onSync();
  expect(api.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: unwraps nested operation store refs', async () => {
  const catalogStore = { RequestSync: async () => ({ Id: 'op-nested' }) } as any;
  const inner = {
    Browse: async () => ({
      Action: 'sync',
      Status: 'succeeded',
      PlanJson: JSON.stringify({
        syncResult: { total: 1, success: 1, failed: 0, catalogRows: 2 },
      }),
    }),
  };
  const api = useModuleCatalogSync(catalogStore, ref(ref(inner)) as any, msgs);
  await api.onSync();
  expect(api.syncBusy.value).toBe(false);
});

test('useModuleCatalogSync: scope dispose stops an in-flight watch', async () => {
  let resolveBrowse: ((row: any) => void) | undefined;
  const catalogStore = { RequestSync: async () => ({ Id: 'op-scope' }) } as any;
  const operationStore = {
    Browse: () =>
      new Promise(resolve => {
        resolveBrowse = resolve;
      }),
  } as any;
  let api: ReturnType<typeof useModuleCatalogSync> | undefined;
  const scope = effectScope();
  scope.run(() => {
    api = useModuleCatalogSync(catalogStore, operationStore, msgs);
    void api.onSync();
  });
  await Promise.resolve();
  expect(api!.syncBusy.value).toBe(true);
  scope.stop();
  expect(api!.syncBusy.value).toBe(false);
  resolveBrowse?.({ Action: 'sync', Status: 'succeeded', PlanJson: '{}' });
});
