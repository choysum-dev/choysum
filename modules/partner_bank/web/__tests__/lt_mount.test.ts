// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import PartnerBankAccountFormView from '../views/PartnerBankAccountFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('PartnerBankAccountFormView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-bank-store', records: {} };
  const wrapper = mount(PartnerBankAccountFormView as any, {
    props: { store, viewMode: 'create' },
    global: buildPageMountGlobal({ route: { path: '/partner/bank', fullPath: '/partner/bank' } }),
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="fe-stub-opage"]').exists() || wrapper.find('[data-testid="fe-stub-child-view"]').exists()).toBe(true);
  wrapper.unmount();
});
