// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive, ref } from 'vue';
import { useModuleCatalogViewModel } from './useModuleCatalogViewModel';

const msgs = {
  all: 'All',
  installed: 'Installed',
  upgradeable: 'Upgradeable',
  uninstalled: 'Not Installed',
  unavailable: 'Unavailable',
  cards: 'Cards',
  table: 'Table',
  unknown: 'Unknown',
  install: 'Install',
  upgrade: 'Upgrade',
  progress: 'View Progress',
  syncTriggered: (id: string) => `ok:${id}`,
  syncTriggeredEmpty: 'ok',
  syncFailed: (msg: string) => `fail:${msg}`,
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

test('useModuleCatalogViewModel: filters cards and labels status / versions / actions', () => {
  const store = makeStore([
    { Id: '1', ModuleName: 'core', Installed: true, InstalledVersion: '1.0', LocalVersion: '1.1', UpdateAvailable: true, Available: true },
    { Id: '2', ModuleName: 'base', Installed: false, LocalVersion: '2.0', Available: true },
    { Id: '3', ModuleName: 'gone', Available: false },
  ]);
  const pushed: string[] = [];
  const replaced: any[] = [];
  const route = makeRoute();
  const api = useModuleCatalogViewModel({
    store,
    route,
    router: {
      push: (to: any) => {
        pushed.push(String(to));
        return Promise.resolve();
      },
      replace: (to: any) => {
        replaced.push(to);
        return Promise.resolve();
      },
    } as any,
    msgs,
  });

  expect(api.filteredCardRows.value).toHaveLength(3);
  api.filter.value = 'upgradeable';
  expect(api.filteredCardRows.value.map((r: any) => r.ModuleName)).toEqual(['core']);
  api.filter.value = 'uninstalled';
  expect(api.filteredCardRows.value.map((r: any) => r.ModuleName)).toEqual(['base']);
  api.filter.value = 'unavailable';
  expect(api.filteredCardRows.value.map((r: any) => r.ModuleName)).toEqual(['gone']);

  const core = store.state.result.rows[0];
  expect(api.statusLabel(core)).toBe('Installed');
  expect(api.versionLine(core)).toBe('1.0 → 1.1');
  expect(api.primaryActionFor(core)).toBe('upgrade');
  expect(api.primaryActionLabel('upgrade')).toBe('Upgrade');
  expect(api.primaryActionLabel(null)).toBe('');
  expect(api.cardKey(core, 0)).toBe('1');
  expect(api.cardKey({}, 9)).toBe('row-9');
  expect(api.statusBadgeVariant(core)).toBeTruthy();
  expect(api.cardBadgeVariant(core)).toBeTruthy();
  expect(api.statusLabel({ Available: false })).toBe('Unavailable');
  expect(api.statusLabel({ Installed: false })).toBe('Not Installed');
  expect(api.statusLabel({})).toBe('Unknown');
  expect(api.versionLine({ Installed: true, InstalledVersion: '1' })).toBe('1');
  expect(api.versionLine({ RegistryVersion: '9' })).toBe('9');

  api.onViewSelect('table');
  expect(api.viewMode.value).toBe('table');
  expect(replaced[0].query.view).toBe('table');
  expect(replaced[0].name).toBe('MetaModuleCatalog');
  expect(replaced[0].params).toEqual({});
  api.onViewSelect('cards');
  expect(api.viewMode.value).toBe('cards');

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

  api.onPrimaryAction({ ModuleName: 'busy', ActiveOperationId: 'op-9' });
  expect(api.drawerAction.value).toBe('progress');
  expect(api.drawerOperationId.value).toBe('op-9');

  api.onPrimaryAction({ Installed: true, UpdateAvailable: false, Available: true });
  expect(api.drawerOperationId.value).toBe('op-9');

  api.closeDrawer();
  expect(api.drawerOpen.value).toBe(false);
  expect(api.drawerOperationId.value).toBe('');

  expect(api.canUninstall({ Installed: true, Available: true })).toBe(true);
  expect(api.canUninstall({ Installed: true, ActiveOperationId: 'op-1' })).toBe(false);
  api.onUninstall({ ModuleName: 'core', Installed: true, Available: true });
  expect(api.drawerAction.value).toBe('uninstall');
  expect(api.drawerModuleName.value).toBe('core');
});

test('useModuleCatalogViewModel: boots and syncs drawer from operation deep link only', () => {
  const store = makeStore([{ Id: '1', ModuleName: 'core' }]);
  const route = makeRoute({
    name: 'MetaModuleOperationDeepLink',
    params: { id: 'op-deep' },
  });
  const api = useModuleCatalogViewModel({
    store,
    route,
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.bootFromRoute();
  expect(api.drawerOpen.value).toBe(true);
  expect(api.drawerOperationId.value).toBe('op-deep');

  route.name = 'MetaModuleCatalog';
  route.params = {};
  api.syncDrawerFromRoute();
  expect(api.drawerOpen.value).toBe(false);

  api.openDrawer({ moduleName: 'core', action: 'install' });
  expect(api.drawerOpen.value).toBe(true);
  // User-opened drawers stay open across non-deep-link route syncs (e.g. view query).
  api.syncDrawerFromRoute();
  expect(api.drawerOpen.value).toBe(true);
  api.closeDrawer();

  expect(api.resolveInitialOperationId()).toBe('');
  const withProp = useModuleCatalogViewModel({
    store,
    route: makeRoute({ name: 'MetaModuleCatalog', params: { id: 'ignored' } }),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    initialOperationId: () => 'op-from-prop',
    msgs,
  });
  expect(withProp.resolveInitialOperationId()).toBe('op-from-prop');
  withProp.bootFromRoute();
  expect(withProp.drawerOperationId.value).toBe('op-from-prop');

  const emptyOp = useModuleCatalogViewModel({
    store,
    route: makeRoute({ name: 'MetaModuleOperationDeepLink', params: { id: '' } }),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  emptyOp.openDrawer({ operationId: 'x', action: 'progress' });
  emptyOp.syncDrawerFromRoute();
  expect(emptyOp.drawerOpen.value).toBe(false);

  const queryRoute = makeRoute({ query: { view: 'table' } });
  const fromQuery = useModuleCatalogViewModel({
    store,
    route: queryRoute,
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  expect(fromQuery.viewMode.value).toBe('table');
  queryRoute.query = { view: 'cards' };
});

test('useModuleCatalogViewModel: onSync delegates to RequestSync', async () => {
  let calls = 0;
  const store = {
    state: { result: { rows: [{ Id: '1' }] }, loading: false },
    RequestSync: async () => {
      calls += 1;
      return 'op-string';
    },
  } as any;
  const api = useModuleCatalogViewModel({
    store,
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  api.onSync();
  await Promise.resolve();
  expect(calls).toBe(1);
});

test('useModuleCatalogViewModel: filterOptions and viewSwitcherOptions expose labels', () => {
  const api = useModuleCatalogViewModel({
    store: makeStore(),
    route: makeRoute(),
    router: { push: async () => undefined, replace: async () => undefined } as any,
    msgs,
  });
  expect(api.filterOptions.value.map(o => o.value)).toEqual([
    'all',
    'installed',
    'upgradeable',
    'uninstalled',
    'unavailable',
  ]);
  expect(api.viewSwitcherOptions.value.map(o => o.value)).toEqual(['cards', 'table']);
  expect(api.field({ ModuleName: null }, 'ModuleName')).toBe('');
  expect(api.field(null, 'ModuleName')).toBe('');
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
