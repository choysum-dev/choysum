// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveAppSettingDetailApplication,
  resolveAppSettingListApplication,
} from './app_setting_page_bind';

test('resolveAppSettingListApplication prefers meta then path', () => {
  expect(
    resolveAppSettingListApplication({
      meta: { appSettingApplication: 'auth' },
      params: {},
      path: '/web/app-settings/web',
    })
  ).toBe('auth');
  expect(
    resolveAppSettingListApplication({
      meta: {},
      params: { application: 'partner' },
      path: '',
    })
  ).toBe('partner');
  expect(
    resolveAppSettingListApplication({
      meta: {},
      params: {},
      path: '/web/app-settings/sandbox',
    })
  ).toBe('sandbox');
});

test('resolveAppSettingDetailApplication reads params.application', () => {
  expect(resolveAppSettingDetailApplication({ params: { application: 'base' } })).toBe('base');
  expect(resolveAppSettingDetailApplication({ params: {} })).toBe('');
});
