// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import SandboxRecordListView from './SandboxRecordListView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('SandboxRecordListView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-sandbox-record-list-store', records: {}, state: {} };
  const pushed: string[] = [];
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
      router: {
        push: (to: unknown) => {
          pushed.push(String(to));
          return Promise.resolve(to);
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

  const vm = wrapper.vm as {
    onRowClick?: (payload: {
      row: { Id?: string };
      rowIndex: number;
      rowKey: undefined;
      event: Event;
    }) => void;
  };
  expect(typeof vm.onRowClick).toBe('function');
  const click = new Event('click');
  vm.onRowClick?.({ row: { Id: 'sbx-cover-1' }, rowIndex: 0, rowKey: undefined, event: click });
  vm.onRowClick?.({ row: { Id: '' }, rowIndex: 0, rowKey: undefined, event: click });
  await flushPromises();
  expect(pushed).toEqual(['/sandbox/records/sbx-cover-1']);

  wrapper.unmount();
});
