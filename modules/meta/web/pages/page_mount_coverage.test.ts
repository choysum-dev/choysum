// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleList from './ModuleList.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('meta page mount: mounts real ModuleList.vue under choysumMount', async () => {
  const wrapper = mount(ModuleList as any, {
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  wrapper.unmount();
});
