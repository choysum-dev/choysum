// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  GROUPING_SELECTION,
  FIRST_DAY_SELECTION,
  isLanguageCodeReadonly,
  LANGUAGE_LIST_COLUMN_PROPS,
} from './languageFormHelpers';

test('languageFormHelpers: selection option catalogs', () => {
  expect(GROUPING_SELECTION).toContain('[3,0]');
  expect(GROUPING_SELECTION).toContain('[]');
  expect(FIRST_DAY_SELECTION).toEqual(['0', '1', '2', '3', '4', '5', '6']);
  expect(LANGUAGE_LIST_COLUMN_PROPS).toContain('DateFormat');
  expect(isLanguageCodeReadonly('create')).toBe(false);
  expect(isLanguageCodeReadonly('edit')).toBe(true);
});
