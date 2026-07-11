// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToPartnerCommercial,
  normalizeOptionalRefId,
  normalizeOptionalText,
  normalizeRequiredText,
  toDateOrUndefined,
} from '@/partner_commercial/service/models/_normalization_bridge';

// ---------------------------------------------------------------------------
// fail
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: fail throws ChoysumError with domain=partner_commercial code=InvalidArgument', () => {
  let err: unknown;
  try {
    fail('something wrong');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('partner_commercial');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('something wrong');
});

// ---------------------------------------------------------------------------
// mapNormalizationToPartnerCommercial
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: mapNormalizationToPartnerCommercial maps normalization errors', () => {
  let err: unknown;
  try {
    mapNormalizationToPartnerCommercial(
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
  expect(ce.domain).toBe('partner_commercial');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('Mapped required message');
});

test('partner_commercial._normalization_bridge: mapNormalizationToPartnerCommercial passes through non-normalization errors', () => {
  const boom = new Error('boom');
  let err: unknown;
  try {
    mapNormalizationToPartnerCommercial(
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

test('partner_commercial._normalization_bridge: mapNormalizationToPartnerCommercial returns result on success', () => {
  expect(
    mapNormalizationToPartnerCommercial(
      () => 'ok',
      () => 'ignored'
    )
  ).toBe('ok');
});

// ---------------------------------------------------------------------------
// normalizeOptionalText
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: normalizeOptionalText returns undefined for undefined', () => {
  expect(normalizeOptionalText(undefined)).toBe(undefined);
});

test('partner_commercial._normalization_bridge: normalizeOptionalText returns null for null', () => {
  expect(normalizeOptionalText(null)).toBe(null);
});

test('partner_commercial._normalization_bridge: normalizeOptionalText returns null for empty', () => {
  expect(normalizeOptionalText('')).toBe(null);
  expect(normalizeOptionalText('   ')).toBe(null);
});

test('partner_commercial._normalization_bridge: normalizeOptionalText trims value', () => {
  expect(normalizeOptionalText('  abc  ')).toBe('abc');
});

test('partner_commercial._normalization_bridge: normalizeOptionalText lowercases with lower option', () => {
  expect(normalizeOptionalText('  ABC  ', { lower: true })).toBe('abc');
});

test('partner_commercial._normalization_bridge: normalizeOptionalText uppercases with upper option', () => {
  expect(normalizeOptionalText('  abc  ', { upper: true })).toBe('ABC');
});

// ---------------------------------------------------------------------------
// normalizeOptionalRefId
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: normalizeOptionalRefId returns undefined for undefined', () => {
  expect(normalizeOptionalRefId(undefined)).toBe(undefined);
});

test('partner_commercial._normalization_bridge: normalizeOptionalRefId returns null for null', () => {
  expect(normalizeOptionalRefId(null)).toBe(null);
});

test('partner_commercial._normalization_bridge: normalizeOptionalRefId returns null for empty', () => {
  expect(normalizeOptionalRefId('')).toBe(null);
  expect(normalizeOptionalRefId('   ')).toBe(null);
});

test('partner_commercial._normalization_bridge: normalizeOptionalRefId resolves object Id', () => {
  expect(normalizeOptionalRefId({ Id: '123' })).toBe('123');
  expect(normalizeOptionalRefId({ id: '456' })).toBe('456');
});

test('partner_commercial._normalization_bridge: normalizeOptionalRefId trims string', () => {
  expect(normalizeOptionalRefId('  abc  ')).toBe('abc');
});

// ---------------------------------------------------------------------------
// normalizeRequiredText
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: normalizeRequiredText trims value', () => {
  expect(normalizeRequiredText('  Value  ', 'Value')).toBe('Value');
});

test('partner_commercial._normalization_bridge: normalizeRequiredText throws for empty', () => {
  expect(() => normalizeRequiredText('', 'Value')).toThrow();
  expect(() => normalizeRequiredText('   ', 'Value')).toThrow();
  expect(() => normalizeRequiredText(undefined, 'Value')).toThrow();
  expect(() => normalizeRequiredText(null, 'Value')).toThrow();
});

test('partner_commercial._normalization_bridge: normalizeRequiredText error message includes fieldName', () => {
  let err: unknown;
  try {
    normalizeRequiredText('', 'IdentifierType');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('IdentifierType is required');
});

test('partner_commercial._normalization_bridge: normalizeRequiredText lowercases with lower option', () => {
  expect(normalizeRequiredText('  ABC  ', 'Field', { lower: true })).toBe('abc');
});

test('partner_commercial._normalization_bridge: normalizeRequiredText uppercases with upper option', () => {
  expect(normalizeRequiredText('  abc  ', 'Field', { upper: true })).toBe('ABC');
});

// ---------------------------------------------------------------------------
// toDateOrUndefined
// ---------------------------------------------------------------------------

test('partner_commercial._normalization_bridge: toDateOrUndefined returns undefined for undefined/null/empty', () => {
  expect(toDateOrUndefined(undefined, 'ValidFrom')).toBe(undefined);
  expect(toDateOrUndefined(null, 'ValidFrom')).toBe(undefined);
  expect(toDateOrUndefined('', 'ValidFrom')).toBe(undefined);
});

test('partner_commercial._normalization_bridge: toDateOrUndefined returns Date instance as-is', () => {
  const d = new Date('2024-06-15T12:00:00Z');
  expect(toDateOrUndefined(d, 'ValidFrom')).toBe(d);
});

test('partner_commercial._normalization_bridge: toDateOrUndefined parses ISO string', () => {
  const result = toDateOrUndefined('2024-06-15T12:00:00Z', 'ValidFrom');
  expect(result instanceof Date).toBe(true);
  expect(result?.toISOString()).toBe('2024-06-15T12:00:00.000Z');
});

test('partner_commercial._normalization_bridge: toDateOrUndefined throws for invalid string', () => {
  let err: unknown;
  try {
    toDateOrUndefined('invalid', 'ValidFrom');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('ValidFrom must be a valid datetime');
});

test('partner_commercial._normalization_bridge: toDateOrUndefined throws for NaN Date instance', () => {
  let err: unknown;
  try {
    toDateOrUndefined(new Date('invalid'), 'ValidTo');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('ValidTo must be a valid datetime');
});
