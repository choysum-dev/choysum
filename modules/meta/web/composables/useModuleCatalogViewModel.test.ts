// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive, ref } from 'vue';
import { useModuleCatalogViewModel } from './useModuleCatalogViewModel';

const msgs = {
  unknown: 'Unknown',
  install: 'Install',
  upgrade: 'Upgrade',
  progress: 'View Progress',
  installed: 'Installed',
  updateAvailable: 'Update available',
  uninstalled: 'Not Installed',
  unavailable: 'Unavailable',
  syncFailed: (msg: string) => `fail:${msg}`,
  syncSettledOk: (summary: string) => (summary ? `ok:${summary}` : 'ok'),
  syncSettledPartial: (summary: string) => `partial:${summary}`,
  syncSettledFailed: (message: string) => `err:${message}`,
  syncJobStillRunning: 'still-running',
  syncServiceRestarting: 'restarting',
  syncFailedToGetStatus: 'no-status',
  formatSyncSummaryOk: (sync: { total: number; catalogRows: number }) =>
    `synced ${sync.total} / catalog ${sync.catalogRows}`,
  formatSyncSummaryPartial: (sync: {
    success: number;
    total: number;
    failed: number;
    catalogRows: number;
  }) => `synced ${sync.success}/${sync.total} (${sync.failed} failed) / catalog ${sync.catalogRows}`,
  opSettledOk: 'Operation Succeeded',
  opSettledFailed: (msg: string) => `Operation Failed: ${msg}`,
  opJobStillRunning: 'still-running',
  opServiceRestarting: 'restarting',
  opFailedToGetStatus: 'no-status',
  opReloadingService: () => 'Almost done, refreshing…',
  opExecuting: () => 'Working on it, please wait…',
  opQueued: () => 'Getting ready…',
  opFinishing: () => 'Wrapping up…',
  opInProgress: () => 'Please wait…',
  opProgressTitle: (action: string, moduleName: string) => {
    const name = String(moduleName || '').trim();
    if (action === 'install') return name ? `Installing ${name}` : 'Installing…';
    if (action === 'upgrade') return name ? `Upgrading ${name}` : 'Upgrading…';
    if (action === 'uninstall') return name ? `Uninstalling ${name}` : 'Uninstalling…';
    if (action === 'load_demo') return 'Loading demo data…';
    return name ? `Working on ${name}` : 'Working…';
  },
};

function makeStore(rows: any[] = []) {
  return {
    state: { result: { rows }, loading: false },
    RequestSync: async () => ({ Id: 'op-sync' }),
  } as any;
}

function makeRoute(partial: Record<string, unknown> = {}) {
  return reactive({
    name: 'MetaModuleCatalog',
    params: {},
    query: {},
    ...partial,
  }) as any;
}

test('useModuleCatalogViewModel: labels status and drives drawer actions', () => {
  const store = makeStore([
    {
      Id: '1',
      ModuleName: 'core',
      Installed: true,
      InstalledVersion: '1.0',
      LocalVersion: '1.1',
      UpdateAvailable: true,
      Available: true,
    },
    { Id: '2', ModuleName: 'base', Installed: false, LocalVersion: '2.0', Available: true },
    { Id: '3', ModuleName: 'gone', Available: false },
  ]);
  const pushed: string[] = [];
  const route = makeRoute();
  let resolveBrowse: ((row: any) => void) | undefined;
  const operationStore = {
    Browse: () =>
      new Promise(resolve => {
        resolveBrowse = (row: any) => resolve(row);
      }),
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    operationStore,
    route,
    router: {
      push: (to: any) => {
        pushed.push(String(to));
        return Promise.resolve();
      },
      replace: async () => undefined,
    } as any,
    msgs,
  });

  const core = store.state.result.rows[0];
  expect(api.statusLabel(core)).toBe('Update available');
  expect(api.primaryActionFor(core)).toBe('upgrade');
  expect(api.primaryActionLabel('upgrade')).toBe('Upgrade');
  expect(api.primaryActionLabel(null)).toBe('');
  expect(api.statusBadgeVariant(core)).toBe('secondary');
  expect(api.statusLabel({ Installed: true, UpdateAvailable: false })).toBe('Installed');
  expect(api.statusBadgeVariant({ Installed: true, UpdateAvailable: false, Available: true })).toBe(
    'outline',
  );
  expect(api.statusLabel({ Available: false })).toBe('Unavailable');
  expect(api.statusLabel({ Installed: false })).toBe('Not Installed');
  expect(api.statusLabel({})).toBe('Unknown');
  expect(api.field({ ModuleName: null }, 'ModuleName')).toBe('');
  expect(api.field(null, 'ModuleName')).toBe('');

  api.onRowClick({ row: core } as any);
  expect(pushed).toContain('/meta/modules/core');
  api.toShow({});
  expect(pushed.filter(p => p.includes('/meta/modules/')).length).toBe(1);

  api.onPrimaryAction(core);
  expect(api.drawerOpen.value).toBe(true);
  expect(api.drawerAction.value).toBe('upgrade');
  expect(api.drawerModuleName.value).toBe('core');

  api.onPrimaryAction({ ModuleName: 'base', Installed: false, Available: true });
  expect(api.drawerAction.value).toBe('install');

  api.closeDrawer();
  api.onPrimaryAction({ ModuleName: 'busy', ActiveOperationId: 'op-9' });
  expect(api.progressBusy.value).toBe(true);
  expect(api.progressHeadline.value).toContain('Getting ready');
  expect(api.drawerOpen.value).toBe(false);

  api.stopOpProgress();
  resolveBrowse?.({ Status: 'succeeded', Phase: 'done' });
  expect(api.progressBusy.value).toBe(false);

  api.closeDrawer();
  expect(api.drawerOpen.value).toBe(false);
  expect(api.drawerOperationId.value).toBe('');

  expect(api.canUninstall({ Installed: true, Available: true })).toBe(true);
  expect(api.canUninstall({ Installed: true, ActiveOperationId: 'op-1' })).toBe(false);
  api.onUninstall({ ModuleName: 'core', Installed: true, Available: true });
  expect(api.drawerAction.value).toBe('uninstall');
  expect(api.drawerModuleName.value).toBe('core');
});

