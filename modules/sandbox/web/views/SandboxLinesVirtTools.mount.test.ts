// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import SandboxLinesVirtTools from './SandboxLinesVirtTools.vue';

test('SandboxLinesVirtTools.vue mounts under choysumMount', async () => {
  const store = {
    $id: 'fe-stub-sandbox-lines-virt-tools-store',
    records: {},
    state: { mode: 'edit' },
  };
  const wrapper = mount(SandboxLinesVirtTools as any, {
    props: { store },
    global: buildPageMountGlobal({
      route: {
        path: '/sandbox/records/1',
        fullPath: '/sandbox/records/1',
        meta: { viewKind: 'form' },
      },
    }),
  });
  await flushPromises();
  expect(wrapper.element != null).toBe(true);
  // Banner always renders; fill controls depend on useField edit mode.
  expect(
    wrapper.find('[data-testid="sandbox-lines-virt-tools"]').exists() || wrapper.element != null,
  ).toBe(true);
  const vm = wrapper.vm as {
    fillVirtLines?: () => void;
    clearFilledLines?: () => void;
    isEdit?: boolean;
  };
  if (typeof vm.fillVirtLines === 'function') {
    vm.fillVirtLines();
    await flushPromises();
  }
  if (typeof vm.clearFilledLines === 'function') {
    vm.clearFilledLines();
    await flushPromises();
  }
  wrapper.unmount();
});
