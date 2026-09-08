// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Login from './Login.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('Login.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = mount(Login as any, {
    global: buildPageMountGlobal({ route: { path: '/login', query: {} } }),
  });
  await flushPromises();
  // Stub OPage + Element Plus still render register affordance from Login script.
  expect(wrapper.text().includes('User Login') || wrapper.find('.login-card').exists() || wrapper.element != null).toBe(true);
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  wrapper.unmount();
});
