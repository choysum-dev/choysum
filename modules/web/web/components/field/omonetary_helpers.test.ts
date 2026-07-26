// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import Decimal from '@/core/utils/decimal';
import {
  asMonetaryDecimal,
  clampMonetaryValue,
  currencyFieldPaths,
  formatMonetaryDisplayText,
  getByPath,
  isIntermediateMonetaryInput,
  leafOfPath,
  parseStrictMonetary,
  quantizeMonetaryForCompare,
  readCurrencyCode,
  readCurrencyDigits,
  readCurrencySymbol,
  resolveAggregateDisplayValue,
  resolveCurrencyValue,
  resolveMonetaryScaleFromRecord,
  validateMonetaryValue,
} from './omonetary_helpers';

describe('omonetary_helpers currency resolution', () => {
  it('reads path / sibling currency and digits/code/symbol', () => {
    expect(getByPath({ a: { b: 1 } }, 'a.b')).toBe(1);
    expect(getByPath(null, 'a')).toBeNull();
    expect(resolveCurrencyValue(null, 'CurrencyId', 'Amount')).toBeUndefined();
    expect(resolveCurrencyValue({ CurrencyId: 'x' }, undefined, 'Amount')).toBeUndefined();
    expect(resolveCurrencyValue({ CurrencyId: { Code: 'USD' } }, 'CurrencyId', 'Amount')).toEqual({ Code: 'USD' });
    expect(resolveCurrencyValue({ nested: { CurrencyId: { Code: 'JPY' } } }, 'CurrencyId', 'nested.Amount')).toEqual({
      Code: 'JPY',
    });
    expect(
      resolveCurrencyValue({ CurrencyId: { Code: 'TOP' }, nested: {} }, 'CurrencyId', 'nested.Amount')
    ).toEqual({ Code: 'TOP' });
    expect(resolveCurrencyValue({}, 'CurrencyId', '')).toBeUndefined();

    expect(readCurrencyDigits(null)).toBeUndefined();
    expect(readCurrencyDigits('x')).toBeUndefined();
    expect(readCurrencyDigits({ DecimalDigits: 2 })).toBe(2);
    expect(readCurrencyDigits({ decimalDigits: 19 })).toBeUndefined();
    expect(readCurrencyCode(null)).toBeUndefined();
    expect(readCurrencyCode({ Code: ' usd ' })).toBe('usd');
    expect(readCurrencyCode({ code: '  ' })).toBeUndefined();
    expect(readCurrencySymbol({ Symbol: '¥' })).toBe('¥');
    expect(readCurrencySymbol({ symbol: '  ' })).toBeUndefined();
    expect(resolveMonetaryScaleFromRecord({ CurrencyId: { DecimalDigits: 0 } }, 'CurrencyId', 'Amount')).toBe(0);
    expect(resolveMonetaryScaleFromRecord({}, 'CurrencyId', 'Amount', 5)).toBe(5);
  });
});

