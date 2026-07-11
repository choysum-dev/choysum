// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToPartner,
  normalizeOptionalText,
  normalizeRequiredText,
  normalizeNonNegativeInt,
  normalizeSequenceInt,
} from '@/partner/service/models/_normalization_bridge';

// ---------------------------------------------------------------------------
// fail
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: fail throws ChoysumError with domain=partner code=InvalidArgument', () => {
  let err: unknown;
  try {
    fail('something wrong');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('partner');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('something wrong');
});

// ---------------------------------------------------------------------------
// mapNormalizationToPartner
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: mapNormalizationToPartner maps NormalizationError to partner InvalidArgument', () => {
  let err: unknown;
  try {
    mapNormalizationToPartner(
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
  expect(ce.domain).toBe('partner');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('Mapped required message');
});

test('partner._normalization_bridge: mapNormalizationToPartner returns value on success', () => {
  const result = mapNormalizationToPartner(
    () => 'hello',
    () => 'unused'
  );
  expect(result).toBe('hello');
});

test('partner._normalization_bridge: mapNormalizationToPartner passes through non-NormalizationError', () => {
  const original = new Error('unrelated');
  let err: unknown;
  try {
    mapNormalizationToPartner(
      () => {
        throw original;
      },
      () => 'unused'
    );
  } catch (e) {
    err = e;
  }
  expect(err).toBe(original);
});

// ---------------------------------------------------------------------------
// normalizeOptionalText
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: normalizeOptionalText returns undefined for undefined', () => {
  expect(normalizeOptionalText(undefined)).toBeUndefined();
});

test('partner._normalization_bridge: normalizeOptionalText returns null for null', () => {
  expect(normalizeOptionalText(null)).toBeNull();
});

test('partner._normalization_bridge: normalizeOptionalText returns null for empty string', () => {
  expect(normalizeOptionalText('')).toBeNull();
});

test('partner._normalization_bridge: normalizeOptionalText returns null for whitespace-only', () => {
  expect(normalizeOptionalText('   ')).toBeNull();
});

test('partner._normalization_bridge: normalizeOptionalText trims surrounding whitespace', () => {
  expect(normalizeOptionalText('  hello  ')).toBe('hello');
});

test('partner._normalization_bridge: normalizeOptionalText uppercases when upper option set', () => {
  expect(normalizeOptionalText('hello', { upper: true })).toBe('HELLO');
});

test('partner._normalization_bridge: normalizeOptionalText lowercases when lower option set', () => {
  expect(normalizeOptionalText('HELLO', { lower: true })).toBe('hello');
});

test('partner._normalization_bridge: normalizeOptionalText upper takes precedence over lower', () => {
  expect(normalizeOptionalText('Hello', { upper: true, lower: true })).toBe('HELLO');
});

test('partner._normalization_bridge: normalizeOptionalText returns unchanged when no options', () => {
  expect(normalizeOptionalText('Hello')).toBe('Hello');
});

// ---------------------------------------------------------------------------
// normalizeRequiredText
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: normalizeRequiredText returns trimmed string', () => {
  expect(normalizeRequiredText('  hello  ', 'Name')).toBe('hello');
});

test('partner._normalization_bridge: normalizeRequiredText throws with field name on empty', () => {
  let err: unknown;
  try {
    normalizeRequiredText('', 'Name');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('Name is required');
});

test('partner._normalization_bridge: normalizeRequiredText throws with field name on whitespace', () => {
  let err: unknown;
  try {
    normalizeRequiredText('   ', 'Code');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('Code is required');
});

test('partner._normalization_bridge: normalizeRequiredText throws with custom field name', () => {
  let err: unknown;
  try {
    normalizeRequiredText('', 'CustomField');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('CustomField is required');
});

// ---------------------------------------------------------------------------
// normalizeNonNegativeInt
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: normalizeNonNegativeInt returns undefined for undefined', () => {
  expect(normalizeNonNegativeInt(undefined, 'Rank')).toBeUndefined();
});

test('partner._normalization_bridge: normalizeNonNegativeInt returns 0 for null', () => {
  expect(normalizeNonNegativeInt(null, 'Rank')).toBe(0);
});

test('partner._normalization_bridge: normalizeNonNegativeInt returns 0 for zero', () => {
  expect(normalizeNonNegativeInt(0, 'Rank')).toBe(0);
});

test('partner._normalization_bridge: normalizeNonNegativeInt returns positive integer', () => {
  expect(normalizeNonNegativeInt(42, 'Rank')).toBe(42);
});

test('partner._normalization_bridge: normalizeNonNegativeInt parses numeric string', () => {
  expect(normalizeNonNegativeInt('5', 'Rank')).toBe(5);
});

test('partner._normalization_bridge: normalizeNonNegativeInt throws for negative', () => {
  let err: unknown;
  try {
    normalizeNonNegativeInt(-1, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: normalizeNonNegativeInt throws for non-integer', () => {
  let err: unknown;
  try {
    normalizeNonNegativeInt(3.5, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: normalizeNonNegativeInt throws for NaN', () => {
  let err: unknown;
  try {
    normalizeNonNegativeInt(NaN, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: normalizeNonNegativeInt throws for Infinity', () => {
  let err: unknown;
  try {
    normalizeNonNegativeInt(Infinity, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

// ---------------------------------------------------------------------------
// normalizeSequenceInt
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: normalizeSequenceInt returns undefined for undefined', () => {
  expect(normalizeSequenceInt(undefined)).toBeUndefined();
});

test('partner._normalization_bridge: normalizeSequenceInt returns 10 for null (default)', () => {
  expect(normalizeSequenceInt(null)).toBe(10);
});

test('partner._normalization_bridge: normalizeSequenceInt returns 10 for empty string (default)', () => {
  expect(normalizeSequenceInt('')).toBe(10);
});

test('partner._normalization_bridge: normalizeSequenceInt returns 10 for whitespace string (default)', () => {
  expect(normalizeSequenceInt('   ')).toBe(10);
});

test('partner._normalization_bridge: normalizeSequenceInt returns positive integer', () => {
  expect(normalizeSequenceInt(5)).toBe(5);
});

test('partner._normalization_bridge: normalizeSequenceInt allows negative integer', () => {
  expect(normalizeSequenceInt(-3)).toBe(-3);
});

test('partner._normalization_bridge: normalizeSequenceInt parses numeric string', () => {
  expect(normalizeSequenceInt('7')).toBe(7);
});

test('partner._normalization_bridge: normalizeSequenceInt throws for non-integer', () => {
  let err: unknown;
  try {
    normalizeSequenceInt(2.5);
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Sequence must be an integer');
});

test('partner._normalization_bridge: normalizeSequenceInt throws for NaN', () => {
  let err: unknown;
  try {
    normalizeSequenceInt('abc');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Sequence must be an integer');
});

test('partner._normalization_bridge: normalizeSequenceInt throws for Infinity', () => {
  let err: unknown;
  try {
    normalizeSequenceInt(Infinity);
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Sequence must be an integer');
});
