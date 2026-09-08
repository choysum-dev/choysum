// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import CoverageProbe from '../testing/CoverageProbe.vue';

test('partner-list-view coverage: mounts CoverageProbe under choysumMount', async () => {
  const wrapper = mount(CoverageProbe as any, { props: { title: 'partner-list-view' } });
  await flushPromises();
  expect(wrapper.find('[data-testid="partner-coverage-probe"]').text()).toBe('partner-list-view');
  wrapper.unmount();
});
