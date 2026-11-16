// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  previewCurrency,
  previewDate,
  previewNumber,
  previewPersonName,
  previewTime,
  GROUPING_SELECTION,
  FIRST_DAY_SELECTION,
  PERSON_NAME_SELECTION,
  isLanguageCodeReadonly,
  LANGUAGE_LIST_COLUMN_PROPS,
} from './languageFormHelpers';

test('languageFormHelpers: number/currency previews follow separators and grouping', () => {
  const source = {
    DecimalSeparator: ',',
    ThousandSeparator: '.',
    Grouping: '[3,0]',
    CurrencySymbolPosition: 'before' as const,
    CurrencySymbolSpacing: false,
  };
  expect(previewNumber(source)).toBe('1.234.567,89');
  expect(previewCurrency(source)).toBe('¥1.234.567,89');
  expect(previewCurrency({ ...source, CurrencySymbolPosition: 'after', CurrencySymbolSpacing: true })).toBe(
    '1.234.567,89 ¥'
  );
});

test('languageFormHelpers: date/time/person-name previews', () => {
  expect(previewDate({ DateFormat: 'DD/MM/YYYY' })).toMatch(/15\/03\/2026/);
  expect(previewTime({ TimeFormat: 'HH:mm:ss' })).toMatch(/\d{2}:\d{2}:\d{2}/);
  expect(previewPersonName({ PersonNameFormat: '{LastName}{FirstName}' })).toBe('LovelaceAda');
  expect(previewPersonName({ PersonNameFormat: '{FirstName} {LastName}' })).toBe('Ada Lovelace');
});

test('languageFormHelpers: selection option catalogs', () => {
  expect(GROUPING_SELECTION).toContain('[3,0]');
  expect(GROUPING_SELECTION).toContain('[]');
  expect(FIRST_DAY_SELECTION).toEqual(['0', '1', '2', '3', '4', '5', '6']);
  expect(PERSON_NAME_SELECTION).toContain('{LastName}{FirstName}');
  expect(LANGUAGE_LIST_COLUMN_PROPS).toContain('DateFormat');
  expect(isLanguageCodeReadonly('create')).toBe(false);
  expect(isLanguageCodeReadonly('edit')).toBe(true);
});
