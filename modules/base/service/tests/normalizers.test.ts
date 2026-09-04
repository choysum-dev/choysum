// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToBase,
  normalizeCodeOptional,
  assertCodeRequired,
  assertCurrencySymbolPosition,
  assertCurrencySymbolSpacing,
  assertDirection,
  assertName,
  assertRatePolicyMode,
  assertRoundingMode,
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
// mapNormalizationToBase
// ---------------------------------------------------------------------------

test('base._normalizers: mapNormalizationToBase maps normalization errors to base InvalidArgument', () => {
  let err: unknown;
  try {
    mapNormalizationToBase(
      () => {
        throw new NormalizationError('required');
      },
      () => 'Mapped required message'
    );
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('base');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('Mapped required message');
});

test('base._normalizers: mapNormalizationToBase passes through non-normalization errors', () => {
  const boom = new Error('boom');
  let err: unknown;
  try {
    mapNormalizationToBase(
      () => {
        throw boom;
      },
      () => 'ignored'
    );
  } catch (e) {
    err = e;
  }
  expect(err).toBe(boom);
});

// ---------------------------------------------------------------------------
// assertCodeRequired
// ---------------------------------------------------------------------------

test('base._normalizers: assertCodeRequired trim + uppercase', () => {
  expect(assertCodeRequired('  abc  ')).toBe('ABC');
});

test('base._normalizers: assertCodeRequired empty throws', () => {
  expect(() => assertCodeRequired('')).toThrow();
  expect(() => assertCodeRequired('   ')).toThrow();
});

test('base._normalizers: assertCodeRequired null/undefined throws', () => {
  expect(() => assertCodeRequired(null)).toThrow();
  expect(() => assertCodeRequired(undefined)).toThrow();
});

test('base._normalizers: assertCodeRequired with uppercase=false preserves case', () => {
  expect(assertCodeRequired('  MyCode  ', { uppercase: false })).toBe('MyCode');
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
// assertName
// ---------------------------------------------------------------------------

test('base._normalizers: assertName trim', () => {
  expect(assertName('  Hello World  ')).toBe('Hello World');
});

test('base._normalizers: assertName empty throws', () => {
  expect(() => assertName('')).toThrow();
  expect(() => assertName('   ')).toThrow();
});

test('base._normalizers: assertName null/undefined throws', () => {
  expect(() => assertName(null)).toThrow();
  expect(() => assertName(undefined)).toThrow();
});

// ---------------------------------------------------------------------------
// option validators (from _option_validators via _normalizers barrel)
// ---------------------------------------------------------------------------

test('base._normalizers: assertDirection invalid throws InvalidArgument', () => {
  expect(() => assertDirection('bogus')).toThrow('Direction must be ltr or rtl');
});

test('base._normalizers: assertDirection allows omit and clear', () => {
  expect(assertDirection(undefined)).toBeUndefined();
  expect(assertDirection(null)).toBeNull();
  expect(assertDirection('ltr')).toBe('ltr');
});

test('base._normalizers: assertCurrencySymbolPosition invalid throws', () => {
  expect(() => assertCurrencySymbolPosition('bogus')).toThrow('CurrencySymbolPosition must be before or after');
});

test('base._normalizers: assertCurrencySymbolPosition does not wash empty to before', () => {
  expect(assertCurrencySymbolPosition(undefined)).toBeUndefined();
  expect(assertCurrencySymbolPosition(null)).toBeNull();
  expect(assertCurrencySymbolPosition('')).toBeNull();
});

test('base._normalizers: assertCurrencySymbolSpacing rejects non-boolean', () => {
  expect(() => assertCurrencySymbolSpacing('yes')).toThrow('CurrencySymbolSpacing must be a boolean');
  expect(assertCurrencySymbolSpacing(true)).toBe(true);
  expect(assertCurrencySymbolSpacing(false)).toBe(false);
});

test('base._normalizers: assertRatePolicyMode invalid throws', () => {
  expect(() => assertRatePolicyMode('bogus')).toThrow('RatePolicy.Mode must be exact or latest_before');
});

test('base._normalizers: assertRatePolicyMode does not wash omit to latest_before', () => {
  expect(assertRatePolicyMode(undefined)).toBeUndefined();
});

test('base._normalizers: assertRoundingMode invalid throws', () => {
  expect(() => assertRoundingMode('bogus')).toThrow('Rounding.Mode must be currency or none');
});
