// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import LanguageFormView from './LanguageFormView.vue';
import LanguageListView from './LanguageListView.vue';
import {
  DATE_FORMAT_PRESETS,
  FIRST_DAY_SELECTION,
  GROUPING_SELECTION,
  isLanguageCodeReadonly,
  LANGUAGE_LIST_COLUMN_PROPS,
  PERSON_NAME_SELECTION,
  TIME_FORMAT_PRESETS,
} from './languageFormHelpers';

test('Language Code is readonly outside create mode', () => {
  expect(isLanguageCodeReadonly('create')).toBe(false);
  expect(isLanguageCodeReadonly('edit')).toBe(true);
  expect(isLanguageCodeReadonly('display')).toBe(true);
  expect(isLanguageCodeReadonly(undefined)).toBe(true);
});

test('Language form helpers expose FirstDay/Grouping/PersonName/DateTime presets', () => {
  expect(FIRST_DAY_SELECTION).toEqual(['0', '1', '2', '3', '4', '5', '6']);
  expect(GROUPING_SELECTION).toContain('[3,0]');
  expect(GROUPING_SELECTION).toContain('[]');
  expect(PERSON_NAME_SELECTION).toContain('{LastName}{FirstName}');
  expect(DATE_FORMAT_PRESETS).toContain('YYYY-MM-DD');
  expect(TIME_FORMAT_PRESETS).toContain('HH:mm:ss');
  expect(LANGUAGE_LIST_COLUMN_PROPS).toEqual(['Name', 'Code', 'Direction', 'IsActive', 'DateFormat']);
});

test('LanguageFormView and LanguageListView components are defined', () => {
  expect(LanguageFormView).toBeTruthy();
  expect(LanguageListView).toBeTruthy();
});
