// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  normalizeStringArray,
  readRefId,
  normalizeRefId,
  normalizeOffset,
  normalizeLimit,
  normalizeFields,
} from '@/core/service/utils/normalization';

test('normalizeOptionalString returns trimmed string or undefined', () => {
  expect(normalizeOptionalString('  hello  ')).toBe('hello');
  expect(normalizeOptionalString(null)).toBe(undefined);
  expect(normalizeOptionalString(undefined)).toBe(undefined);
  expect(normalizeOptionalString('')).toBe(undefined);
  expect(normalizeOptionalString('   ')).toBe(undefined);
  expect(normalizeOptionalString(123)).toBe('123');
});

test('normalizeStringArray deduplicates and filters empty strings', () => {
  expect(normalizeStringArray(null)).toEqual([]);
  expect(normalizeStringArray(undefined)).toEqual([]);
  expect(normalizeStringArray('not-an-array')).toEqual([]);
  expect(normalizeStringArray(123)).toEqual([]);
  expect(normalizeStringArray([])).toEqual([]);
  expect(normalizeStringArray(['a', 'b', 'a'])).toEqual(['a', 'b']);
  expect(normalizeStringArray(['  hello ', '', ' world ', '  hello  '])).toEqual(['hello', 'world']);
  expect(normalizeStringArray([null, undefined, '  valid  '])).toEqual(['valid']);
});

test('readRefId extracts id from string or object', () => {
  expect(readRefId(null)).toBe(undefined);
  expect(readRefId(undefined)).toBe(undefined);
  expect(readRefId('')).toBe(undefined);
  expect(readRefId(0)).toBe(undefined);
  expect(readRefId('  id123  ')).toBe('id123');
  expect(readRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(readRefId({ Id: '' })).toBe(undefined);
  expect(readRefId({ Id: null })).toBe(undefined);
  expect(readRefId({ id: 'lowercase' })).toBe(undefined);
  expect(readRefId(true)).toBe(undefined);
});

test('normalizeRefId returns trimmed string or null', () => {
  expect(normalizeRefId(null)).toBe(null);
  expect(normalizeRefId(undefined)).toBe(null);
  expect(normalizeRefId('')).toBe(null);
  expect(normalizeRefId(0)).toBe('0');
  expect(normalizeRefId('  id123  ')).toBe('id123');
  expect(normalizeRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(normalizeRefId({ id: 'lowercase' })).toBe('lowercase');
  expect(normalizeRefId({ Id: '', id: 'fallback' })).toBe('fallback');
  expect(normalizeRefId({ Id: '' })).toBe(null);
  expect(normalizeRefId(true)).toBe(null);
});

test('normalizeOffset returns non-negative floored integer', () => {
  expect(normalizeOffset(null)).toBe(0);
  expect(normalizeOffset(undefined)).toBe(0);
  expect(normalizeOffset(NaN)).toBe(0);
  expect(normalizeOffset(-1)).toBe(0);
  expect(normalizeOffset(-100)).toBe(0);
  expect(normalizeOffset(0)).toBe(0);
  expect(normalizeOffset(5)).toBe(5);
  expect(normalizeOffset(5.7)).toBe(5);
  expect(normalizeOffset('10')).toBe(10);
});

test('normalizeLimit returns positive floored integer or null', () => {
  expect(normalizeLimit(null)).toBe(null);
  expect(normalizeLimit(undefined)).toBe(null);
  expect(normalizeLimit(NaN)).toBe(null);
  expect(normalizeLimit(0)).toBe(null);
  expect(normalizeLimit(-1)).toBe(null);
  expect(normalizeLimit(-100)).toBe(null);
  expect(normalizeLimit(5)).toBe(5);
  expect(normalizeLimit(5.7)).toBe(5);
  expect(normalizeLimit('10')).toBe(10);
});

test('normalizeFields trims deduplicates and filters empty strings', () => {
  expect(normalizeFields(null)).toEqual([]);
  expect(normalizeFields(undefined)).toEqual([]);
  expect(normalizeFields('not-array')).toEqual([]);
  expect(normalizeFields([' Name ', '', ' Code ', ' name ', ''])).toEqual(['Name', 'Code', 'name']);
  expect(normalizeFields([])).toEqual([]);
});
