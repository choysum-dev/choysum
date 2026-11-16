// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createRelativeTimeFormatter,
  formatCurrency,
  formatDateTime,
  formatNumber,
  formatRelativeFromDiffMs,
  getDateTimeFormats,
  getNumberFormats,
  matchBestLang,
  pickRelativeUnit,
  readPreferredLanguageTags,
  resolveRelativeTargetMs,
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
    longDate: 'YYYY.MM.DD',
    shortTime: 'HH:mm',
    longTime: 'HH:mm:ss',
  };
  const d = new Date(2026, 0, 15, 9, 30, 5);
  expect(formatDateTime(d, config, { type: 'date' })).toBe('2026-01-15');
  expect(formatDateTime(d, config, { type: 'date', isLong: true })).toBe('2026.01.15');
  expect(formatDateTime(d, config, { type: 'time' })).toBe('09:30');
  expect(formatDateTime(d, config, { type: 'time', isLong: true })).toBe('09:30:05');
});

test('formatDateTime: datetime uses timezone conversion path', () => {
  const config = {
    shortDate: 'YYYY-MM-DD',
    shortTime: 'HH:mm',
  };
  const utcInstant = '2024-07-01T12:00:00.000Z';
  const out = formatDateTime(utcInstant, config, {
    type: 'datetime',
    timeZone: 'Asia/Shanghai',
  });
  expect(String(out)).toMatch(/2024-07-01/);
});

test('pickRelativeUnit: maps millisecond deltas to units', () => {
  const dayMs = 86_400_000;
  expect(pickRelativeUnit(-30_000).unit).toBe('second');
  expect(pickRelativeUnit(-120_000).unit).toBe('minute');
  expect(pickRelativeUnit(-3_600_000).unit).toBe('hour');
  expect(pickRelativeUnit(-dayMs).unit).toBe('day');
  expect(pickRelativeUnit(-7 * dayMs).unit).toBe('week');
  expect(pickRelativeUnit(-35 * dayMs).unit).toBe('month');
  expect(pickRelativeUnit(-400 * dayMs).unit).toBe('year');
  expect(pickRelativeUnit(45_000).value).toBeGreaterThan(0);
});

test('formatRelativeFromDiffMs: uses injected formatter and invalid-tag fallback', () => {
  const calls: string[] = [];
  const out = formatRelativeFromDiffMs(-60_000, 'xx-BAD', tag => {
    calls.push(tag);
    if (tag !== 'en-US') {
      throw new Error('bad tag');
    }
    return { format: (value, unit) => `${value}:${unit}` };
  });
  expect(calls).toEqual(['xx-BAD', 'en-US']);
  expect(out).toMatch(/minute/);
});

test('formatRelativeFromDiffMs: plain fallback when formatter always fails', () => {
  const out = formatRelativeFromDiffMs(-120_000, '', () => {
    throw new Error('no intl');
  });
  expect(out).toMatch(/ago/);
});

test('formatRelativeFromDiffMs: default factory throws without RelativeTimeFormat support', () => {
  const RTF = (Intl as any).RelativeTimeFormat;
  let threw = false;
  try {
    createRelativeTimeFormatter('en-US');
  } catch {
    threw = true;
  }
  if (typeof RTF !== 'function') {
    expect(threw).toBe(true);
  }
  const out = formatRelativeFromDiffMs(90_000, 'en-US');
  expect(String(out)).not.toBe('');
});

test('createRelativeTimeFormatter: constructs when RelativeTimeFormat is polyfilled', () => {
  const prev = (Intl as any).RelativeTimeFormat;
  (Intl as any).RelativeTimeFormat = class {
    tag: string;
    constructor(tag: string) {
      this.tag = tag;
    }
    format(value: number, unit: string) {
      return `${this.tag}:${value}:${unit}`;
    }
  };
  try {
    expect(createRelativeTimeFormatter('en-US').format(-1, 'minute')).toBe('en-US:-1:minute');
    expect(formatRelativeFromDiffMs(-60_000, 'fr-FR')).toMatch(/^fr-FR:/);
  } finally {
    (Intl as any).RelativeTimeFormat = prev;
  }
});

test('resolveRelativeTargetMs: Date / number / invalid string paths', () => {
  const fromDate = resolveRelativeTargetMs(new Date(Date.UTC(2024, 0, 1)), 'Bad/Zone');
  expect(Number.isFinite(fromDate)).toBe(true);
  const fromNumber = resolveRelativeTargetMs(1_700_000_000_000, 'Bad/Zone');
  expect(fromNumber).toBe(1_700_000_000_000);
  const fromInvalid = resolveRelativeTargetMs('not-an-iso', 'Bad/Zone');
  expect(Number.isFinite(fromInvalid)).toBe(true);
});

test('formatDateTime: relative works without tz when parseUtc/tz throws', () => {
  const past = Date.now() - 120_000;
  expect(String(formatDateTime(past, {}, { type: 'relative', bcp47: 'en-US' }))).not.toBe('');
  expect(
    String(
      formatDateTime('not-an-iso', {}, { type: 'relative', bcp47: 'en-US', timeZone: 'Bad/Zone' }),
    ),
  ).not.toBe('');
  expect(
    String(formatDateTime(new Date(past), {}, { type: 'relative', bcp47: 'en-US', timeZone: 'Bad/Zone' })),
  ).not.toBe('');
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

test('matchBestLang: empty preferred tag mainLang continues to next', () => {
  expect(matchBestLang(['', 'en-us'], ['de_DE', 'en_US'])).toBe('en_US');
});

test('readPreferredLanguageTags: languages array, language fallback, missing navigator, and throw', () => {
  const desc = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  const setNav = (value: unknown) => {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      enumerable: true,
      writable: true,
      value,
    });
  };
  try {
    setNav({ languages: ['zh-CN', 'en-US'], language: 'en-US' });
    expect(readPreferredLanguageTags()).toEqual(['zh-CN', 'en-US']);

    setNav({ languages: [], language: 'de-DE' });
    expect(readPreferredLanguageTags()).toEqual(['de-DE']);

    setNav(undefined);
    expect(readPreferredLanguageTags()).toEqual([]);

    setNav(
      new Proxy(
        {},
        {
          get() {
            throw new Error('nav boom');
          },
        },
      ),
    );
    expect(readPreferredLanguageTags()).toEqual([]);
  } finally {
    if (desc) {
      Object.defineProperty(globalThis, 'navigator', desc);
    }
  }
});
