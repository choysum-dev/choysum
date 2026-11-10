// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogList from './ModuleCatalogList.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogList.vue mounts under choysumMount', async () => {
  const wrapper = mount(ModuleCatalogList as any, {
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
  expect(
    wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists(),
  ).toBe(true);
  wrapper.unmount();
});
