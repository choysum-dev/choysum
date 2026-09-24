// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  formatChoyMonetary,
  parseChoyNumber,
  resolveChoyFieldVisible,
} from './fieldHelpers';

describe('fieldHelpers', () => {
  test('resolveChoyFieldVisible defaults to true', () => {
    expect(resolveChoyFieldVisible()).toBe(true);
    expect(resolveChoyFieldVisible(undefined)).toBe(true);
    expect(resolveChoyFieldVisible(true)).toBe(true);
    expect(resolveChoyFieldVisible(false)).toBe(false);
  });

  test('formatChoyMonetary formats precision and currency', () => {
    expect(formatChoyMonetary(null)).toBe('');
    expect(formatChoyMonetary(undefined)).toBe('');
    expect(formatChoyMonetary('')).toBe('');
    expect(formatChoyMonetary('bad')).toBe('');
    expect(formatChoyMonetary(Number.POSITIVE_INFINITY)).toBe('');
    expect(formatChoyMonetary(12.345)).toBe('12.35');
    expect(formatChoyMonetary('12.3', { precision: 2 })).toBe('12.30');
    expect(formatChoyMonetary(12.3, { precision: 0 })).toBe('12');
    expect(formatChoyMonetary(12.3, { precision: 1000 })).toBe(
      (12.3).toFixed(100),
    );
    expect(formatChoyMonetary(12.3, { precision: 2, currency: 'USD' })).toBe('12.30 USD');
    expect(formatChoyMonetary(0)).toBe('0.00');
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
