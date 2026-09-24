// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  formatChoyMonetary,
  parseChoyNumber,
  resolveChoyFieldVisible,
  resolveChoyMonetaryPrecision,
  resolveChoyNumberDraftText,
  roundChoyDecimal,
} from './fieldHelpers';

describe('fieldHelpers', () => {
  test('resolveChoyFieldVisible defaults to true', () => {
    expect(resolveChoyFieldVisible()).toBe(true);
    expect(resolveChoyFieldVisible(undefined)).toBe(true);
    expect(resolveChoyFieldVisible(true)).toBe(true);
    expect(resolveChoyFieldVisible(false)).toBe(false);
  });

  test('resolveChoyMonetaryPrecision defaults and clamps', () => {
    expect(resolveChoyMonetaryPrecision()).toBe(2);
    expect(resolveChoyMonetaryPrecision(-1)).toBe(2);
    expect(resolveChoyMonetaryPrecision(0)).toBe(0);
    expect(resolveChoyMonetaryPrecision(2.9)).toBe(2);
    expect(resolveChoyMonetaryPrecision(1000)).toBe(100);
    expect(resolveChoyMonetaryPrecision(Number.NaN)).toBe(2);
  });

  test('roundChoyDecimal uses decimal half-away-from-zero', () => {
    expect(roundChoyDecimal('1.005', 2)).toEqual({ value: 1.01, text: '1.01' });
    expect(roundChoyDecimal('-1.005', 2)).toEqual({ value: -1.01, text: '-1.01' });
    expect(roundChoyDecimal('0.995', 2)).toEqual({ value: 1, text: '1.00' });
    expect(roundChoyDecimal('-0.000', 2)).toEqual({ value: 0, text: '0.00' });
    expect(roundChoyDecimal('12.3', 2)).toEqual({ value: 12.3, text: '12.30' });
    expect(roundChoyDecimal('bad', 2)).toBeNull();
    // Full carry past the leading digit inserts a new high-place 1.
    expect(roundChoyDecimal('9.5', 0)).toEqual({ value: 10, text: '10' });
    expect(roundChoyDecimal('9.999', 2)).toEqual({ value: 10, text: '10.00' });
    // Lossy integer magnitudes (beyond MAX_SAFE_INTEGER) are rejected.
    expect(roundChoyDecimal('9007199254740993', 0)).toBeNull();
    // Overflowing digit strings become non-finite after Number(...).
    expect(roundChoyDecimal('9'.repeat(400), 0)).toBeNull();
  });

  test('formatChoyMonetary formats precision and currency', () => {
    expect(formatChoyMonetary(null)).toBe('');
    expect(formatChoyMonetary(undefined)).toBe('');
    expect(formatChoyMonetary('')).toBe('');
    expect(formatChoyMonetary('bad')).toBe('');
    expect(formatChoyMonetary(Number.POSITIVE_INFINITY)).toBe('');
    expect(formatChoyMonetary(12.345)).toBe('12.35');
    expect(formatChoyMonetary(2.675)).toBe('2.68');
    expect(formatChoyMonetary('2.675')).toBe('2.68');
    expect(formatChoyMonetary('12.3', { precision: 2 })).toBe('12.30');
    expect(formatChoyMonetary('1.005', { precision: 2 })).toBe('1.01');
    expect(formatChoyMonetary(12.3, { precision: 0 })).toBe('12');
    // precision clamps to 100 → "12.3" + 99 trailing zeros
    expect(formatChoyMonetary(12.3, { precision: 1000 })).toBe(`12.3${'0'.repeat(99)}`);
    expect(formatChoyMonetary(12.3, { precision: 2, currency: 'USD' })).toBe('12.30 USD');
    expect(formatChoyMonetary(0)).toBe('0.00');
    expect(formatChoyMonetary(-0)).toBe('0.00');
    // Exponential string: Number() may re-canonicalize into a decimal digit string.
    expect(formatChoyMonetary('2e0', { precision: 2 })).toBe('2.00');
    expect(formatChoyMonetary('1.005e0', { precision: 2 })).toBe('1.01');
    // Exponential string / number still needing toFixed when String(numeric) is not decimal.
    expect(formatChoyMonetary('1e-7', { precision: 2 })).toBe('0.00');
    expect(formatChoyMonetary(1e21, { precision: 2 })).toBe('1e+21');
    // Non-decimal literals are rejected (Number('0x10') is finite but not monetary text).
    expect(formatChoyMonetary('0x10')).toBe('');
  });

  test('resolveChoyNumberDraftText keeps preferred when String is exponential', () => {
    const tiny = Number('0.0000000000000000000001');
    expect(String(tiny)).toMatch(/e/i);
    expect(parseChoyNumber(String(tiny), 'float')).toBeNull();
    expect(
      resolveChoyNumberDraftText(tiny, '0.0000000000000000000001', 'float'),
    ).toBe('0.0000000000000000000001');
    // Preferred text that does not parse back to `value` is ignored.
    expect(resolveChoyNumberDraftText(tiny, '   ', 'float')).toBe(String(tiny));
    expect(resolveChoyNumberDraftText(tiny, 'not-a-number', 'float')).toBe(String(tiny));
    expect(resolveChoyNumberDraftText(12, '12', 'integer')).toBe('12');
  });

  test('parseChoyNumber modes', () => {
    expect(parseChoyNumber('', 'integer')).toBeNull();
    expect(parseChoyNumber('  ', 'float')).toBeNull();
    expect(parseChoyNumber('12', 'integer')).toBe(12);
    expect(parseChoyNumber('-3', 'integer')).toBe(-3);
    expect(parseChoyNumber('12.5', 'integer')).toBeNull();
    expect(parseChoyNumber('9007199254740994', 'integer')).toBeNull();
    expect(parseChoyNumber('12.5', 'float')).toBe(12.5);
    expect(parseChoyNumber('12.50', 'decimal')).toBe(12.5);
    expect(parseChoyNumber('abc', 'float')).toBeNull();
    expect(parseChoyNumber('1e2', 'float')).toBeNull();
  });
});
