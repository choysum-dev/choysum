// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { useAuthStore } from '@/auth/web/stores/auth';
import ModuleCatalogFormView from './ModuleCatalogFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

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
    $id: 'fe-stub-module-catalog-form-store',
    state: {
      records: { [recordId]: { Id: recordId, ...row } },
      result: { rows: [] },
    },
    Browse: async () => ({}),
    Search: async () => [],
  };
  const wrapper = mount(ModuleCatalogFormView as any, {
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

test('ModuleCatalogFormView.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mountShow({ ModuleName: 'core', Installed: false, Available: true });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogFormView.vue resolves CTAs from Form Browse RecordRow payload shape', async () => {
  // beginDisplay writes `{ kind, key, payload }` into store.state.result.rows — not flat rows
  // and not always state.records[id]. CTAs must unwrap payload or the header stays empty.
  // Installed must be present in the payload (Form registers it via ChoyVirtualField for Browse).
  const wrapper = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        $id: 'fe-stub-module-catalog-form-recordrow',
        state: {
          records: {},
          result: {
            rows: [
              {
                kind: 'record',
                key: 'cat-rr',
                payload: {
                  Id: 'cat-rr',
                  ModuleName: 'meta',
                  Installed: true,
                  Available: true,
                  UpdateAvailable: false,
                },
              },
            ],
          },
        },
        Browse: async () => ({}),
      },
      recordId: 'cat-rr',
      showHeader: true,
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/meta',
        fullPath: '/meta/modules/meta',
        name: 'MetaModuleShow',
        params: { name: 'meta' },
      },
    }),
  });
  seedModuleActionPermissions();
  await flushPromises();
  expect((wrapper.vm as any).primaryAction).toBe('upgrade');
  expect(wrapper.find('[data-testid="module-catalog-form-status"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="module-catalog-form-primary-upgrade"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-catalog-form-uninstall"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogFormView.vue primary and uninstall actions open the drawer for the matched row', async () => {
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
  expect(install.find('[data-testid="module-catalog-form-primary-install"]').exists()).toBe(true);
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
  expect(upgrade.find('[data-testid="module-catalog-form-primary-upgrade"]').exists()).toBe(true);
  expect(upgrade.find('[data-testid="module-catalog-form-uninstall"]').exists()).toBe(true);
  upgradeVm.onPrimaryClick();
  expect(upgradeVm.drawerAction).toBe('upgrade');
  upgrade.unmount();

  // Primary CTAs follow row state (List/Kanban parity); uninstall still needs action ACL.
  const noUninstallAcl = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
  });
  seedModuleActionPermissions([]);
  await flushPromises();
  expect((noUninstallAcl.vm as any).primaryAction).toBe('upgrade');
  expect(noUninstallAcl.find('[data-testid="module-catalog-form-uninstall"]').exists()).toBe(false);
  noUninstallAcl.unmount();

  let resolveBrowse: ((row: any) => void) | undefined;
  const progress = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        $id: 'fe-stub-module-catalog-form-progress',
        state: {
          records: {
            'cat-1': {
              Id: 'cat-1',
              ModuleName: 'core',
              Installed: true,
              Available: true,
              ActiveOperationId: 'op-live',
            },
          },
          result: { rows: [] },
        },
        Browse: async () => ({}),
      },
      recordId: 'cat-1',
      showHeader: true,
      operationStore: {
        Browse: () =>
          new Promise(resolve => {
            resolveBrowse = (row: any) => resolve(row);
          }),
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
  seedModuleActionPermissions([]);
  await flushPromises();
  const progressVm = progress.vm as any;
  expect(progressVm.primaryAction).toBe('progress');
  expect(String(progressVm.primaryActionLabel || '')).toContain('Progress');
  progressVm.onPrimaryClick();
  expect(!!progressVm.progressBusy).toBe(true);
  expect(progressVm.drawerOpen).toBe(false);
  resolveBrowse?.({ Status: 'succeeded', Phase: 'done' });
  progress.unmount();

  const forceUpgrade = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
    UpdateAvailable: false,
  });
  await flushPromises();
  const forceUpgradeVm = forceUpgrade.vm as any;
  expect(forceUpgradeVm.primaryAction).toBe('upgrade');
  expect(forceUpgradeVm.primaryButtonVariant).toBe('outline');
  forceUpgradeVm.onPrimaryClick();
  expect(forceUpgradeVm.drawerAction).toBe('upgrade');
  forceUpgrade.unmount();

  const emphasizedUpgrade = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
    UpdateAvailable: true,
  });
  await flushPromises();
  expect((emphasizedUpgrade.vm as any).primaryButtonVariant).toBe('default');
  emphasizedUpgrade.unmount();

  const uninstall = mountShow({
    ModuleName: 'core',
    Installed: true,
    Available: true,
  });
  await flushPromises();
  const uninstallVm = uninstall.vm as any;
  expect(uninstallVm.primaryAction).toBe('upgrade');
  uninstallVm.onUninstallClick();
  expect(uninstallVm.drawerAction).toBe('uninstall');
  uninstall.unmount();

  // Unmatched recordId must not act on an unrelated rows[0] module.
  const empty = mount(ModuleCatalogFormView as any, {
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

test('ModuleCatalogFormView.vue mounts Depends/Dependents as M2M lists and navigates by module name', async () => {
  const pushed: string[] = [];
  const wrapper = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        $id: 'fe-stub-module-catalog-form-depends',
        state: {
          records: {
            'cat-1': {
              Id: 'cat-1',
              ModuleName: 'meta',
              Installed: true,
              Available: true,
              Depends: [{ Id: 'dep-core', ModuleName: 'core', InstalledVersion: '1.0.0', Installed: true }],
              Dependents: [{ Id: 'dep-auth', ModuleName: 'auth', InstalledVersion: '1.1.0', Installed: true }],
            },
          },
          result: { rows: [] },
        },
        Browse: async () => ({}),
        Search: async () => [],
      },
      recordId: 'cat-1',
      showHeader: true,
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/meta',
        fullPath: '/meta/modules/meta',
        name: 'MetaModuleShow',
        params: { name: 'meta' },
      },
      router: {
        push: (to: unknown) => {
          pushed.push(String(to));
          return Promise.resolve();
        },
      },
    }),
  });
  seedModuleActionPermissions();
  await flushPromises();

  expect(wrapper.find('[data-testid="module-catalog-form-depends"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-catalog-form-dependents"]').exists()).toBe(true);
  const body = wrapper.text();
  expect(body).toContain('Depends');
  expect(body).toContain('Dependents');
  expect(body).toContain('Module Name');
  expect(body).toContain('Application');
  expect(body).toContain('License');
  expect(body).toContain('Version');
  expect(body).toContain('Status');

  (wrapper.vm as any).openCatalogModule('core');
  await flushPromises();
  expect(pushed).toContain('/meta/modules/core');
  (wrapper.vm as any).onDependRowClick({
    rowData: { ModuleName: 'auth' },
  });
  await flushPromises();
  expect(pushed).toContain('/meta/modules/auth');
  (wrapper.vm as any).openCatalogModule('');
  expect(pushed.length).toBe(2);

  wrapper.unmount();
});

