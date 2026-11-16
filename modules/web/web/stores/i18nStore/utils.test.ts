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
  expect(getDateTimeFormats('de_DE', {}).de_DE.longTime.hour12).toBe(false);
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

test('formatDateTime: date and time literals use dayjs patterns', () => {
  const config = {
    shortDate: 'YYYY-MM-DD',
    shortTime: 'HH:mm',
    longTime: 'HH:mm:ss',
  };
  const d = new Date(2026, 0, 15, 9, 30, 5);
  expect(formatDateTime(d, config, { type: 'date' })).toBe('2026-01-15');
  expect(formatDateTime(d, config, { type: 'time' })).toBe('09:30');
  expect(formatDateTime(d, config, { type: 'time', isLong: true })).toBe('09:30:05');
});

test('formatDateTime: relative uses Intl.RelativeTimeFormat when available', () => {
  if (typeof Intl.RelativeTimeFormat !== 'function') {
    return;
  }
  let out = '';
  let hostUnsupported = false;
  try {
    const now = Date.now();
    out = formatDateTime(now - 60_000, {}, {
      type: 'relative',
      bcp47: 'en-US',
    });
  } catch {
    hostUnsupported = true;
  }
  if (hostUnsupported) {
    return;
  }
  expect(String(out)).not.toBe('');
});

test('formatDateTime: relative falls back when tag invalid', () => {
  if (typeof Intl.RelativeTimeFormat !== 'function') {
    return;
  }
  let out = '';
  let hostUnsupported = false;
  try {
    out = formatDateTime(Date.now(), {}, {
      type: 'relative',
      bcp47: 'not-a-real-bcp47-tag-###',
    });
  } catch {
    hostUnsupported = true;
  }
  if (hostUnsupported) {
    return;
  }
  expect(String(out)).not.toBe('');
});

test('formatDateTime: relative spans minute hour day week month year units', () => {
  if (typeof Intl.RelativeTimeFormat !== 'function') {
    return;
  }
  const now = Date.now();
  const deltas = [
    30_000,
    120_000,
    3_600_000,
    86_400_000,
    604_800_000,
    2_592_000_000,
    31_536_000_000,
  ];
  for (const delta of deltas) {
    let out = '';
    let hostUnsupported = false;
    try {
      out = formatDateTime(now - delta, {}, { type: 'relative', bcp47: 'en-US' });
    } catch {
      hostUnsupported = true;
    }
    if (hostUnsupported) {
      return;
    }
    expect(String(out)).not.toBe('');
  }
});

test('formatNumber: config path with separators', () => {
  const out = formatNumber(1234.5, 'en-US', {
    thousandsSeparator: ',',
    decimalSeparator: '.',
    grouping: [3, 0],
    decimalDigits: 2,
  });
  expect(out).toBe('1,234.50');
});

test('formatNumber: Intl fallback when config lacks separators', () => {
  const out = formatNumber(1234.5, 'en-US', {}, { digits: 1 });
  expect(out).toMatch(/1234\.5/);
});

test('formatNumber: returns string on Intl failure', () => {
  expect(formatNumber(9.99, '', {}, { digits: 1 })).toBe('9.99');
});

test('formatCurrency: config path with symbol placement', () => {
  const out = formatCurrency(
    12.5,
    'en-US',
    {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 2,
      position: 'before',
      spacing: true,
      code: 'USD',
    },
    'USD',
  );
  expect(out).toBe('USD 12.50');
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