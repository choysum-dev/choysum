// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import MetaModelData from './MetaModelData.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('MetaModelData.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mount(MetaModelData as any, {
    global: buildPageMountGlobal({
      route: {
        path: '/web/external-ids/1',
        fullPath: '/web/external-ids/1',
        params: { id: '1' },
      },
    }),
  });
  await flushPromises();
  expect(
    wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-opage"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists()
  ).toBe(true);
  wrapper.unmount();
});
