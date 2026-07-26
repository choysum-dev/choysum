// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { SUPPORTED_LOCALES } from './locales';
import {
  formatCurrencyFromConfig,
  formatNumberFromConfig,
  formatFixedDecimalString,
  parseGrouping,
  resolveFormatConfig,
} from './language_format';
import { formatDateTime, formatNumber } from './utils';

describe('language_format (P2)', () => {
  it('T2.2: formatNumber uses Language separators and Grouping, not catalog hardcoding', () => {
    const language = {
      DecimalSeparator: ',',
      ThousandSeparator: '.',
      Grouping: '[3,0]',
      DateFormat: 'DD/MM/YYYY',
      TimeFormat: 'HH:mm',
    };
    const catalog = SUPPORTED_LOCALES['zh-CN'];
    const resolved = resolveFormatConfig(catalog.numberFormat, catalog.dateTimeFormat, language, null);

    expect(formatNumber(1234567.89, 'zh-CN', resolved.numberFormat, { digits: 2 })).toBe('1.234.567,89');
    expect(formatNumberFromConfig(1234567.89, resolved.numberFormat, { digits: 2 })).toBe('1.234.567,89');
    // Catalog alone would use ',' thousands — Language overlay wins.
    expect(catalog.numberFormat?.thousandsSeparator).toBe(',');
    expect(resolved.numberFormat.thousandsSeparator).toBe('.');
  });

  it('T2.3: Preferences.display.dateFormat overrides Language; unset keeps Language', () => {
    const language = {
      DateFormat: 'YYYY-MM-DD',
      TimeFormat: 'HH:mm:ss',
    };
    const withOverride = resolveFormatConfig(null, null, language, { dateFormat: 'DD.MM.YYYY' });
    expect(withOverride.dateTimeFormat.shortDate).toBe('DD.MM.YYYY');
    expect(withOverride.dateTimeFormat.shortTime).toBe('HH:mm:ss');

    const without = resolveFormatConfig(null, null, language, null);
    expect(without.dateTimeFormat.shortDate).toBe('YYYY-MM-DD');

    const date = new Date(2026, 0, 2, 15, 4, 5);
    expect(formatDateTime(date, withOverride.dateTimeFormat, { type: 'date' })).toBe('02.01.2026');
    expect(formatDateTime(date, without.dateTimeFormat, { type: 'date' })).toBe('2026-01-02');
  });

  it('datetime uses user timezone; date does not shift calendar day', () => {
    const config = { shortDate: 'YYYY-MM-DD', shortTime: 'HH:mm:ss' };
    const utcInstant = '2024-06-30T16:00:00.000Z'; // 2024-07-01 00:00 Shanghai

    expect(
      formatDateTime(utcInstant, config, { type: 'datetime', timeZone: 'Asia/Shanghai' })
    ).toBe('2024-07-01 00:00:00');
    expect(
      formatDateTime(utcInstant, config, { type: 'datetime', timeZone: 'America/New_York' })
    ).toBe('2024-06-30 12:00:00');

    // date type stays literal (local dayjs of the Date/string), not re-zoned as a business datetime.
    const calendar = new Date(2024, 6, 1); // July 1 local
    expect(formatDateTime(calendar, config, { type: 'date', timeZone: 'America/New_York' })).toBe('2024-07-01');
  });

  it('T2.4: catalog still exposes Element/dayjs package names; missing format falls back safely', () => {
    const zh = SUPPORTED_LOCALES['zh-CN'];
    expect(zh.elementLocaleCode).toBe('zh-cn');
    expect(zh.dayjsLocaleCode).toBe('zh-cn');

    const sparse = SUPPORTED_LOCALES.el;
    expect(sparse.elementLocaleCode).toBe('el');
    expect(sparse.numberFormat).toBeUndefined();

    const resolved = resolveFormatConfig(sparse.numberFormat, sparse.dateTimeFormat, null, null);
    expect(() => formatNumber(1234.5, 'el', resolved.numberFormat, { digits: 1 })).not.toThrow();
    expect(formatNumber(1234.5, 'el', resolved.numberFormat, { digits: 1 })).toBe('1,234.5');
  });

  it('parseGrouping accepts Odoo-style [3,0]', () => {
    expect(parseGrouping('[3,0]')).toEqual([3, 0]);
    expect(parseGrouping([3, 2, 0])).toEqual([3, 2, 0]);
  });

  it('formats fixed decimal strings without Number conversion', () => {
    expect(
      formatFixedDecimalString('1234567.890000000000000000', {
        thousandsSeparator: '.',
        decimalSeparator: ',',
        grouping: [3, 0],
      })
    ).toBe('1.234.567,890000000000000000');
  });

  it('formats currency from a pre-quantized string without Number conversion', () => {
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

  it('pads currency string fractional digits to decimalDigits without Number conversion', () => {
    expect(
      formatCurrencyFromConfig('1.2', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        position: 'after',
        spacing: true,
      }, 'USD')
    ).toBe('1.20 USD');

    expect(
      formatCurrencyFromConfig('-5', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 3,
        symbol: '$',
        position: 'before',
        spacing: false,
      })
    ).toBe('$-5.000');

    // Longer tails stay intact (precision-safe).
    expect(
      formatCurrencyFromConfig('1.2345', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
      })
    ).toBe('1.2345');
  });

  it('covers ensureDecimalDigitsString edge cases for currency formatting', () => {
    expect(formatCurrencyFromConfig('  ', { decimalDigits: 2 }, 'USD')).toBe('USD');
    expect(formatCurrencyFromConfig('', { decimalDigits: 2 })).toBe('');

    expect(
      formatCurrencyFromConfig('12', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 0,
      }, 'USD')
    ).toBe('USD12');

    expect(
      formatCurrencyFromConfig('-12', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 0,
      })
    ).toBe('-12');

    expect(
      formatCurrencyFromConfig('12.30', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 0,
      }, 'USD')
    ).toBe('USD12.30');

    expect(
      formatCurrencyFromConfig('1.2', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: -1 as any,
      })
    ).toBe('1.2');

    expect(
      formatCurrencyFromConfig('1.2', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 1.5 as any,
      })
    ).toBe('1.2');

    expect(
      formatCurrencyFromConfig('1.2', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
      })
    ).toBe('1.20');

    expect(
      formatCurrencyFromConfig(12.5, {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        position: 'before',
        spacing: true,
      }, 'USD')
    ).toBe('USD 12.50');

    expect(
      formatCurrencyFromConfig('.5', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        spacing: true,
      }, 'USD')
    ).toBe('USD 0.50');

    expect(
      formatCurrencyFromConfig('-.5', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
        decimalDigits: 2,
        symbol: '$',
        position: 'before',
        spacing: false,
      })
    ).toBe('$-0.50');
  });
});
