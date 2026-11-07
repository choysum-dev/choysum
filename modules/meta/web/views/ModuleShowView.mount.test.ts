// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleShowView from './ModuleShowView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleShowView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-module-show-store', records: {}, state: {} };
  const wrapper = mount(ModuleShowView as any, {
    props: { store, recordId: 'cat-1', showHeader: true },
    global: buildPageMountGlobal({
      route: {
        path: '/meta/modules/core',
        fullPath: '/meta/modules/core',
        name: 'MetaModuleShow',
        params: { name: 'core' },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  wrapper.unmount();
});
