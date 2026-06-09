// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from 'decimal.js';
import {
  toDecimalRounding,
  isDecimal,
  isDecimalLike,
  isDecimalLeak,
  serialize,
  deserialize,
  decimalEqual,
  isBigdecimalEnvelope,
  asBigdecimal,
  toDecimalString,
  getScaleAndRound,
  normalizeDecimalByMeta,
} from '@/core/utils/decimal';

test('decimal utils: rounding mapping and envelope guards', () => {
  expect(toDecimalRounding('ROUND_HALF_DOWN')).toBe((Decimal as any).ROUND_HALF_DOWN);
  expect(toDecimalRounding(7 as any)).toBe(7 as any);
  expect(toDecimalRounding('UNKNOWN' as any)).toBe((Decimal as any).ROUND_HALF_UP);

  expect(isBigdecimalEnvelope({ $bigdecimal: '1.23' })).toBe(true);
  expect(isBigdecimalEnvelope({ $bigdecimal: 1.23 })).toBe(true);
  expect(isBigdecimalEnvelope({})).toBe(false);
});

test('decimal utils: toDecimalRounding maps all named constants and defaults safely', () => {
  const entries: Array<[any, any]> = [
    ['ROUND_UP', (Decimal as any).ROUND_UP],
    ['ROUND_DOWN', (Decimal as any).ROUND_DOWN],
    ['ROUND_CEIL', (Decimal as any).ROUND_CEIL],
    ['ROUND_FLOOR', (Decimal as any).ROUND_FLOOR],
    ['ROUND_HALF_UP', (Decimal as any).ROUND_HALF_UP],
    ['ROUND_HALF_DOWN', (Decimal as any).ROUND_HALF_DOWN],
    ['ROUND_HALF_EVEN', (Decimal as any).ROUND_HALF_EVEN],
    ['ROUND_HALF_CEIL', (Decimal as any).ROUND_HALF_CEIL],
    ['ROUND_HALF_FLOOR', (Decimal as any).ROUND_HALF_FLOOR],
  ];

  for (const [name, expected] of entries) {
    expect(toDecimalRounding(name as any)).toBe(expected);
  }
  expect(toDecimalRounding(undefined as any)).toBe((Decimal as any).ROUND_HALF_UP);
});

test('decimal utils: decimal identity and leak detection', () => {
  const d = new Decimal('12.30');
  expect(isDecimal(d)).toBe(true);
  expect(isDecimal({ toString: () => '12.3' })).toBe(false);

  expect(isDecimalLike(d)).toBe(true);
  expect(isDecimalLike({ toString: () => 'x', decimalPlaces: () => 2 })).toBe(true);
  expect(isDecimalLike({})).toBe(false);

  expect(isDecimalLeak({ s: 1, e: 1, d: [123] })).toBe(true);
  expect(isDecimalLeak({ s: 1, e: 1, d: ['x'] })).toBe(false);
});

test('decimal utils: isDecimal and isDecimalLike guard non-object and incomplete duck types', () => {
  expect(isDecimal(null)).toBe(false);
  expect(isDecimal(1)).toBe(false);
  expect(
    isDecimal({
      constructor: { name: 'DecimalLike' },
      toString: () => '1',
      toNumber: () => 1,
      plus: () => 1,
    })
  ).toBe(false);

  expect(isDecimalLike(null)).toBe(false);
  expect(isDecimalLike(1)).toBe(false);
});

test('decimal utils: serialize and deserialize handle bigint/decimal/leak/object recursion', () => {
  const leak = Object.create({ marker: true });
  leak.s = 1;
  leak.e = 1;
  leak.d = [123];
  const cycle: any = { Name: 'n1' };
  cycle.self = cycle;

  const serialized = serialize({
    Big: 9n,
    Dec: new Decimal('10.50'),
    Leak: leak,
    Nested: { Arr: [new Decimal('1.20')] },
    Cycled: cycle,
  });

  expect(serialized.Big).toEqual({ $bigint: '9' });
  expect(serialized.Dec).toEqual({ $bigdecimal: '10.5' });
  expect(serialized.Leak).toEqual({ $bigdecimal: '12.3' });
  expect(serialized.Nested.Arr[0]).toEqual({ $bigdecimal: '1.2' });
  expect(serialized.Cycled.self).toBe(serialized.Cycled);

  const deserialized = deserialize({
    A: { $bigdecimal: '8.90' },
    B: { $bigint: '42' },
    C: [{ $bigdecimal: '1.00' }],
  });

  expect((deserialized.A as Decimal).toString()).toBe('8.9');
  expect(typeof deserialized.B).toBe('bigint');
  expect(String(deserialized.B)).toBe('42');
  expect((deserialized.C[0] as Decimal).toString()).toBe('1');
});

