// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import PartnerIdentifierFormView from '../views/PartnerIdentifierFormView.vue';
import { buildPageMountGlobal } from '../testing/page_mount';

test('lt_mount: mounts real PartnerIdentifierFormView under choysumMount', async () => {
  const store = { $id: 'fe-stub-id-store', records: {} };
  const wrapper = mount(PartnerIdentifierFormView as any, {
    props: { store, viewMode: 'create' },
    global: buildPageMountGlobal(),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  wrapper.unmount();
});
