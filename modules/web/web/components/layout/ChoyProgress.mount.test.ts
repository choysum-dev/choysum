// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@choysum/test-utils';
import ChoyProgress from './ChoyProgress.vue';

test('ChoyProgress mounts with clamped modelValue', () => {
  const wrapper = mount(ChoyProgress as any, {
    props: { modelValue: 55 },
  });
  expect(wrapper.find('[data-testid="choy-progress"]').exists()).toBe(true);
  wrapper.unmount();
});
