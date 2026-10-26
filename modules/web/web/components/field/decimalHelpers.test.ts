/*
 * SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import Decimal from '@/core/utils/decimal';
// Import pure formatters only — i18nStore/index pulls Pinia persist(localStorage).
import { formatFixedDecimalString } from '@/web/web/stores/i18nStore/languageFormat';
import { formatDecimalDisplayText, resolveDecimalEditScale, resolveDecimalCellScale } from './decimalHelpers';

test('formatDecimalDisplayText > pads when a fixed scale is declared', () => {
  expect(formatDecimalDisplayText(new Decimal('0.01'), 2, Decimal.ROUND_HALF_UP)).toBe('0.01');
  expect(formatDecimalDisplayText(new Decimal('0.01'), 4, Decimal.ROUND_HALF_UP)).toBe('0.0100');
});

test('formatDecimalDisplayText > keeps significant digits when scale is unset (no zero pad)', () => {
  expect(formatDecimalDisplayText(new Decimal('0.01'), undefined, Decimal.ROUND_HALF_UP)).toBe('0.01');
  expect(formatDecimalDisplayText(new Decimal('0.010000000000000000'), undefined, Decimal.ROUND_HALF_UP)).toBe('0.01');
  expect(formatDecimalDisplayText(new Decimal('1.234567890123456789'), undefined, Decimal.ROUND_HALF_UP)).toBe(
    '1.234567890123456789'
  );
});

test('formatDecimalDisplayText > applies locale separators without forcing extra fractional zeros', () => {
  const numberFormat = { thousandsSeparator: ',', decimalSeparator: '.', grouping: [3, 0] };
  expect(
    formatDecimalDisplayText(new Decimal('1234.5'), undefined, Decimal.ROUND_HALF_UP, {
      numberFormat,
      formatFixedDecimalString,
    })
  ).toBe('1,234.5');
  expect(
    formatDecimalDisplayText(new Decimal('1234.5'), 2, Decimal.ROUND_HALF_UP, {
      numberFormat,
      formatFixedDecimalString,
    })
  ).toBe('1,234.50');
});

test('formatDecimalDisplayText > falls back to plain text when formatFixedDecimalString throws', () => {
  expect(
    formatDecimalDisplayText(new Decimal('1.5'), 2, Decimal.ROUND_HALF_UP, {
      numberFormat: { thousandsSeparator: ',', decimalSeparator: '.' },
      formatFixedDecimalString: () => {
        throw new Error('fmt boom');
      },
    })
  ).toBe('1.50');
});

test('resolveDecimalEditScale > uses declared fixed scale, otherwise DB soft max 18', () => {
  expect(resolveDecimalEditScale(2)).toBe(2);
  expect(resolveDecimalEditScale(undefined)).toBe(18);
  expect(resolveDecimalEditScale(-1)).toBe(18);
});

test('resolveDecimalCellScale > prefers a valid getScale result and falls back to 18', () => {
  expect(resolveDecimalCellScale(() => 2)).toBe(2);
  expect(resolveDecimalCellScale(() => 0)).toBe(0);
  expect(resolveDecimalCellScale(() => 18)).toBe(18);
  expect(resolveDecimalCellScale(() => 19)).toBe(18);
  expect(resolveDecimalCellScale(() => 1.5)).toBe(18);
  expect(resolveDecimalCellScale(undefined)).toBe(18);
  expect(
    resolveDecimalCellScale(() => {
      throw new Error('scale boom');
    })
  ).toBe(18);
});

