// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { defineComponent, h } from 'vue';

test('base page mount coverage: mounts a trivial host under choysumMount', async () => {
  const Comp = defineComponent({
    name: 'BasePageMountSmoke',
    setup() {
      return () => h('div', { class: 'base-page-mount-smoke', 'data-testid': 'base-page-smoke' }, 'ok');
    },
  });
  const wrapper = mount(Comp);
  await flushPromises();
  expect(wrapper.find('[data-testid="base-page-smoke"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="base-page-smoke"]').text()).toBe('ok');
  wrapper.unmount();
});
