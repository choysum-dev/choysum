// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import AuthSettings from './AuthSettings.vue';
import { setAuthSettingsAppSettingStoreForTests } from './auth_settings_page';

test('AuthSettings page mounts with injected AppSetting store', async () => {
  setAuthSettingsAppSettingStoreForTests(() => ({
    Get: async (_key, def) => def ?? '1',
    Set: async () => null,
  }));
  try {
    const wrapper = mount(AuthSettings as any, {
      global: buildPageMountGlobal({
        route: {
          path: '/auth/settings',
          fullPath: '/auth/settings',
          name: 'AuthSettings',
          meta: {
            viewKind: 'settings',
            viewPaths: {
              settings: '/auth/settings',
              list: '/web/app-settings/auth',
            },
          },
        },
      }),
    });
    await flushPromises();
    expect(
      wrapper.find('[data-anchor="auth.settings.sections"]').exists() ||
        wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
        wrapper.find('[data-anchor="choy.page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-child-view"]').exists(),
    ).toBe(true);
    wrapper.unmount();
  } finally {
    setAuthSettingsAppSettingStoreForTests(null);
  }
});
