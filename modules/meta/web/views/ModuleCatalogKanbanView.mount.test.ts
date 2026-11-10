// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogKanbanView from './ModuleCatalogKanbanView.vue';
import { createModuleCatalogKanbanErrorHandlers, moduleCatalogKanbanRowToCard } from './module_catalog_kanban_view';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogKanbanView.vue mounts under choysumMount', async () => {
  const store = {
    $id: 'fe-stub-module-catalog-kanban-store',
    records: {},
    state: { result: { rows: [] }, loading: false },
    RequestSync: async () => ({ Id: 'op-sync' }),
  };
  const wrapper = mount(ModuleCatalogKanbanView as any, {
    props: { store },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules',
        fullPath: '/meta/modules',
        name: 'MetaModuleCatalog',
        params: {},
        meta: {
          viewKind: 'kanban',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-view"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogKanbanView.vue expose openDrawer closeDrawer and boots operation progress', async () => {
  const wrapper = mount(ModuleCatalogKanbanView as any, {
    props: {
      store: {
        state: { result: { rows: [] }, loading: false },
        RequestSync: async () => ({ Id: 'op-sync' }),
      },
      initialOperationId: 'op-initial',
      operationStore: {
        Browse: async () => ({ Status: 'succeeded', Phase: 'done' }),
      },
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules',
        fullPath: '/meta/modules',
        name: 'MetaModuleCatalog',
        params: {},
        meta: {
          viewKind: 'kanban',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
    }),
  });
  await flushPromises();
  const vm = wrapper.vm as any;
  expect(typeof vm.openDrawer).toBe('function');
  vm.openDrawer({ moduleName: 'core', action: 'upgrade', registryNewerThanLocal: true });
  vm.closeDrawer();
  wrapper.unmount();
});

test('ModuleCatalogKanbanView.vue card helpers label status and navigate on click', async () => {
  const pushed: string[] = [];
  const wrapper = mount(ModuleCatalogKanbanView as any, {
    props: {
      store: {
        state: { result: { rows: [] }, loading: false },
        RequestSync: async () => ({ Id: '' }),
      },
    },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules',
        fullPath: '/meta/modules',
        name: 'MetaModuleCatalog',
        params: {},
        meta: {
          viewKind: 'kanban',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
      router: {
        push: (to: string) => {
          pushed.push(to);
          return Promise.resolve();
        },
      },
    }),
  });
  await flushPromises();
  const vm = wrapper.vm as any;
  const card = moduleCatalogKanbanRowToCard(
    {
      Id: '1',
      ModuleName: 'partner',
      Installed: true,
      UpdateAvailable: true,
      Available: true,
    },
    0,
    'all',
  );
  expect(String(vm.cardStatusLabel(card)).length).toBeGreaterThan(0);
  vm.onCardClick(card);
  expect(pushed).toContain('/meta/modules/partner');
  wrapper.unmount();
});

test('ModuleCatalogKanbanView.vue wires load error handlers through shared factory', () => {
  const reported: string[] = [];
  const handlers = createModuleCatalogKanbanErrorHandlers(
    { loadFailed: 'a', searchFailed: 'b', loadMoreFailed: 'c' },
    msg => reported.push(msg),
  );
  handlers.onLoadError('e');
  handlers.onSearchError('e');
  handlers.onLoadMoreError('e');
  expect(reported).toEqual(['a', 'b', 'c']);
});
