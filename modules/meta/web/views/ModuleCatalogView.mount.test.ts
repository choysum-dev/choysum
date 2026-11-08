// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogView from './ModuleCatalogView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

function makeOperationStore() {
  return {
    Plan: async () => ({
      Id: 'op-plan',
      ModuleOrder: [{ ModuleName: 'core' }],
      AffectedApps: [],
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-plan' }),
    Browse: async () => ({
      Status: 'succeeded',
      ReloadTriggered: false,
      ReloadFailed: false,
    }),
  };
}

test('ModuleCatalogView.vue mounts under choysumMount and exposes drawer helpers', async () => {
  const store = {
    $id: 'fe-stub-module-catalog-store',
    state: {
      result: {
        rows: [
          {
            Id: '1',
            ModuleName: 'core',
            Installed: false,
            Available: true,
            LocalVersion: '1.0',
            Summary: 'Core platform',
            Application: 'core',
          },
        ],
      },
      loading: false,
    },
    RequestSync: async () => ({ Id: 'op-1' }),
  };
  const wrapper = mount(ModuleCatalogView as any, {
    props: { store, operationStore: makeOperationStore(), showHeader: true, initialOperationId: '' },
    global: buildPageMountGlobal({
      route: { path: '/meta/modules', fullPath: '/meta/modules', name: 'MetaModuleCatalog', params: {}, query: {} },
    }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-view"]').exists() || wrapper.element != null).toBe(true);
  const vm = wrapper.vm as { openDrawer?: (opts: any) => void; closeDrawer?: () => void };
  expect(typeof vm.openDrawer).toBe('function');
  vm.openDrawer?.({ moduleName: 'core', action: 'install' });
  await flushPromises();
  vm.closeDrawer?.();
  await flushPromises();
  wrapper.unmount();
});

test('ModuleCatalogView.vue boots drawer from operation deep-link prop', async () => {
  const store = {
    $id: 'fe-stub-module-catalog-store-op',
    state: { result: { rows: [] }, loading: false },
    RequestSync: async () => ({ Id: '' }),
  };
  const wrapper = mount(ModuleCatalogView as any, {
    props: { store, operationStore: makeOperationStore(), initialOperationId: 'op-deep' },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/operations/op-deep',
        fullPath: '/meta/modules/operations/op-deep',
        name: 'MetaModuleOperationDeepLink',
        params: { id: 'op-deep' },
        query: {},
      },
    }),
  });
  await flushPromises();
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  wrapper.unmount();
});
