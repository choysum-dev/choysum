// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToPartner,
  normalizeOptionalText,
  assertRequiredText,
  assertRequiredTranslatedText,
  normalizeOptionalTranslatedText,
  translatedTextHasValue,
  assertNonNegativeInt,
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
// assertRequiredText
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: assertRequiredText returns trimmed string', () => {
  expect(assertRequiredText('  hello  ', 'Name')).toBe('hello');
});

test('partner._normalization_bridge: assertRequiredText throws with field name on empty', () => {
  let err: unknown;
  try {
    assertRequiredText('', 'Name');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('Name is required');
});

test('partner._normalization_bridge: assertRequiredText throws with field name on whitespace', () => {
  let err: unknown;
  try {
    assertRequiredText('   ', 'Code');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('Code is required');
});

test('partner._normalization_bridge: assertRequiredText throws with custom field name', () => {
  let err: unknown;
  try {
    assertRequiredText('', 'CustomField');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.message).toBe('CustomField is required');
});

// ---------------------------------------------------------------------------
// assertNonNegativeInt
// ---------------------------------------------------------------------------

test('partner._normalization_bridge: assertNonNegativeInt returns undefined for undefined', () => {
  expect(assertNonNegativeInt(undefined, 'Rank')).toBeUndefined();
});

test('partner._normalization_bridge: assertNonNegativeInt returns 0 for null', () => {
  expect(assertNonNegativeInt(null, 'Rank')).toBe(0);
});

test('partner._normalization_bridge: assertNonNegativeInt returns 0 for zero', () => {
  expect(assertNonNegativeInt(0, 'Rank')).toBe(0);
});

test('partner._normalization_bridge: assertNonNegativeInt returns positive integer', () => {
  expect(assertNonNegativeInt(42, 'Rank')).toBe(42);
});

test('partner._normalization_bridge: assertNonNegativeInt parses numeric string', () => {
  expect(assertNonNegativeInt('5', 'Rank')).toBe(5);
});

test('partner._normalization_bridge: assertNonNegativeInt throws for negative', () => {
  let err: unknown;
  try {
    assertNonNegativeInt(-1, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: assertNonNegativeInt throws for non-integer', () => {
  let err: unknown;
  try {
    assertNonNegativeInt(3.5, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: assertNonNegativeInt throws for NaN', () => {
  let err: unknown;
  try {
    assertNonNegativeInt(NaN, 'Rank');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Rank must be a non-negative integer');
});

test('partner._normalization_bridge: assertNonNegativeInt throws for Infinity', () => {
  let err: unknown;
  try {
    assertNonNegativeInt(Infinity, 'Rank');
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

test('partner._normalization_bridge: assertRequiredTranslatedText accepts lang maps', () => {
  expect(assertRequiredTranslatedText({ en_US: ' Acme ', zh_CN: '' }, 'Name')).toEqual({
    en_US: 'Acme',
    zh_CN: '',
  });
  expect(assertRequiredTranslatedText(' Solo ', 'Name')).toBe('Solo');
});

test('partner._normalization_bridge: assertRequiredTranslatedText rejects empty maps', () => {
  let err: unknown;
  try {
    assertRequiredTranslatedText({ en_US: '  ', zh_CN: '' }, 'Name');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Name is required');
});

test('partner._normalization_bridge: normalizeOptionalTranslatedText and translatedTextHasValue', () => {
  expect(normalizeOptionalTranslatedText(undefined)).toBeUndefined();
  expect(normalizeOptionalTranslatedText({ en_US: ' A ', zh_CN: '  ' })).toEqual({ en_US: 'A', zh_CN: '' });
  expect(normalizeOptionalTranslatedText({ en_US: 'A', zh_CN: '' })).toEqual({ en_US: 'A', zh_CN: '' });
  expect(translatedTextHasValue('x')).toBe(true);
  expect(translatedTextHasValue({ en_US: '', zh_CN: '甲' })).toBe(true);
  expect(translatedTextHasValue({ en_US: '', zh_CN: '' })).toBe(false);
});
