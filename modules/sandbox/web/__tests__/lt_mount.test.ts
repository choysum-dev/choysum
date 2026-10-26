// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import SandboxRecordList from '../pages/SandboxRecordList.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('lt_mount: mounts real SandboxRecordList.vue under choysumMount', async () => {
  const wrapper = mount(SandboxRecordList as any, {
    global: buildPageMountGlobal({
      route: { path: '/sandbox/records', fullPath: '/sandbox/records' },
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
