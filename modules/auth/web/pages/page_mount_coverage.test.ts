// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Login from './Login.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('auth page mount: mounts real Login.vue under choysumMount', async () => {
  const wrapper = mount(Login as any, {
    global: buildPageMountGlobal({ route: { path: '/login', query: {} } }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  expect(wrapper.text().includes('User Login') || wrapper.find('.fe-stub-ElCard').exists()).toBe(true);
  wrapper.unmount();
});