describe('omonetary_helpers display and parse', () => {
  it('formats with code, symbol, and fallbacks', () => {
    expect(formatMonetaryDisplayText(null, { scale: 2, roundingMode: Decimal.ROUND_HALF_UP, currency: null })).toBe('');
    expect(formatMonetaryDisplayText('bad', { scale: 2, roundingMode: Decimal.ROUND_HALF_UP, currency: null })).toBe('');

    const withCode = formatMonetaryDisplayText('12.345', {
      scale: 2,
      roundingMode: Decimal.ROUND_HALF_UP,
      currency: { Code: 'USD' },
      formatters: {
        formatCurrencyFromConfig: (_v, _c, code) => `FMT:${code}`,
        formatFixedDecimalString: fixed => `FIX:${fixed}`,
        numberFormat: { decimalDigits: 2 },
      },
    });
    expect(withCode).toBe('FMT:USD');

    let currencyFmtArg: number | string | undefined;
    const large = formatMonetaryDisplayText('9007199254740993.12', {
      scale: 2,
      roundingMode: Decimal.ROUND_HALF_UP,
      currency: { Code: 'USD' },
      formatters: {
        formatCurrencyFromConfig: (v, _c, code) => {
          currencyFmtArg = v;
          return `FMT:${code}:${v}`;
        },
        formatFixedDecimalString: fixed => fixed,
        numberFormat: { decimalDigits: 2 },
      },
    });
    expect(typeof currencyFmtArg).toBe('string');
    expect(currencyFmtArg).toBe('9007199254740993.12');
    expect(large).toBe('FMT:USD:9007199254740993.12');

    const withSymbol = formatMonetaryDisplayText('12.345', {
      scale: 2,
      roundingMode: Decimal.ROUND_HALF_UP,
      currency: { Symbol: '$' },
      formatters: {
        formatCurrencyFromConfig: () => 'unused',
        formatFixedDecimalString: fixed => fixed,
      },
    });
    expect(withSymbol).toBe('$ 12.35');

    const plain = formatMonetaryDisplayText(new Decimal('1.2'), {
      scale: 2,
      roundingMode: Decimal.ROUND_HALF_UP,
      currency: null,
    });
    expect(plain).toBe('1.20');

    expect(
      formatMonetaryDisplayText('12.3', {
        scale: 1,
        roundingMode: Decimal.ROUND_HALF_UP,
        currency: { Code: 'USD', Symbol: '$' },
        formatters: {
          formatCurrencyFromConfig: undefined as any,
          formatFixedDecimalString: fixed => `FIX:${fixed}`,
        },
      })
    ).toBe('$ FIX:12.3');
  });

  it('parses, clamps, validates, and compares', () => {
    expect(asMonetaryDecimal(null)).toBeNull();
    expect(asMonetaryDecimal('x')).toBeNull();
    expect(asMonetaryDecimal(new Decimal(1))!.eq(1)).toBe(true);
    expect(isIntermediateMonetaryInput(null)).toBe(false);
    expect(isIntermediateMonetaryInput('-')).toBe(true);
    expect(isIntermediateMonetaryInput('12.')).toBe(true);
    expect(isIntermediateMonetaryInput('12.3')).toBe(false);

    const bounds = { precision: 10, min: new Decimal(0), max: new Decimal(100) };
    expect(parseStrictMonetary('1.234', 2, bounds)).toBeNull();
    expect(parseStrictMonetary('1.23', 2, bounds)?.toString()).toBe('1.23');
    expect(parseStrictMonetary('abc', 2, bounds)).toBeNull();
    expect(parseStrictMonetary('-1', 2, bounds)).toBeNull();
    expect(parseStrictMonetary('', 2, bounds)).toBeNull();
    expect(parseStrictMonetary('101', 2, bounds)).toBeNull();
    expect(clampMonetaryValue(new Decimal('1.239'), 2, Decimal.ROUND_HALF_UP, bounds).toString()).toBe('1.24');
    expect(
      clampMonetaryValue(new Decimal('12345678901'), 0, Decimal.ROUND_HALF_UP, { precision: 5 }).toString()
    ).toBe('12346');
    expect(
      clampMonetaryValue(new Decimal('50'), 0, Decimal.ROUND_HALF_UP, {
        precision: 10,
        min: new Decimal(60),
        max: new Decimal(100),
      }).toString()
    ).toBe('60');
    expect(quantizeMonetaryForCompare(null, 2, Decimal.ROUND_HALF_UP)).toBeNull();
    expect(quantizeMonetaryForCompare('1.239', 2, Decimal.ROUND_HALF_UP)?.toString()).toBe('1.24');

    // formatters throw → fall back to Decimal#toString
    expect(
      formatMonetaryDisplayText('1.25', {
        scale: 2,
        roundingMode: Decimal.ROUND_HALF_UP,
        currency: { Code: 'USD' },
        formatters: {
          formatCurrencyFromConfig: () => {
            throw new Error('fmt boom');
          },
          formatFixedDecimalString: () => {
            throw new Error('fix boom');
          },
        },
      })
    ).toBe('1.25');

    const t = (msg: string, ...args: unknown[]) => `${msg}:${args.join(',')}`;
    expect(validateMonetaryValue(null, 2, bounds, t)).toBeNull();
    expect(validateMonetaryValue('x', 2, bounds, t)).toContain('valid number');
    expect(validateMonetaryValue('1.234', 2, bounds, t)).toContain('Decimal places');
    expect(validateMonetaryValue('1.23', 2, { precision: 2 }, t)).toContain('Total digits');
    expect(validateMonetaryValue('-1', 2, bounds, t)).toContain('less than');
    expect(validateMonetaryValue('101', 2, bounds, t)).toContain('greater than');
    expect(validateMonetaryValue('1.23', 2, bounds, t)).toBeNull();
  });
});

