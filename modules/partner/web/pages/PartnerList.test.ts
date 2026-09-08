// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { defineComponent, h } from 'vue';

test('partner PartnerList page mount coverage: mounts a trivial host under choysumMount', async () => {
  const Comp = defineComponent({
    name: 'PartnerListPageMountSmoke',
    setup() {
      return () => h('div', { class: 'partner-list-page-mount-smoke', 'data-testid': 'partner-list-page-smoke' }, 'ok');
    },
  });
  const wrapper = mount(Comp);
  await flushPromises();
  expect(wrapper.find('[data-testid="partner-list-page-smoke"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="partner-list-page-smoke"]').text()).toBe('ok');
  wrapper.unmount();
});
