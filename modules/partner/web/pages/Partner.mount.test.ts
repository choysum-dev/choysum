// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Partner from './Partner.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('Partner.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mount(Partner as any, {
    global: buildPageMountGlobal({ route: { path: '/partner/1', fullPath: '/partner/1' } }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  wrapper.unmount();
});
