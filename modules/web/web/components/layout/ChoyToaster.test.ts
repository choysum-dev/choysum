// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { mount } from '@choysum/test-utils';
import { restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyToaster from './ChoyToaster.vue';
import Toaster from '../vendor/ui/sonner/Sonner.vue';

test('ChoyToaster: mounts product host and forwards error-icon override', () => {
  stubSfc(Toaster as any, {
    name: 'Toaster',
    setup(_: unknown, { slots }: any) {
      return () =>
        h('div', { 'data-test': 'kit-toaster' }, slots['error-icon'] ? slots['error-icon']() : null);
    },
  });
  try {
    const wrapper = mount(ChoyToaster as any);
    expect(wrapper.find('[data-test=kit-toaster]').exists()).toBe(true);
    wrapper.unmount();
  } finally {
    restoreSfc(Toaster as any);
  }
});
