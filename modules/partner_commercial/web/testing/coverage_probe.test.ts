// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import CoverageProbe from './CoverageProbe.vue';

test('CoverageProbe mount: runs Vue SFC script for partner_commercial FE lcov sampling', async () => {
  const wrapper = mount(CoverageProbe as any, { props: { title: 'probe' } });
  await flushPromises();
  expect(wrapper.find('[data-testid="partner-commercial-coverage-probe"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="partner-commercial-coverage-probe"]').text()).toBe('probe');
  wrapper.unmount();
});
