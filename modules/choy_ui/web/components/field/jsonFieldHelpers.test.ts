// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeChoyJsonIncoming,
  stringifyChoyJson,
  tryParseChoyJson,
} from './jsonFieldHelpers';

describe('jsonFieldHelpers', () => {
  test('stringifyChoyJson sorts object keys', () => {
    expect(stringifyChoyJson({ b: 1, a: 2 })).toBe('{\n  "a": 2,\n  "b": 1\n}');
  });

  test('stringifyChoyJson preserves nested object keys', () => {
    expect(stringifyChoyJson({ b: 1, nested: { z: 9, a: 1 } })).toBe(
      '{\n  "b": 1,\n  "nested": {\n    "a": 1,\n    "z": 9\n  }\n}',
    );
  });

  test('stringifyChoyJson preserves own __proto__ keys', () => {
    const parsed = JSON.parse('{"b":1,"__proto__":{"x":1},"a":2}') as Record<string, unknown>;
    const text = stringifyChoyJson(parsed);
    expect(text).toContain('"__proto__"');
    expect(text).toContain('"a": 2');
    const roundTrip = JSON.parse(text) as Record<string, unknown>;
    // Object-literal `{ __proto__: … }` sets the prototype; assert via own keys instead.
    expect(Object.keys(roundTrip).sort()).toEqual(['__proto__', 'a', 'b']);
    expect(roundTrip.a).toBe(2);
    expect(roundTrip.b).toBe(1);
    expect(Object.prototype.hasOwnProperty.call(roundTrip, '__proto__')).toBe(true);
    expect(roundTrip['__proto__']).toEqual({ x: 1 });
  });

  test('stringifyChoyJson preserves Date via toJSON', () => {
    const createdAt = new Date('2020-01-01T00:00:00.000Z');
    const text = stringifyChoyJson({ z: 1, createdAt });
    expect(text).toContain('"createdAt": "2020-01-01T00:00:00.000Z"');
    expect(text).toContain('"z": 1');
  });

  test('stringifyChoyJson handles arrays, scalars, compact mode, and null', () => {
    expect(stringifyChoyJson(null)).toBe('');
    expect(stringifyChoyJson([1, { b: 1, a: 2 }], false)).toBe('[1,{"a":2,"b":1}]');
    expect(stringifyChoyJson(42)).toBe('42');
    expect(stringifyChoyJson({ a: 1 }, false)).toBe('{"a":1}');
  });

  test('stringifyChoyJson returns empty string when stringify throws', () => {
    const bad = {
      toJSON(): never {
        throw new Error('boom');
      },
    };
    expect(stringifyChoyJson(bad)).toBe('');
  });

  test('stringifyChoyJson coalesces undefined toJSON results to empty string', () => {
    const blank = {
      toJSON(): undefined {
        return undefined;
      },
    };
    expect(stringifyChoyJson(blank)).toBe('');
  });

  test('normalizeChoyJsonIncoming accepts object and JSON string', () => {
    expect(normalizeChoyJsonIncoming({ x: 1 })).toEqual({ x: 1 });
    expect(normalizeChoyJsonIncoming('{"y":2}')).toEqual({ y: 2 });
    expect(normalizeChoyJsonIncoming('not-json')).toBeNull();
    expect(normalizeChoyJsonIncoming(null)).toBeNull();
    expect(normalizeChoyJsonIncoming(3)).toBeNull();
  });

  test('tryParseChoyJson validates object / array / null', () => {
    expect(tryParseChoyJson('{"a":1}')).toEqual({ ok: true, value: { a: 1 } });
    expect(tryParseChoyJson('[1]', { allowArray: false }).ok).toBe(false);
    expect(tryParseChoyJson('[1]', { allowArray: true })).toEqual({ ok: true, value: [1] });
    expect(tryParseChoyJson('null', { nullable: true })).toEqual({ ok: true, value: null });
    expect(tryParseChoyJson('null', { nullable: false }).ok).toBe(false);
    expect(tryParseChoyJson('42').ok).toBe(false);
    expect(tryParseChoyJson('{').ok).toBe(false);
    expect(tryParseChoyJson('', { nullable: true })).toEqual({ ok: true, value: null });
    expect(tryParseChoyJson('', { nullable: false }).ok).toBe(false);
  });
});