describe('omonetary_helpers aggregate display', () => {
  it('resolves metrics aliases and unique top-level agg keys', () => {
    expect(leafOfPath('a.b.Amount')).toBe('Amount');
    expect(leafOfPath('')).toBe('');
    expect(resolveAggregateDisplayValue('raw', null, { bindingProp: 'Amount' })).toBe('raw');
    expect(resolveAggregateDisplayValue(null, null, { bindingProp: 'Amount' })).toBeNull();

    expect(
      resolveAggregateDisplayValue(null, { metrics: { 'Amount__sum': 9 } }, { bindingProp: 'Amount', agg: 'sum' })
    ).toBe(9);
    expect(
      resolveAggregateDisplayValue(null, { Amount__avg: 3 }, { bindingProp: 'Amount', agg: { agg: 'avg', alias: 'A' } })
    ).toBe(3);
    expect(resolveAggregateDisplayValue(null, { metrics: { __count: 2 } }, { bindingProp: 'Amount', agg: 'count' })).toBe(2);
    expect(
      resolveAggregateDisplayValue(null, { metrics: { custom: 7 } }, { bindingProp: 'Amount', agg: { agg: 'sum', alias: 'custom' } })
    ).toBe(7);

    expect(
      resolveAggregateDisplayValue(null, { metrics: { Amount__max: 5 } }, { bindingProp: 'Amount' })
    ).toBe(5);
    expect(resolveAggregateDisplayValue(null, { Amount__min: 1 }, { bindingProp: 'Amount' })).toBe(1);
    expect(
      resolveAggregateDisplayValue(null, { metrics: { Amount__count_distinct: 4 } }, { bindingProp: 'line.Amount' })
    ).toBe(4);
    expect(
      resolveAggregateDisplayValue(null, { metrics: { Amount__sum: 1, Amount__avg: 2 } }, { bindingProp: 'Amount' })
    ).toBeNull();
    expect(resolveAggregateDisplayValue(null, { Amount__sum: 1, Amount__avg: 2 }, { bindingProp: 'Amount' })).toBeNull();
    expect(resolveAggregateDisplayValue(null, { Other: 1 }, { bindingProp: 'Amount' })).toBeNull();
  });

  it('builds currency sibling registration paths', () => {
    expect(currencyFieldPaths('Amount', undefined)).toEqual([]);
    expect(currencyFieldPaths('', 'CurrencyId')).toEqual([]);
    expect(currencyFieldPaths('Amount', 'CurrencyId')).toEqual([
      'CurrencyId',
      'CurrencyId.DecimalDigits',
      'CurrencyId.Symbol',
      'CurrencyId.Code',
    ]);
    expect(currencyFieldPaths('line.Amount', ' CurrencyId ')).toEqual([
      'line.CurrencyId',
      'line.CurrencyId.DecimalDigits',
      'line.CurrencyId.Symbol',
      'line.CurrencyId.Code',
    ]);
  });
});
