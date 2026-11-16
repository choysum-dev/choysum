// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  collectActiveLanguages,
  formatCurrencyFromConfig,
  formatNumberFromConfig,
  formatFixedDecimalString,
  overlayFromActiveLanguageRow,
  parseGrouping,
  resolveFormatConfig,
} from './languageFormat';
import { formatDateTime, formatNumber } from './utils';

test('languageFormat (P2): T2.2: formatNumber uses Language separators and Grouping, not catalog hardcoding', () => {
  const language = {
    DecimalSeparator: ',',
    ThousandSeparator: '.',
    Grouping: '[3,0]',
    DateFormat: 'DD/MM/YYYY',
    TimeFormat: 'HH:mm',
  };
  const resolved = resolveFormatConfig(language);

  expect(formatNumber(1234567.89, 'de-DE', resolved.numberFormat, { digits: 2 })).toBe('1.234.567,89');
  expect(formatNumberFromConfig(1234567.89, resolved.numberFormat, { digits: 2 })).toBe('1.234.567,89');
});

test('languageFormat (P2): T2.3: Language DateFormat/TimeFormat drive resolveFormatConfig', () => {
  const language = {
    DateFormat: 'DD.MM.YYYY',
    TimeFormat: 'HH:mm:ss',
  };
  const resolved = resolveFormatConfig(language);
  expect(resolved.dateTimeFormat.shortDate).toBe('DD.MM.YYYY');
  expect(resolved.dateTimeFormat.shortTime).toBe('HH:mm:ss');

  const withCatalog = resolveFormatConfig({
    DateFormat: 'YYYY-MM-DD',
    TimeFormat: 'HH:mm',
  });
  expect(withCatalog.dateTimeFormat.shortDate).toBe('YYYY-MM-DD');

  const date = new Date(2026, 0, 2, 15, 4, 5);
  expect(formatDateTime(date, resolved.dateTimeFormat, { type: 'date' })).toBe('02.01.2026');
  expect(formatDateTime(date, withCatalog.dateTimeFormat, { type: 'date' })).toBe('2026-01-02');
});

test('languageFormat (P2): datetime uses user timezone; date does not shift calendar day', () => {
  const config = { shortDate: 'YYYY-MM-DD', shortTime: 'HH:mm:ss' };
  const utcInstant = '2024-06-30T16:00:00.000Z'; // 2024-07-01 00:00 Shanghai

  expect(
    formatDateTime(utcInstant, config, { type: 'datetime', timeZone: 'Asia/Shanghai' })
  ).toBe('2024-07-01 00:00:00');
  expect(
    formatDateTime(utcInstant, config, { type: 'datetime', timeZone: 'America/New_York' })
  ).toBe('2024-06-30 12:00:00');

  const calendar = new Date(2024, 6, 1);
  expect(formatDateTime(calendar, config, { type: 'date', timeZone: 'America/New_York' })).toBe('2024-07-01');
});

test('languageFormat (P2): T2.4: missing format falls back safely', () => {
  const resolved = resolveFormatConfig(null);
  expect(() => formatNumber(1234.5, 'en-US', resolved.numberFormat, { digits: 1 })).not.toThrow();
  expect(formatNumber(1234.5, 'en-US', resolved.numberFormat, { digits: 1 })).toBe('1,234.5');
});

test('languageFormat (P2): parseGrouping accepts Odoo-style [3,0]', () => {
  expect(parseGrouping('[3,0]')).toEqual([3, 0]);
  expect(parseGrouping([3, 2, 0])).toEqual([3, 2, 0]);
});

test('languageFormat (P2): formats fixed decimal strings without Number conversion', () => {
  expect(
    formatFixedDecimalString('1234567.890000000000000000', {
      thousandsSeparator: '.',
      decimalSeparator: ',',
      grouping: [3, 0],
    })
  ).toBe('1.234.567,890000000000000000');
});

test('languageFormat (P2): formats currency from a pre-quantized string without Number conversion', () => {
  expect(
    formatCurrencyFromConfig(
      '9007199254740993.12',
      {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        position: 'before',
        spacing: true,
      },
      'USD'
    )
  ).toBe('USD 9,007,199,254,740,993.12');
});

test('overlayFromActiveLanguageRow: maps Direction rtl and ltr', () => {
  expect(overlayFromActiveLanguageRow({ Direction: 'rtl' }).Direction).toBe('rtl');
  expect(overlayFromActiveLanguageRow({ Direction: 'LTR' }).Direction).toBe('ltr');
  expect(overlayFromActiveLanguageRow({ direction: 'rtl' }).Direction).toBe('rtl');
  expect(overlayFromActiveLanguageRow({}).Direction).toBeUndefined();
});

test('formatCurrencyFromConfig: pads fractional digits in string input', () => {
  expect(
    formatCurrencyFromConfig(
      '12.5',
      {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        position: 'before',
        spacing: false,
      },
      'USD',
    ),
  ).toBe('USD12.50');
});

test('overlayFromActiveLanguageRow: prefers PascalCase PersonNameFormat', () => {
  expect(
    overlayFromActiveLanguageRow({
      PersonNameFormat: '{LastName}{FirstName}',
      personNameFormat: 'ignored',
    }).PersonNameFormat,
  ).toBe('{LastName}{FirstName}');
  expect(
    overlayFromActiveLanguageRow({
      personNameFormat: '{FirstName} {LastName}',
    }).PersonNameFormat,
  ).toBe('{FirstName} {LastName}');
  expect(overlayFromActiveLanguageRow({}).PersonNameFormat).toBeUndefined();
});

test('collectActiveLanguages: maps overlays and keeps POSIX codes', () => {
  const { rows, byCode } = collectActiveLanguages([
    { Code: 'zh_CN', PersonNameFormat: '{LastName}{FirstName}' },
    { code: 'en_US', personNameFormat: '{FirstName} {LastName}' },
    { Code: 'xx_YY', PersonNameFormat: '{FirstName}' },
  ]);
  expect(byCode.zh_CN?.PersonNameFormat).toBe('{LastName}{FirstName}');
  expect(byCode.en_US?.PersonNameFormat).toBe('{FirstName} {LastName}');
  expect(rows.map(r => r.Code)).toEqual(['zh_CN', 'en_US', 'xx_YY']);
  expect(collectActiveLanguages(null).rows).toEqual([]);
});

test('collectActiveLanguages: skips empty Code and maps Name and Direction', () => {
  const { rows, byCode } = collectActiveLanguages([
    { Code: '' },
    { Code: 'ar_001', Direction: 'rtl', Name: 'Arabic' },
    { Code: 'bad_dir', Direction: 'sideways' },
  ]);
  expect(rows.map(r => r.Code)).toEqual(['ar_001', 'bad_dir']);
  expect(rows[0]?.Name).toBe('Arabic');
  expect(rows[0]?.Direction).toBe('rtl');
  expect(byCode.ar_001?.Name).toBe('Arabic');
  expect(byCode.bad_dir?.Direction).toBeUndefined();
});