// @vitest-environment happy-dom
/*
 * SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it } from 'vitest';
import Decimal from '@/core/utils/decimal';
// Import pure formatters only — i18nStore/index pulls Pinia persist(localStorage).
import { formatFixedDecimalString } from '@/web/web/stores/i18nStore/language_format';
import { formatODecimalDisplayText, resolveODecimalEditScale, resolveODecimalCellScale } from './odecimal_helpers';

describe('formatODecimalDisplayText', () => {
  it('pads when a fixed scale is declared', () => {
    expect(formatODecimalDisplayText(new Decimal('0.01'), 2, Decimal.ROUND_HALF_UP)).toBe('0.01');
    expect(formatODecimalDisplayText(new Decimal('0.01'), 4, Decimal.ROUND_HALF_UP)).toBe('0.0100');
  });

  it('keeps significant digits when scale is unset (no zero pad)', () => {
    expect(formatODecimalDisplayText(new Decimal('0.01'), undefined, Decimal.ROUND_HALF_UP)).toBe('0.01');
    expect(formatODecimalDisplayText(new Decimal('0.010000000000000000'), undefined, Decimal.ROUND_HALF_UP)).toBe('0.01');
    expect(formatODecimalDisplayText(new Decimal('1.234567890123456789'), undefined, Decimal.ROUND_HALF_UP)).toBe(
      '1.234567890123456789'
    );
  });

  it('applies locale separators without forcing extra fractional zeros', () => {
    const numberFormat = { thousandsSeparator: ',', decimalSeparator: '.', grouping: [3, 0] };
    expect(
      formatODecimalDisplayText(new Decimal('1234.5'), undefined, Decimal.ROUND_HALF_UP, {
        numberFormat,
        formatFixedDecimalString,
      })
    ).toBe('1,234.5');
    expect(
      formatODecimalDisplayText(new Decimal('1234.5'), 2, Decimal.ROUND_HALF_UP, {
        numberFormat,
        formatFixedDecimalString,
      })
    ).toBe('1,234.50');
  });

  it('falls back to plain text when formatFixedDecimalString throws', () => {
    expect(
      formatODecimalDisplayText(new Decimal('1.5'), 2, Decimal.ROUND_HALF_UP, {
        numberFormat: { thousandsSeparator: ',', decimalSeparator: '.' },
        formatFixedDecimalString: () => {
          throw new Error('fmt boom');
        },
      })
    ).toBe('1.50');
  });
});

describe('resolveODecimalEditScale', () => {
  it('uses declared fixed scale, otherwise DB soft max 18', () => {
    expect(resolveODecimalEditScale(2)).toBe(2);
    expect(resolveODecimalEditScale(undefined)).toBe(18);
    expect(resolveODecimalEditScale(-1)).toBe(18);
  });
});

describe('resolveODecimalCellScale', () => {
  it('prefers a valid getScale result and falls back to 18', () => {
    expect(resolveODecimalCellScale(() => 2)).toBe(2);
    expect(resolveODecimalCellScale(() => 0)).toBe(0);
    expect(resolveODecimalCellScale(() => 18)).toBe(18);
    expect(resolveODecimalCellScale(() => 19)).toBe(18);
    expect(resolveODecimalCellScale(() => 1.5)).toBe(18);
    expect(resolveODecimalCellScale(undefined)).toBe(18);
    expect(
      resolveODecimalCellScale(() => {
        throw new Error('scale boom');
      })
    ).toBe(18);
  });
});
