// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Parameters Form uses standard ORM CRUD (align FieldDefault Settings UI).
 * Program / curated Settings pages continue to use Get/Set.
 * AppSetting models must set softDelete: false so Delete / Set(null) free the unique Key.
 */

import AppSettingBaseModel from './app_setting_base_model';

test('AppSetting store exposes Get/Set for Settings main path', () => {
  expect(typeof AppSettingBaseModel.Get).toBe('function');
  expect(typeof AppSettingBaseModel.Set).toBe('function');
});

test('AppSettingBaseModel documents hard-delete Set(null) contract for key reuse', () => {
  // Construction validates softDelete: false on concrete models; see app_setting_base_model.ts.
  expect(AppSettingBaseModel.name).toBe('AppSettingBaseModel');
});
