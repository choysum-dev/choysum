// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogKanban from './ModuleCatalogKanban.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogKanban.vue mounts under choysumMount', async () => {
  const wrapper = mount(ModuleCatalogKanban as any, {
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
  expect(
    wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists(),
  ).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogKanban.vue passes deep-link operation id to kanban view', async () => {
  const wrapper = mount(ModuleCatalogKanban as any, {
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/operations/op-deep',
        fullPath: '/meta/modules/operations/op-deep',
        name: 'MetaModuleOperationDeepLink',
        params: { id: 'op-deep' },
        meta: {
          viewKind: 'kanban',
          defaultGroups: [],
          viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
        },
      },
    }),
  });
  await flushPromises();
  const exposed = (wrapper.vm as any).initialOperationId;
  expect(String(exposed?.value ?? exposed ?? '')).toBe('op-deep');
  wrapper.unmount();
});
