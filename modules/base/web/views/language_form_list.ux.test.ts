// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import LanguageFormView from './LanguageFormView.vue';
import LanguageListView from './LanguageListView.vue';
import {
  FIRST_DAY_SELECTION,
  GROUPING_SELECTION,
  isLanguageCodeReadonly,
  LANGUAGE_LIST_COLUMN_PROPS,
} from './languageFormHelpers';

test('Language Code is readonly outside create mode', () => {
  expect(isLanguageCodeReadonly('create')).toBe(false);
  expect(isLanguageCodeReadonly('edit')).toBe(true);
  expect(isLanguageCodeReadonly('display')).toBe(true);
  expect(isLanguageCodeReadonly(undefined)).toBe(true);
});

test('Language form helpers expose selection catalogs for closed enums', () => {
  expect(FIRST_DAY_SELECTION).toEqual(['0', '1', '2', '3', '4', '5', '6']);
  expect(GROUPING_SELECTION).toContain('[3,0]');
  expect(GROUPING_SELECTION).toContain('[]');
  expect(LANGUAGE_LIST_COLUMN_PROPS).toEqual(['Name', 'Code', 'Direction', 'IsActive', 'DateFormat']);
});

test('LanguageFormView and LanguageListView components are defined', () => {
  expect(LanguageFormView).toBeTruthy();
  expect(LanguageListView).toBeTruthy();
});

test('Language list column contract includes DateFormat as a listed prop', () => {
  // Keep list/search UX aligned with LANGUAGE_LIST_COLUMN_PROPS (Name/Code/Direction/IsActive/DateFormat).
  expect([...LANGUAGE_LIST_COLUMN_PROPS]).toContain('DateFormat');
  expect([...LANGUAGE_LIST_COLUMN_PROPS]).not.toContain('FirstDayOfWeek');
});
