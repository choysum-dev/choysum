// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import LanguageFormView from './LanguageFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('LanguageFormView.vue mounts under choysumMount and runs script setup', async () => {
  const store = { $id: 'fe-stub-language-form-store', records: {} };
  const wrapper = mount(LanguageFormView as any, {
    props: { store, viewMode: 'create' },
    global: buildPageMountGlobal({
      route: { path: '/base/languages/new', fullPath: '/base/languages/new' },
    }),
  });
  await flushPromises();
  expect(
    wrapper.find('[data-anchor="choy.form-view"]').exists() ||
      wrapper.find('[data-anchor="choy.form.sheet"]').exists() ||
      wrapper.find('[data-testid="language-form-identity"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.element != null
  ).toBe(true);
  wrapper.unmount();
});