test('useModuleCatalogViewModel: load-demo button visibility and drawer open', () => {
  const route = makeRoute();
  const router = { push: async () => undefined, replace: async () => undefined } as any;
  const pending = useModuleCatalogViewModel({
    store: makeStore([
      { Id: '1', ModuleName: 'base', Installed: true, HasDemo: true, DemoLoaded: false },
      { Id: '2', ModuleName: 'core', Installed: true, HasDemo: true, DemoLoaded: true },
      { Id: '3', ModuleName: 'task', Installed: true, HasDemo: false, DemoLoaded: false },
    ]),
    route,
    router,
    msgs,
  });
  expect(pending.hasPendingDemo.value).toBe(true);
  expect(pending.pendingDemoRows.value.map((r: any) => r.ModuleName)).toEqual(['base']);
  pending.onLoadDemo();
  expect(pending.drawerOpen.value).toBe(true);
  expect(pending.drawerAction.value).toBe('load_demo');

  const done = useModuleCatalogViewModel({
    store: makeStore([{ Id: '2', ModuleName: 'core', Installed: true, HasDemo: true, DemoLoaded: true }]),
    route,
    router,
    msgs,
  });
  expect(done.hasPendingDemo.value).toBe(false);
  done.onLoadDemo();
  expect(done.drawerOpen.value).toBe(false);
});

