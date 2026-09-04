// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import {
  fail,
  mapNormalizationToPartnerCommercial,
  normalizeOptionalRefId,
  normalizeOptionalText,
  normalizeOptionalTranslatedText,
  assertRequiredText,
  assertDateOrUndefined,
} from '@/partner_commercial/service/models/_partner_commercial_bridge';

// ---------------------------------------------------------------------------
// fail
// ---------------------------------------------------------------------------

test('partner_commercial._partner_commercial_bridge: fail throws ChoysumError with domain=partner_commercial code=InvalidArgument', () => {
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

test('partner_commercial._partner_commercial_bridge: mapNormalizationToPartnerCommercial maps normalization errors', () => {
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

test('partner_commercial._partner_commercial_bridge: mapNormalizationToPartnerCommercial passes through non-normalization errors', () => {
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

test('partner_commercial._partner_commercial_bridge: mapNormalizationToPartnerCommercial returns result on success', () => {
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

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText returns undefined for undefined', () => {
  expect(normalizeOptionalText(undefined)).toBe(undefined);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText returns null for null', () => {
  expect(normalizeOptionalText(null)).toBe(null);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText returns null for empty', () => {
  expect(normalizeOptionalText('')).toBe(null);
  expect(normalizeOptionalText('   ')).toBe(null);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText trims value', () => {
  expect(normalizeOptionalText('  abc  ')).toBe('abc');
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText lowercases with lower option', () => {
  expect(normalizeOptionalText('  ABC  ', { lower: true })).toBe('abc');
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalText uppercases with upper option', () => {
  expect(normalizeOptionalText('  abc  ', { upper: true })).toBe('ABC');
});

// ---------------------------------------------------------------------------
// normalizeOptionalRefId
// ---------------------------------------------------------------------------

test('partner_commercial._partner_commercial_bridge: normalizeOptionalRefId returns undefined for undefined', () => {
  expect(normalizeOptionalRefId(undefined)).toBe(undefined);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalRefId returns null for null', () => {
  expect(normalizeOptionalRefId(null)).toBe(null);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalRefId returns null for empty', () => {
  expect(normalizeOptionalRefId('')).toBe(null);
  expect(normalizeOptionalRefId('   ')).toBe(null);
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalRefId resolves object Id', () => {
  expect(normalizeOptionalRefId({ Id: '123' })).toBe('123');
  expect(normalizeOptionalRefId({ id: '456' })).toBe('456');
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalRefId trims string', () => {
  expect(normalizeOptionalRefId('  abc  ')).toBe('abc');
});

// ---------------------------------------------------------------------------
// assertRequiredText
// ---------------------------------------------------------------------------

test('partner_commercial._partner_commercial_bridge: assertRequiredText trims value', () => {
  expect(assertRequiredText('  Value  ', 'Value')).toBe('Value');
});

test('partner_commercial._partner_commercial_bridge: assertRequiredText throws for empty', () => {
  expect(() => assertRequiredText('', 'Value')).toThrow();
  expect(() => assertRequiredText('   ', 'Value')).toThrow();
  expect(() => assertRequiredText(undefined, 'Value')).toThrow();
  expect(() => assertRequiredText(null, 'Value')).toThrow();
});

test('partner_commercial._partner_commercial_bridge: assertRequiredText error message includes fieldName', () => {
  let err: unknown;
  try {
    assertRequiredText('', 'IdentifierType');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('IdentifierType is required');
});

test('partner_commercial._partner_commercial_bridge: assertRequiredText lowercases with lower option', () => {
  expect(assertRequiredText('  ABC  ', 'Field', { lower: true })).toBe('abc');
});

test('partner_commercial._partner_commercial_bridge: assertRequiredText uppercases with upper option', () => {
  expect(assertRequiredText('  abc  ', 'Field', { upper: true })).toBe('ABC');
});

// ---------------------------------------------------------------------------
// assertDateOrUndefined
// ---------------------------------------------------------------------------

test('partner_commercial._partner_commercial_bridge: assertDateOrUndefined returns undefined for undefined/null/empty', () => {
  expect(assertDateOrUndefined(undefined, 'ValidFrom')).toBe(undefined);
  expect(assertDateOrUndefined(null, 'ValidFrom')).toBe(undefined);
  expect(assertDateOrUndefined('', 'ValidFrom')).toBe(undefined);
});

test('partner_commercial._partner_commercial_bridge: assertDateOrUndefined returns Date instance as-is', () => {
  const d = new Date('2024-06-15T12:00:00Z');
  expect(assertDateOrUndefined(d, 'ValidFrom')).toBe(d);
});

test('partner_commercial._partner_commercial_bridge: assertDateOrUndefined parses ISO string', () => {
  const result = assertDateOrUndefined('2024-06-15T12:00:00Z', 'ValidFrom');
  expect(result instanceof Date).toBe(true);
  expect(result?.toISOString()).toBe('2024-06-15T12:00:00.000Z');
});

test('partner_commercial._partner_commercial_bridge: assertDateOrUndefined throws for invalid string', () => {
  let err: unknown;
  try {
    assertDateOrUndefined('invalid', 'ValidFrom');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('ValidFrom must be a valid datetime');
});

test('partner_commercial._partner_commercial_bridge: assertDateOrUndefined throws for NaN Date instance', () => {
  let err: unknown;
  try {
    assertDateOrUndefined(new Date('invalid'), 'ValidTo');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('ValidTo must be a valid datetime');
});

test('partner_commercial._partner_commercial_bridge: normalizeOptionalTranslatedText accepts lang maps', () => {
  expect(normalizeOptionalTranslatedText(undefined)).toBeUndefined();
  expect(normalizeOptionalTranslatedText({ en_US: ' A ', zh_CN: '  ' })).toEqual({ en_US: 'A', zh_CN: '' });
  expect(normalizeOptionalTranslatedText(' Solo ')).toBe('Solo');
});
