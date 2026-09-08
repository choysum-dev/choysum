// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import CoverageProbe from '../testing/CoverageProbe.vue';

test('lt_mount coverage: mounts CoverageProbe under choysumMount', async () => {
  const wrapper = mount(CoverageProbe as any, { props: { title: 'lt' } });
  await flushPromises();
  expect(wrapper.find('[data-testid="meta-coverage-probe"]').text()).toBe('lt');
  wrapper.unmount();
});
