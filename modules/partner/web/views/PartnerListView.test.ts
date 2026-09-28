// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import PartnerListView from './PartnerListView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('PartnerListView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-partner-list-store', records: {} };
  const wrapper = mount(PartnerListView as any, {
    props: { store },
    global: buildPageMountGlobal({ route: { path: '/partner', fullPath: '/partner' } }),
  });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  expect(
    wrapper.find('[data-testid="fe-stub-child-view"]').exists() ||
      wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists(),
  ).toBe(true);

  // Exercise exposed row-click navigation (permission stub always allows).
  const vm = wrapper.vm as { onRowClick?: (row: Record<string, unknown>) => void };
  expect(typeof vm.onRowClick).toBe('function');
  vm.onRowClick?.({ Id: 'ptn-cover-1' });
  vm.onRowClick?.({ Id: '' });
  await flushPromises();

  wrapper.unmount();
});