test('ModuleCatalogFormView.vue settled handler refreshes the catalog row', async () => {
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
  const browseWrapper = mount(ModuleCatalogFormView as any, {
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
  const searchWrapper = mount(ModuleCatalogFormView as any, {
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

test('ModuleCatalogFormView.vue depend helpers unwrap payload and raw rows', async () => {
  const pushed: string[] = [];
  const wrapper = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        state: {
          records: {
            'cat-1': { Id: 'cat-1', ModuleName: 'meta', Installed: true, Available: true },
          },
          result: { rows: [] },
        },
      },
      recordId: 'cat-1',
      showHeader: true,
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/meta',
        fullPath: '/meta/modules/meta',
        name: 'MetaModuleShow',
        params: { name: 'meta' },
      },
      router: {
        push: (to: unknown) => {
          pushed.push(String(to));
          return Promise.resolve();
        },
      },
    }),
  });
  seedModuleActionPermissions();
  await flushPromises();
  (wrapper.vm as any).onDependRowClick({
    rowData: { raw: { ModuleName: 'core' } },
  });
  await flushPromises();
  expect(pushed).toContain('/meta/modules/core');
  wrapper.unmount();
});

test('ModuleCatalogFormView.vue depend helpers and onOpApplied cover status branches', async () => {
  const wrapper = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        state: {
          records: {
            'cat-1': { Id: 'cat-1', ModuleName: 'gone', Installed: false, Available: false },
          },
          result: { rows: [] },
        },
        Browse: async () => ({}),
      },
      recordId: 'cat-1',
      showHeader: true,
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/gone',
        fullPath: '/meta/modules/gone',
        name: 'MetaModuleShow',
        params: { name: 'gone' },
      },
    }),
  });
  seedModuleActionPermissions();
  await flushPromises();
  const vm = wrapper.vm as any;
  expect(vm.primaryAction).toBe(null);
  expect(vm.primaryActionLabel).toBe('');
  expect(typeof vm.dependFieldText).toBe('function');
  expect(vm.dependFieldText({}, 'ModuleName')).toBe('—');
  expect(vm.dependVersionLine({
    ModuleName: 'core',
    InstalledVersion: '1.0.0',
    LocalVersion: '1.0.0',
    Installed: true,
  })).toContain('1.0.0');
  expect(vm.dependStatusLabel({ Available: false })).toContain('Unavailable');
  expect(vm.dependStatusLabel({ Installed: true, UpdateAvailable: true })).toContain('Update');
  expect(vm.dependStatusLabel({ Installed: true })).toContain('Installed');
  expect(vm.dependStatusLabel({ Installed: false })).toContain('Not Installed');
  expect(vm.dependStatusLabel({ Installed: 'weird' })).toBeTruthy();
  expect(String(vm.dependStatusBadgeVariant({ Installed: true, Available: true })).length).toBeGreaterThan(0);
  const withOp = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        state: {
          records: {
            'cat-1': { Id: 'cat-1', ModuleName: 'core', Installed: false, Available: true },
          },
          result: { rows: [] },
        },
        Browse: async () => ({}),
      },
      recordId: 'cat-1',
      showHeader: true,
      operationStore: { Browse: async () => ({ Status: 'succeeded', Phase: 'done' }) },
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
  await flushPromises();
  const withOpVm = withOp.vm as any;
  withOpVm.drawerAction = 'install';
  withOpVm.drawerModuleName = 'core';
  withOpVm.drawerOpen = true;
  withOpVm.onOpApplied('op-applied');
  expect(withOpVm.drawerOpen).toBe(false);
  withOp.unmount();

  expect(typeof vm.onOpApplied).toBe('function');
  vm.onOpApplied('');
  wrapper.unmount();
});

test('ModuleCatalogFormView.vue matches RecordRow via key when records map is empty', async () => {
  const wrapper = mount(ModuleCatalogFormView as any, {
    props: {
      store: {
        state: {
          records: {},
          result: {
            rows: [
              {
                kind: 'record',
                key: 'by-key',
                payload: {
                  Id: 'by-key',
                  ModuleName: 'auth',
                  Installed: true,
                  Available: true,
                  ActiveOperationId: 'op-live',
                },
              },
            ],
          },
        },
      },
      recordId: 'by-key',
      showHeader: true,
      operationStore: { Browse: async () => ({ Status: 'queued', Phase: 'queued' }) },
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/auth',
        fullPath: '/meta/modules/auth',
        name: 'MetaModuleShow',
        params: { name: 'auth' },
      },
    }),
  });
  seedModuleActionPermissions();
  await flushPromises();
  expect((wrapper.vm as any).primaryAction).toBe('progress');
  wrapper.unmount();
});
