// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { NormalizationError } from '@/core/service/utils/normalization';
import { ChoysumError } from '@/core/service/error';
import { fail, mapNormalizationToPartnerBank, normalizeOptionalText, assertRequiredText } from '@/partner_bank/service/models/_partner_bank_bridge';

// ---------------------------------------------------------------------------
// fail
// ---------------------------------------------------------------------------

test('partner_bank._partner_bank_bridge: fail throws ChoysumError with domain=partner_bank code=InvalidArgument', () => {
  let err: unknown;
  try {
    fail('something wrong');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('partner_bank');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('something wrong');
});

// ---------------------------------------------------------------------------
// mapNormalizationToPartnerBank
// ---------------------------------------------------------------------------

test('partner_bank._partner_bank_bridge: mapNormalizationToPartnerBank maps normalization errors', () => {
  let err: unknown;
  try {
    mapNormalizationToPartnerBank(
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
  expect(ce.domain).toBe('partner_bank');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('Mapped required message');
});

test('partner_bank._partner_bank_bridge: mapNormalizationToPartnerBank passes through non-normalization errors', () => {
  const boom = new Error('boom');
  let err: unknown;
  try {
    mapNormalizationToPartnerBank(
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

test('partner_bank._partner_bank_bridge: mapNormalizationToPartnerBank returns result on success', () => {
  const result = mapNormalizationToPartnerBank(
    () => 'ok',
    () => 'ignored'
  );
  expect(result).toBe('ok');
});

// ---------------------------------------------------------------------------
// normalizeOptionalText
// ---------------------------------------------------------------------------

test('partner_bank._partner_bank_bridge: normalizeOptionalText returns undefined for undefined', () => {
  expect(normalizeOptionalText(undefined)).toBe(undefined);
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText returns null for null', () => {
  expect(normalizeOptionalText(null)).toBe(null);
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText returns null for empty', () => {
  expect(normalizeOptionalText('')).toBe(null);
  expect(normalizeOptionalText('   ')).toBe(null);
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText trims value', () => {
  expect(normalizeOptionalText('  abc  ')).toBe('abc');
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText preserves case by default', () => {
  expect(normalizeOptionalText('MyValue')).toBe('MyValue');
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText uppercases with upper option', () => {
  expect(normalizeOptionalText('  abc  ', { upper: true })).toBe('ABC');
});

test('partner_bank._partner_bank_bridge: normalizeOptionalText lowercases with lower option', () => {
  expect(normalizeOptionalText('  ABC  ', { lower: true })).toBe('abc');
});

// ---------------------------------------------------------------------------
// assertRequiredText
// ---------------------------------------------------------------------------

test('partner_bank._partner_bank_bridge: assertRequiredText trims value', () => {
  expect(assertRequiredText('  AccountName  ', 'AccountName')).toBe('AccountName');
});

test('partner_bank._partner_bank_bridge: assertRequiredText throws for empty', () => {
  expect(() => assertRequiredText('', 'AccountName')).toThrow();
  expect(() => assertRequiredText('   ', 'AccountName')).toThrow();
  expect(() => assertRequiredText(undefined, 'AccountName')).toThrow();
  expect(() => assertRequiredText(null, 'AccountName')).toThrow();
});

test('partner_bank._partner_bank_bridge: assertRequiredText error message includes fieldName', () => {
  let err: unknown;
  try {
    assertRequiredText('', 'AccountName');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('AccountName is required');
});

test('partner_bank._partner_bank_bridge: assertRequiredText uppercases with upper option', () => {
  expect(assertRequiredText('  abc  ', 'Field', { upper: true })).toBe('ABC');
});

test('partner_bank._partner_bank_bridge: assertRequiredText preserves case without upper option', () => {
  expect(assertRequiredText('  abc  ', 'Field')).toBe('abc');
});
