// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
import { useModuleCatalogOpProgress } from './useModuleCatalogOpProgress';

const msgs = {
  jobStillRunning: 'still-running',
  serviceRestarting: 'restarting',
  failedToGetStatus: 'no-status',
  settledOk: 'Operation Succeeded',
  settledFailed: (message: string) => `Operation Failed: ${message}`,
  reloadingService: () => 'Almost done, refreshing…',
  executing: () => 'Working on it, please wait…',
  queued: () => 'Getting ready…',
  finishing: () => 'Wrapping up…',
  inProgress: () => 'Please wait…',
  titleFor: (action: string, moduleName: string) =>
    moduleName ? `${action || 'Working on'} ${moduleName}` : action || 'Working…',
};

test('useModuleCatalogOpProgress: watches to terminal and clears busy', async () => {
  const refreshes: number[] = [];
  const store = {
    Browse: async () => ({
      Status: 'succeeded',
      Phase: 'done',
      ReloadTriggered: false,
      ReloadFailed: false,
    }),
  } as any;
  const api = useModuleCatalogOpProgress(store, msgs, {
    refreshCatalog: async () => {
      refreshes.push(1);
    },
  });

  const done = api.startOpProgress('op-1', { moduleName: 'partner', action: 'install' });
  expect(api.progressBusy.value).toBe(true);
  expect(api.progressHeadline.value).toContain('Getting ready');
  expect(api.progressTitle.value).toContain('partner');
  await done;
  expect(api.progressBusy.value).toBe(false);
  expect(refreshes.length).toBe(1);
});

test('useModuleCatalogOpProgress: stop cancels busy strip', async () => {
  let resolveBrowse: ((row: any) => void) | undefined;
  const store = {
    Browse: () =>
      new Promise(resolve => {
        resolveBrowse = (row: any) => resolve(row);
      }),
  } as any;
  const api = useModuleCatalogOpProgress(store, msgs);
  const pending = api.startOpProgress('op-hang', { moduleName: 'core', action: 'upgrade' });
  expect(api.progressBusy.value).toBe(true);
  api.stop();
  expect(api.progressBusy.value).toBe(false);
  resolveBrowse?.({ Status: 'succeeded', Phase: 'done' });
  await pending;
  expect(api.progressBusy.value).toBe(false);
});

test('useModuleCatalogOpProgress: failed terminal still refreshes', async () => {
  let refreshed = 0;
  const store = {
    Browse: async () => ({
      Status: 'failed',
      Phase: 'done',
      ErrorMessage: 'nope',
    }),
  } as any;
  const api = useModuleCatalogOpProgress(store, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await api.startOpProgress('op-fail', { moduleName: 'core', action: 'upgrade' });
  expect(refreshed).toBe(1);
  expect(api.progressBusy.value).toBe(false);
});

test('useModuleCatalogOpProgress: empty id and missing store are no-ops', async () => {
  let refreshed = 0;
  const api = useModuleCatalogOpProgress(null, msgs, {
    refreshCatalog: async () => {
      refreshed += 1;
    },
  });
  await api.startOpProgress('  ');
  expect(api.progressBusy.value).toBe(false);
  await api.startOpProgress('op-nostore', { moduleName: 'x', action: 'install' });
  expect(api.progressBusy.value).toBe(false);
  expect(refreshed).toBe(1);
});

test('useModuleCatalogOpProgress: titleFor falls back without module name', async () => {
  const store = {
    Browse: async () => ({ Status: 'succeeded', Phase: 'done' }),
  } as any;
  const api = useModuleCatalogOpProgress(store, {
    ...msgs,
    titleFor: (action: string, moduleName: string) =>
      moduleName ? `${action}:${moduleName}` : `${action}:blank`,
  });
  const done = api.startOpProgress('op-title', { moduleName: '', action: 'uninstall' });
  expect(api.progressTitle.value).toBe('uninstall:blank');
  await done;
});

test('useModuleCatalogOpProgress: scope dispose stops the progress strip', async () => {
  let resolveBrowse: ((row: any) => void) | undefined;
  const store = {
    Browse: () =>
      new Promise(resolve => {
        resolveBrowse = resolve;
      }),
  } as any;
  let api: ReturnType<typeof useModuleCatalogOpProgress> | undefined;
  const scope = effectScope();
  scope.run(() => {
    api = useModuleCatalogOpProgress(store, msgs);
    void api.startOpProgress('op-scope', { moduleName: 'core', action: 'install' });
  });
  await Promise.resolve();
  expect(api!.progressBusy.value).toBe(true);
  scope.stop();
  expect(api!.progressBusy.value).toBe(false);
  resolveBrowse?.({ Status: 'succeeded', Phase: 'done' });
});
