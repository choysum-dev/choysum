// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  META_MODULE_STATUS_INSTALLED,
  collectModuleNamesFromMetaModuleRows,
  installedModulesSearchCondition,
} from './terminologyExportModules';

test('collectModuleNamesFromMetaModuleRows: distinct sorted Name', () => {
  expect(
    collectModuleNamesFromMetaModuleRows([
      { Name: 'web' },
      { Name: 'auth' },
      { Name: 'web' },
      { Name: '  ' },
      {},
      null,
    ])
  ).toEqual(['auth', 'web']);
});

test('collectModuleNamesFromMetaModuleRows: empty / non-array', () => {
  expect(collectModuleNamesFromMetaModuleRows(null)).toEqual([]);
  expect(collectModuleNamesFromMetaModuleRows(undefined)).toEqual([]);
  expect(collectModuleNamesFromMetaModuleRows([])).toEqual([]);
});

test('installedModulesSearchCondition: ApplicationStr + installed', () => {
  expect(installedModulesSearchCondition('auth')).toEqual({
    And: [
      ['ApplicationStr', '=', 'auth'],
      ['Status', '=', META_MODULE_STATUS_INSTALLED],
    ],
  });
  expect(META_MODULE_STATUS_INSTALLED).toBe('installed');
});