test('useModuleCatalogViewModel: boots and syncs drawer from operation deep link only', () => {
  const store = makeStore([{ Id: '1', ModuleName: 'core' }]);
  const route = makeRoute({
    name: 'MetaModuleOperationDeepLink',
    params: { id: 'op-deep' },
  });
  let resolveBrowse: ((row: any) => void) | undefined;
  const api = useModuleCatalogViewModel({
    store,
    operationStore: {
      Browse: () =>
        new Promise(resolve => {
          resolveBrowse = (row: any) => resolve(row);
        }),
    } as any,
    route,
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.bootFromRoute();
  expect(api.progressBusy.value).toBe(true);
  expect(api.drawerOpen.value).toBe(false);

  route.name = 'MetaModuleCatalog';
  route.params = {};
  api.syncDrawerFromRoute();
  expect(api.progressBusy.value).toBe(false);
  resolveBrowse?.({ Status: 'succeeded', Phase: 'done' });

  api.openDrawer({ moduleName: 'core', action: 'install' });
  expect(api.drawerOpen.value).toBe(true);
  api.syncDrawerFromRoute();
  expect(api.drawerOpen.value).toBe(true);
  api.closeDrawer();

  expect(api.resolveInitialOperationId()).toBe('');
  const withProp = useModuleCatalogViewModel({
    store,
    operationStore: {
      Browse: () => new Promise(() => {}),
    } as any,
    route: makeRoute({ name: 'MetaModuleCatalog', params: { id: 'ignored' } }),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    initialOperationId: () => 'op-from-prop',
    msgs,
  });
  expect(withProp.resolveInitialOperationId()).toBe('op-from-prop');
  withProp.bootFromRoute();
  expect(withProp.progressBusy.value).toBe(true);
  withProp.stopOpProgress();

  const emptyOp = useModuleCatalogViewModel({
    store,
    route: makeRoute({ name: 'MetaModuleOperationDeepLink', params: { id: '' } }),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  emptyOp.openDrawer({ operationId: 'x', action: 'progress' });
  emptyOp.syncDrawerFromRoute();
  expect(emptyOp.drawerOpen.value).toBe(false);

  const initialRef = ref('op-ref');
  const withRef = useModuleCatalogViewModel({
    store: makeStore(),
    route: makeRoute({ name: 'MetaModuleCatalog' }),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    initialOperationId: initialRef,
    msgs,
  });
  expect(withRef.resolveInitialOperationId()).toBe('op-ref');
});

test('useModuleCatalogViewModel: onSync does not open drawer and refreshes on settle', async () => {
  let calls = 0;
  let refreshed = 0;
  const store = {
    state: {
      result: { rows: [{ Id: '1' }] },
      loading: false,
      list: {
        refresh: async () => {
          refreshed += 1;
        },
      },
    },
    RequestSync: async () => {
      calls += 1;
      return 'op-string';
    },
  } as any;
  const operationStore = {
    Browse: async () => ({
      Action: 'sync',
      Status: 'succeeded',
      PlanJson: JSON.stringify({
        syncResult: { total: 1, success: 1, failed: 0, catalogRows: 3 },
      }),
    }),
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    operationStore,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  await api.onSync();
  expect(calls).toBe(1);
  expect(api.drawerOpen.value).toBe(false);
  expect(api.syncBusy.value).toBe(false);
  expect(refreshed).toBe(1);

  // Idle again — second click enqueues again.
  await api.onSync();
  expect(calls).toBe(2);
});

test('useModuleCatalogViewModel: onDrawerSettled refreshes install/upgrade settle', async () => {
  let refreshed = 0;
  const store = {
    state: {
      result: { rows: [{ Id: '1' }] },
      loading: false,
      list: {
        refresh: async () => {
          refreshed += 1;
        },
      },
    },
    RequestSync: async () => ({ Id: 'op-settle' }),
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.onDrawerSettled();
  await Promise.resolve();
  expect(refreshed).toBe(1);
});

test('useModuleCatalogViewModel: bootFromRoute does not request sync', async () => {
  let calls = 0;
  const store = {
    state: { result: { rows: [{ Id: '1' }] }, loading: false },
    RequestSync: async () => {
      calls += 1;
      return { Id: 'op-boot' };
    },
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.bootFromRoute();
  await Promise.resolve();
  expect(calls).toBe(0);
});

test('useModuleCatalogViewModel: lastSyncAtLabel picks newest raw timestamp', () => {
  const store = makeStore([
    { Id: '1', LastSyncAt: '2026-01-02T00:00:00Z' },
    { Id: '2', LastSyncAt: '2026-03-01T00:00:00Z' },
    { Id: '3', LastSyncAt: 'not-a-date' },
  ]);
  const api = useModuleCatalogViewModel({
    store,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  expect(api.lastSyncAtLabel.value).toContain('2026');
});

test('useModuleCatalogViewModel: refreshCatalog prefers list.refresh then Search', async () => {
  let refreshed = 0;
  let searched = 0;
  const withList = {
    state: {
      result: { rows: [] },
      loading: false,
      list: {
        refresh: async () => {
          refreshed += 1;
        },
      },
    },
    Search: async () => {
      searched += 1;
    },
  } as any;
  const api = useModuleCatalogViewModel({
    store: withList,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  await api.refreshCatalog();
  expect(refreshed).toBe(1);
  expect(searched).toBe(0);

  const searchOnly = {
    state: { result: { rows: [] }, loading: false },
    Search: async () => {
      searched += 1;
    },
  } as any;
  const api2 = useModuleCatalogViewModel({
    store: searchOnly,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  await api2.refreshCatalog();
  expect(searched).toBe(1);
});

test('useModuleCatalogViewModel: onOpApplied starts progress and onDrawerSettled refreshes only', async () => {
  let refreshed = 0;
  const store = {
    state: {
      result: { rows: [{ Id: '1', ModuleName: 'core' }] },
      list: { refresh: async () => { refreshed += 1; } },
    },
    RequestSync: async () => ({ Id: 'x' }),
  } as any;
  const operationStore = {
    Browse: async () => ({ Status: 'succeeded', Phase: 'done' }),
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    operationStore,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.openDrawer({ moduleName: 'core', action: 'install' });
  api.onOpApplied('op-applied');
  expect(api.drawerOpen.value).toBe(false);
  await Promise.resolve();
  expect(refreshed).toBeGreaterThanOrEqual(1);

  api.onDrawerSettled({ resultStatus: 'FAILED', summary: 'boom' });
  api.onDrawerSettled({ resultStatus: 'SUCCEEDED' });
  expect(api.canUninstall({ Installed: true, Available: false })).toBe(false);
});

test('useModuleCatalogViewModel: lastSyncAtLabel accepts Date and numeric timestamps', () => {
  const store = makeStore([
    { Id: '1', LastSyncAt: new Date('2026-02-01T00:00:00Z') },
    { Id: '2', LastSyncAt: Date.UTC(2026, 5, 1) },
  ]);
  const api = useModuleCatalogViewModel({
    store,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  expect(String(api.lastSyncAtLabel.value).length).toBeGreaterThan(0);
});
