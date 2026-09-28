// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import PartnerIdentifierFormView from './PartnerIdentifierFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('PartnerIdentifierFormView mount coverage', async () => {
  const store = { $id: 'fe-stub-id-store', records: {} };
  const wrapper = mount(PartnerIdentifierFormView as any, {
    props: { store, viewMode: 'create' },
    global: buildPageMountGlobal(),
  });
  await flushPromises();
  expect(
    wrapper.find('[data-anchor="choy.form-view"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-anchor="choy.page"]').exists(),
  ).toBe(true);
  wrapper.unmount();
});
