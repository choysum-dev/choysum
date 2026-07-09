// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToBase,
  normalizeCodeOptional,
  normalizeCodeRequired,
  normalizeCurrencySymbolPosition,
  normalizeDirection,
  normalizeName,
  normalizeRatePolicyMode,
  normalizeRoundingMode,
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
// option normalizers (from _option_normalizers via _normalizers barrel)
// ---------------------------------------------------------------------------

test('base._normalizers: normalizeDirection invalid throws InvalidArgument', () => {
  expect(() => normalizeDirection('bogus')).toThrow('Direction must be ltr or rtl');
});

test('base._normalizers: normalizeCurrencySymbolPosition invalid throws', () => {
  expect(() => normalizeCurrencySymbolPosition('bogus')).toThrow('CurrencySymbolPosition must be before or after');
});

test('base._normalizers: normalizeRatePolicyMode invalid throws', () => {
  expect(() => normalizeRatePolicyMode('bogus')).toThrow('RatePolicy.Mode must be exact or latest_before');
});

test('base._normalizers: normalizeRoundingMode invalid throws', () => {
  expect(() => normalizeRoundingMode('bogus')).toThrow('Rounding.Mode must be currency or none');
});
