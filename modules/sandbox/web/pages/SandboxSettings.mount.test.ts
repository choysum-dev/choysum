// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import SandboxSettings from './SandboxSettings.vue';
import { setSandboxSettingsAppSettingStoreForTests } from './sandbox_settings_page';

test('SandboxSettings page mounts with injected AppSetting store', async () => {
  setSandboxSettingsAppSettingStoreForTests(() => ({
    Get: async (_key, def) => def ?? '',
    Set: async () => null,
  }));
  try {
    const wrapper = mount(SandboxSettings as any, {
      global: buildPageMountGlobal({
        route: {
          path: '/sandbox/settings',
          fullPath: '/sandbox/settings',
          name: 'SandboxSettings',
          meta: {
            viewKind: 'settings',
            viewPaths: {
              settings: '/sandbox/settings',
              list: '/web/app-settings/sandbox',
            },
          },
        },
      }),
    });
    await flushPromises();
    expect(
      wrapper.find('[data-anchor="sandbox.settings.sections"]').exists() ||
        wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
        wrapper.find('[data-anchor="choy.page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-child-view"]').exists(),
    ).toBe(true);
    wrapper.unmount();
  } finally {
    setSandboxSettingsAppSettingStoreForTests(null);
  }
});
