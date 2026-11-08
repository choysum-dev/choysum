// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { useAuthStore } from '@/auth/web/stores/auth';
import ModuleShowView from './ModuleShowView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { moduleUninstallAction } from './module_actions';

function seedModuleActionPermissions(actions: string[] = ['*']) {
  const auth = useAuthStore();
  auth.identity = {
    metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: [], actions } },
    },
  } as any;
}

function mountShow(row: Record<string, unknown>, recordId = 'cat-1') {
  const store = {
    $id: 'fe-stub-module-show-store',
    state: {
      records: { [recordId]: { Id: recordId, ...row } },
      result: { rows: [] },
    },
    Browse: async () => ({}),
    Search: async () => [],
  };
  const wrapper = mount(ModuleShowView as any, {
    props: {
      store,
      recordId,
      showHeader: true,
      operationStore: {
        Plan: async () => ({
          Id: 'op-1',
          ModuleOrder: [],
          AffectedApps: [],
          Risks: [],
          Blockers: [],
        }),
        Apply: async () => ({ Id: 'op-1' }),
        Browse: async () => ({ Status: 'succeeded' }),
      },
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/core',
        fullPath: '/meta/modules/core',
        name: 'MetaModuleShow',
        params: { name: 'core' },
      },
    }),
  });
  seedModuleActionPermissions();
  return wrapper;
}

test('ModuleShowView.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mountShow({ ModuleName: 'core', Installed: false, Available: true });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  wrapper.unmount();
});

test('ModuleShowView.vue primary and uninstall actions open the drawer for the matched row', async () => {
  const install = mountShow({
    ModuleName: 'core',
    Installed: false,
    Available: true,
    RegistryNewerThanLocal: true,
  });
  await flushPromises();
  const installVm = install.vm as any;
  expect(installVm.primaryAction).toBe('install');
  expect(String(installVm.primaryActionLabel || '')).toContain('Install');
  installVm.onPrimaryClick();
  expect(installVm.drawerOpen).toBe(true);
  expect(installVm.drawerAction).toBe('install');
  expect(installVm.drawerModuleName).toBe('core');
  expect(installVm.drawerRegistryNewer).toBe(true);
  install.unmount();

  const upgrade = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
    UpdateAvailable: true,
  });
  await flushPromises();
  const upgradeVm = upgrade.vm as any;
  expect(upgradeVm.primaryAction).toBe('upgrade');
  expect(String(upgradeVm.primaryActionLabel || '')).toContain('Upgrade');
  upgradeVm.onPrimaryClick();
  expect(upgradeVm.drawerAction).toBe('upgrade');
  upgrade.unmount();

  // Install/upgrade CTAs stay hidden without action permission; progress stays visible.
  const denied = mountShow({
    ModuleName: 'core',
    Installed: false,
    Available: true,
  });
  seedModuleActionPermissions([moduleUninstallAction]);
  await flushPromises();
  expect((denied.vm as any).primaryAction).toBe(null);
  denied.unmount();

  const progress = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
    ActiveOperationId: 'op-live',
  });
  seedModuleActionPermissions([]);
  await flushPromises();
  const progressVm = progress.vm as any;
  expect(progressVm.primaryAction).toBe('progress');
  expect(String(progressVm.primaryActionLabel || '')).toContain('Progress');
  progressVm.onPrimaryClick();
  expect(progressVm.drawerAction).toBe('progress');
  expect(progressVm.drawerOperationId).toBe('op-live');
  progress.unmount();

  const uninstall = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
  });
  await flushPromises();
  const uninstallVm = uninstall.vm as any;
  expect(uninstallVm.primaryAction).toBe(null);
  expect(uninstallVm.primaryActionLabel).toBe('');
  uninstallVm.onUninstallClick();
  expect(uninstallVm.drawerAction).toBe('uninstall');
  uninstall.unmount();

  // Unmatched recordId must not act on an unrelated rows[0] module.
  const empty = mount(ModuleShowView as any, {
    props: {
      store: {
        state: {
          records: {},
          result: { rows: [{ Id: 'other', ModuleName: 'x', Installed: false, Available: true }] },
        },
      },
      recordId: 'missing',
      showHeader: true,
    },
    global: buildPageMountGlobal({
      route: { path: '/meta/modules/x', fullPath: '/meta/modules/x', name: 'MetaModuleShow', params: { name: 'x' } },
    }),
  });
  await flushPromises();
  expect((empty.vm as any).primaryAction).toBe(null);
  empty.unmount();
});

test('ModuleShowView.vue settled handler refreshes the catalog row', async () => {
  let browsed = 0;
  const browseStore = {
    state: {
      records: {
        'cat-1': { Id: 'cat-1', ModuleName: 'core', Installed: false, Available: true },
      },
      result: { rows: [] },
    },
    Browse: async () => {
      browsed += 1;
      return {};
    },
  };
  const browseWrapper = mount(ModuleShowView as any, {
    props: { store: browseStore, recordId: 'cat-1', showHeader: true },
    global: buildPageMountGlobal({
      route: { path: '/meta/modules/core', fullPath: '/meta/modules/core', name: 'MetaModuleShow', params: { name: 'core' } },
    }),
  });
  await flushPromises();
  await (browseWrapper.vm as any).onDrawerSettled();
  expect(browsed).toBe(1);
  browseWrapper.unmount();

  let searched = 0;
  const searchStore = {
    state: {
      records: {
        'cat-1': { Id: 'cat-1', ModuleName: 'core', Installed: false, Available: true },
      },
      result: { rows: [] },
    },
    Search: async () => {
      searched += 1;
      return [];
    },
  };
  const searchWrapper = mount(ModuleShowView as any, {
    props: { store: searchStore, recordId: 'cat-1', showHeader: true },
    global: buildPageMountGlobal({
      route: { path: '/meta/modules/core', fullPath: '/meta/modules/core', name: 'MetaModuleShow', params: { name: 'core' } },
    }),
  });
  await flushPromises();
  await (searchWrapper.vm as any).onDrawerSettled();
  expect(searched).toBe(1);
  searchWrapper.unmount();
});
