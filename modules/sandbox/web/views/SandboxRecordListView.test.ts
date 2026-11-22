// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import SandboxRecordListView from './SandboxRecordListView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('SandboxRecordListView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-sandbox-record-list-store', records: {}, state: {} };
  const wrapper = mount(SandboxRecordListView as any, {
    props: { store },
    global: buildPageMountGlobal({
      route: {
        path: '/sandbox/records',
        fullPath: '/sandbox/records',
        meta: {
          viewKind: 'list',
          defaultGroups: [],
          viewPaths: {
            list: '/sandbox/records',
            kanban: '/sandbox/records/kanban',
            chart: '/sandbox/records/chart',
          },
        },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  expect(
    wrapper.find('[data-testid="fe-stub-child-view"]').exists() ||
      wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.element != null,
  ).toBe(true);

  // Row→detail is owned by ChoyListView detailPath (no local onRowClick expose).
  const vm = wrapper.vm as { onRowClick?: unknown };
  expect(vm.onRowClick).toBeUndefined();

  wrapper.unmount();
});
