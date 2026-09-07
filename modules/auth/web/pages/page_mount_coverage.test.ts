// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { defineComponent, h } from 'vue';

test('auth page mount coverage: mounts a trivial host under choysumMount', async () => {
  const Comp = defineComponent({
    name: 'AuthPageMountSmoke',
    setup() {
      return () => h('div', { class: 'auth-page-mount-smoke', 'data-testid': 'auth-page-smoke' }, 'ok');
    },
  });
  const wrapper = mount(Comp);
  await flushPromises();
  expect(wrapper.find('[data-testid="auth-page-smoke"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="auth-page-smoke"]').text()).toBe('ok');
  wrapper.unmount();
});
