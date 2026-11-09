// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogListView from './ModuleCatalogListView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogListView.vue mounts under choysumMount', async () => {
  const store = {
    $id: 'fe-stub-module-catalog-list-store',
    records: {},
    state: { result: { rows: [] }, loading: false },
    RequestSync: async () => ({ Id: 'op-sync' }),
  };
  const wrapper = mount(ModuleCatalogListView as any, {
    props: { store },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/list',
        fullPath: '/meta/modules/list',
        name: 'MetaModuleCatalogList',
        params: {},
        meta: {
          viewKind: 'list',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-view"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-catalog-table"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogListView.vue expose openDrawer and boots from initialOperationId', async () => {
  const wrapper = mount(ModuleCatalogListView as any, {
    props: {
      store: {
        state: { result: { rows: [] }, loading: false },
        RequestSync: async () => ({ Id: 'op-sync' }),
      },
      initialOperationId: 'op-list-boot',
      operationStore: {
        Browse: async () => ({ Status: 'succeeded', Phase: 'done' }),
      },
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/list',
        fullPath: '/meta/modules/list',
        name: 'MetaModuleCatalogList',
        params: {},
        meta: {
          viewKind: 'list',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
    }),
  });
  await flushPromises();
  const vm = wrapper.vm as any;
  expect(typeof vm.openDrawer).toBe('function');
  vm.openDrawer({ moduleName: 'auth', action: 'install' });
  vm.closeDrawer();
  wrapper.unmount();
});
