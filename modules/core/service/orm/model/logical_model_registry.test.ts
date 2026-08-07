// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  __resetLogicalModelNamesForTest,
  isRegisteredLogicalModelName,
  listLogicalModelNames,
  listLogicalModelSelection,
  registerLogicalModelName,
} from './logical_model_registry';

// Side-effect: platform inject bases self-register on import.
import './app_setting_base_model';
import './field_default_base_model';
import './translation_term_base_model';

test('platform inject bases self-register logical model short names', () => {
  expect(listLogicalModelNames()).toEqual(['AppSetting', 'FieldDefault', 'TranslationTerm']);
  expect(isRegisteredLogicalModelName('TranslationTerm')).toBe(true);
  expect(isRegisteredLogicalModelName('Partner')).toBe(false);
  expect(isRegisteredLogicalModelName('')).toBe(false);
});

test('listLogicalModelSelection mirrors registered names for FieldsGet', () => {
  expect(listLogicalModelSelection()).toEqual([
    { value: 'AppSetting', label: 'AppSetting' },
    { value: 'FieldDefault', label: 'FieldDefault' },
    { value: 'TranslationTerm', label: 'TranslationTerm' },
  ]);
});

test('registerLogicalModelName is idempotent and ignores blanks', () => {
  registerLogicalModelName('  ');
  registerLogicalModelName('AppSetting');
  expect(listLogicalModelNames()).toEqual(['AppSetting', 'FieldDefault', 'TranslationTerm']);
});

test('__resetLogicalModelNamesForTest clears registry for isolation', () => {
  __resetLogicalModelNamesForTest();
  expect(listLogicalModelNames()).toEqual([]);
  registerLogicalModelName('TmpLogical');
  expect(isRegisteredLogicalModelName('TmpLogical')).toBe(true);
  // Restore platform names for sibling tests in the same process.
  __resetLogicalModelNamesForTest();
  registerLogicalModelName('AppSetting');
  registerLogicalModelName('FieldDefault');
  registerLogicalModelName('TranslationTerm');
  expect(listLogicalModelNames()).toEqual(['AppSetting', 'FieldDefault', 'TranslationTerm']);
});
