// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogKanbanView from './ModuleCatalogKanbanView.vue';
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
