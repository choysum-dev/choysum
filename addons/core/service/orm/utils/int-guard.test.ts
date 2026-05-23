// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { assertSafeInt } from './int-guard';

test('assertSafeInt accepts nullish and safe integer values', () => {
  assertSafeInt(undefined, 'A');
  assertSafeInt(null, 'A');
  assertSafeInt(0, 'A');
  assertSafeInt(-1, 'A');
  assertSafeInt(Number.MAX_SAFE_INTEGER, 'A');
  expect(true).toBe(true);
});

test('assertSafeInt rejects non-number and non-integer values', () => {
  expect(() => assertSafeInt('1' as any, 'Qty')).toThrow('Field Qty expects integer');
  expect(() => assertSafeInt(1.2, 'Qty')).toThrow('Field Qty expects integer');
});

test('assertSafeInt rejects unsafe integer values', () => {
  expect(() => assertSafeInt(Number.MAX_SAFE_INTEGER + 1, 'Amount')).toThrow('Field Amount exceeds safe integer range');
  expect(() => assertSafeInt(Number.MIN_SAFE_INTEGER - 1, 'Amount')).toThrow('Field Amount exceeds safe integer range');
});
