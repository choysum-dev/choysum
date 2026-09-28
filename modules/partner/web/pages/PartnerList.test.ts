// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import PartnerList from './PartnerList.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('PartnerList.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mount(PartnerList as any, {
    global: buildPageMountGlobal({
      route: { path: '/partner/partners', fullPath: '/partner/partners' },
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
