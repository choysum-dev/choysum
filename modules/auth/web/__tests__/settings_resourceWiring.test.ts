// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { authSettingsAppMenu } from '../menu/menus';
import { appRoutes } from '../route/routes';

test('auth resource wiring: AppSetting parameters list without curated Settings page', () => {
  const byName = Object.fromEntries(appRoutes.map(route => [String(route.name), route]));
  expect(byName.AppSettingList_auth?.path).toBe('web/app-settings/auth');
  expect(byName.AppSettingList_auth?.meta).toMatchObject({
    requiresAuth: true,
    appSettingApplication: 'auth',
    viewKind: 'list',
  });
  expect(byName.AuthSettings).toBeUndefined();
  expect(authSettingsAppMenu.children?.some(child => child.path === '/web/app-settings/auth')).toBe(
    true
  );
  expect(authSettingsAppMenu.children?.some(child => child.path === '/auth/settings')).toBe(false);
});
