// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  normalizeCodeRequired,
  normalizeCodeOptional,
  normalizeName,
  toPositiveDecimal,
  normalizePositiveDecimalString,
} from '@/base/service/models/_normalizers';

// ---------------------------------------------------------------------------
// fail
// ---------------------------------------------------------------------------

test('base._normalizers: fail throws ChoysumError with domain=base code=InvalidArgument', () => {
  let err: unknown;
  try {
    fail('something wrong');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('base');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('something wrong');
});

// ---------------------------------------------------------------------------
// normalizeCodeRequired
// ---------------------------------------------------------------------------

test('base._normalizers: normalizeCodeRequired trim + uppercase', () => {
  expect(normalizeCodeRequired('  abc  ')).toBe('ABC');
});

test('base._normalizers: normalizeCodeRequired empty throws', () => {
  expect(() => normalizeCodeRequired('')).toThrow();
  expect(() => normalizeCodeRequired('   ')).toThrow();
});

test('base._normalizers: normalizeCodeRequired null/undefined throws', () => {
  expect(() => normalizeCodeRequired(null)).toThrow();
  expect(() => normalizeCodeRequired(undefined)).toThrow();
});

test('base._normalizers: normalizeCodeRequired with uppercase=false preserves case', () => {
  expect(normalizeCodeRequired('  MyCode  ', { uppercase: false })).toBe('MyCode');
});

// ---------------------------------------------------------------------------
// normalizeCodeOptional
// ---------------------------------------------------------------------------

test('base._normalizers: normalizeCodeOptional undefined → undefined', () => {
  expect(normalizeCodeOptional(undefined)).toBeUndefined();
});

test('base._normalizers: normalizeCodeOptional null → null', () => {
  expect(normalizeCodeOptional(null)).toBeNull();
});

test('base._normalizers: normalizeCodeOptional empty → null', () => {
  expect(normalizeCodeOptional('   ')).toBeNull();
});

test('base._normalizers: normalizeCodeOptional trim + uppercase', () => {
  expect(normalizeCodeOptional('  xyz  ')).toBe('XYZ');
});

test('base._normalizers: normalizeCodeOptional with uppercase=false preserves case', () => {
  expect(normalizeCodeOptional('  XyZ  ', { uppercase: false })).toBe('XyZ');
});

// ---------------------------------------------------------------------------
// normalizeName
// ---------------------------------------------------------------------------

test('base._normalizers: normalizeName trim', () => {
  expect(normalizeName('  Hello World  ')).toBe('Hello World');
});

test('base._normalizers: normalizeName empty throws', () => {
  expect(() => normalizeName('')).toThrow();
  expect(() => normalizeName('   ')).toThrow();
});

test('base._normalizers: normalizeName null/undefined throws', () => {
  expect(() => normalizeName(null)).toThrow();
  expect(() => normalizeName(undefined)).toThrow();
});

// ---------------------------------------------------------------------------
// toPositiveDecimal
// ---------------------------------------------------------------------------

test('base._normalizers: toPositiveDecimal from string', () => {
  const d = toPositiveDecimal('3.14', 'Rate');
  expect(d instanceof Decimal).toBe(true);
  expect(d.eq(new Decimal('3.14'))).toBe(true);
});

test('base._normalizers: toPositiveDecimal from Decimal', () => {
  const input = new Decimal('5');
  const d = toPositiveDecimal(input, 'Factor');
  expect(d.eq(new Decimal('5'))).toBe(true);
});

test('base._normalizers: toPositiveDecimal zero throws', () => {
  expect(() => toPositiveDecimal('0', 'Rate')).toThrow();
});

test('base._normalizers: toPositiveDecimal negative throws', () => {
  expect(() => toPositiveDecimal('-1', 'Rate')).toThrow();
});

test('base._normalizers: toPositiveDecimal empty throws', () => {
  expect(() => toPositiveDecimal('', 'Rate')).toThrow();
});

test('base._normalizers: toPositiveDecimal null/undefined throws', () => {
  expect(() => toPositiveDecimal(null, 'Rate')).toThrow();
  expect(() => toPositiveDecimal(undefined, 'Rate')).toThrow();
});

test('base._normalizers: toPositiveDecimal unparseable throws', () => {
  expect(() => toPositiveDecimal('not-a-number', 'Rate')).toThrow();
});

// ---------------------------------------------------------------------------
// normalizePositiveDecimalString
// ---------------------------------------------------------------------------

test('base._normalizers: normalizePositiveDecimalString from string', () => {
  expect(normalizePositiveDecimalString('3.14', 'Rounding')).toBe('3.14');
});

test('base._normalizers: normalizePositiveDecimalString zero throws', () => {
  expect(() => normalizePositiveDecimalString('0', 'Rounding')).toThrow();
});

test('base._normalizers: normalizePositiveDecimalString empty throws', () => {
  expect(() => normalizePositiveDecimalString('', 'Rounding')).toThrow();
});

test('base._normalizers: normalizePositiveDecimalString null throws', () => {
  expect(() => normalizePositiveDecimalString(null, 'Rounding')).toThrow();
});

test('base._normalizers: normalizePositiveDecimalString from Decimal', () => {
  expect(normalizePositiveDecimalString(new Decimal('0.01'), 'Rounding')).toBe('0.01');
});