test('decimal utils: deserialize falls back gracefully on invalid numeric payloads', () => {
  const out = deserialize({
    BadDec: { $bigdecimal: 'not-a-number' },
    BadBigInt: { $bigint: 'bad' },
  });

  expect(out.BadDec).toBe('not-a-number');
  expect(out.BadBigInt).toBe('bad');
});

test('decimal utils: serialize and deserialize keep nullish and malformed envelope payloads stable', () => {
  expect(serialize(null)).toBe(null);
  expect(deserialize(null)).toBe(null);

  const serializedEnvelope = serialize({ $bigdecimal: 12.3 });
  expect(serializedEnvelope).toEqual({ $bigdecimal: '12.3' });

  const malformedDec = deserialize({ $bigdecimal: { bad: true } });
  expect(malformedDec).toEqual({ $bigdecimal: { bad: true } });

  const malformedBigInt = deserialize({ $bigint: { bad: true } });
  expect(malformedBigInt).toEqual({ $bigint: { bad: true } });
});

test('decimal utils: comparison and conversion helpers', () => {
  expect(decimalEqual('1.0', new Decimal('1'))).toBe(true);
  expect(decimalEqual('1.01', new Decimal('1'))).toBe(false);

  expect(asBigdecimal(new Decimal('3.40'))).toEqual({ $bigdecimal: '3.4' });
  expect(asBigdecimal(5)).toEqual({ $bigdecimal: '5' });
  expect(asBigdecimal({ $bigdecimal: 9 })).toEqual({ $bigdecimal: '9' });

  expect(toDecimalString('2.500')).toBe('2.5');
  expect(toDecimalString({ $bigdecimal: '7.00' })).toBe('7');
  expect(toDecimalString('bad-number')).toBeUndefined();
});

test('decimal utils: comparison helper returns false on non-decimal operands', () => {
  expect(decimalEqual({ foo: 1 }, { bar: 2 })).toBe(false);
  expect(decimalEqual(undefined, '1')).toBe(false);
});

test('decimal utils: getScaleAndRound and normalizeDecimalByMeta enforce precision rules', () => {
  const meta = { column: { precision: 6, scale: 2, round: 'ROUND_HALF_UP' } } as any;
  const sr = getScaleAndRound(meta);
  expect(sr.scale).toBe(2);
  expect(sr.round).toBe((Decimal as any).ROUND_HALF_UP);

  const ok = normalizeDecimalByMeta(meta, '123.456');
  expect((ok as Decimal).toString()).toBe('123.46');

  const overflowScale = normalizeDecimalByMeta({ column: { precision: 30, scale: 19 } } as any, '1.1234567890123456789');
  expect(overflowScale).toBeUndefined();

  const overflowInt = normalizeDecimalByMeta({ column: { precision: 38, scale: 0 } } as any, '123456789012345678901');
  expect(overflowInt).toBeUndefined();

  const invalid = normalizeDecimalByMeta(meta, 'not-a-number');
  expect(invalid).toBeUndefined();
});

test('decimal utils: normalizeDecimalByMeta handles envelope, infinity and null input branches', () => {
  const meta = { column: { precision: 8, scale: 2, round: 'ROUND_HALF_UP' } } as any;

  const fromEnvelope = normalizeDecimalByMeta(meta, { $bigdecimal: '3.456' });
  expect((fromEnvelope as Decimal).toString()).toBe('3.46');

  const fromInfinity = normalizeDecimalByMeta(meta, Infinity as any);
  expect(fromInfinity).toBeUndefined();

  const fromNull = normalizeDecimalByMeta(meta, null);
  expect(fromNull).toBeUndefined();
});
