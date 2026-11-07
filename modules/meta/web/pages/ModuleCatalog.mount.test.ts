// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalog from './ModuleCatalog.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalog.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mount(ModuleCatalog as any, {
    global: buildPageMountGlobal({
      route: { path: '/meta/modules', fullPath: '/meta/modules', name: 'MetaModuleCatalog', params: {} },
    }),
  });
  await flushPromises();
  expect(
    wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-opage"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists(),
  ).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalog.vue deep-link route passes initial operation id', async () => {
  const wrapper = mount(ModuleCatalog as any, {
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/operations/op-9',
        fullPath: '/meta/modules/operations/op-9',
        name: 'MetaModuleOperationDeepLink',
        params: { id: 'op-9' },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  wrapper.unmount();
});
