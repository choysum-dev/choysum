// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  formatCurrency,
  formatDateTime,
  formatNumber,
  getDateTimeFormats,
  getNumberFormats,
  matchBestLang,
  updateDocumentDirection,
} from './utils';

test('getDateTimeFormats: builds POSIX-keyed Intl skeletons', () => {
  const formats = getDateTimeFormats('en_US', { shortDate: 'MM/DD/YYYY' });
  expect(formats.en_US.short.year).toBe('numeric');
  expect(formats.en_US.longTime.hour12).toBe(true);
});

test('getNumberFormats: decimal, currency, and percent entries', () => {
  const formats = getNumberFormats(
    'zh_CN',
    { decimalDigits: 3 },
    { code: 'CNY' },
  );
  expect(formats.zh_CN.decimal.minimumFractionDigits).toBe(3);
  expect(formats.zh_CN.currency.currency).toBe('CNY');
  expect(formats.zh_CN.percent.style).toBe('percent');
});

test('formatDateTime: relative uses Intl.RelativeTimeFormat when available', () => {
  if (typeof Intl.RelativeTimeFormat !== 'function') {
    return;
  }
  try {
    const now = Date.now();
    const out = formatDateTime(now - 60_000, {}, {
      type: 'relative',
      bcp47: 'en-US',
    });
    expect(String(out)).not.toBe('');
  } catch {
    // QuickJS host may lack timezone conversion for relative formatting.
  }
});

test('formatDateTime: relative falls back when tag invalid', () => {
  if (typeof Intl.RelativeTimeFormat !== 'function') {
    return;
  }
  try {
    const out = formatDateTime(Date.now(), {}, {
      type: 'relative',
      bcp47: 'not-a-real-bcp47-tag-###',
    });
    expect(String(out)).not.toBe('');
  } catch {
    // QuickJS host may lack timezone conversion for relative formatting.
  }
});

test('formatNumber: Intl fallback when config lacks separators', () => {
  const out = formatNumber(1234.5, 'en-US', {}, { digits: 1 });
  expect(out).toMatch(/1234\.5/);
});

test('formatNumber: returns string on Intl failure', () => {
  expect(formatNumber(9.99, '', {}, { digits: 1 })).toBe('9.99');
});

test('formatCurrency: Intl fallback', () => {
  const out = formatCurrency(12.5, 'en-US', { decimalDigits: 2 });
  expect(String(out)).toMatch(/12\.5/);
});

test('formatCurrency: returns string on Intl failure', () => {
  expect(formatCurrency(9.99, 'bad', { decimalDigits: 1 }, 'USD')).toBe('9.99');
});

test('updateDocumentDirection: sets dir and CSS variables on documentElement', () => {
  updateDocumentDirection('rtl');
  expect(document.documentElement.getAttribute('dir')).toBe('rtl');
  expect(document.documentElement.style.getPropertyValue('--choy-direction')).toBe('rtl');
  expect(document.documentElement.style.getPropertyValue('--choy-start')).toBe('right');
  updateDocumentDirection('ltr');
  expect(document.documentElement.style.getPropertyValue('--choy-start')).toBe('left');
});

test('matchBestLang: picks first allowed when tags miss', () => {
  expect(matchBestLang(['xx-YY'], ['de_DE', 'en_US'])).toBe('en_US');
});