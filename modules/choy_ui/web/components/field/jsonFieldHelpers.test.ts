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

  test('normalizeChoyJsonIncoming accepts object and JSON string', () => {
    expect(normalizeChoyJsonIncoming({ x: 1 })).toEqual({ x: 1 });
    expect(normalizeChoyJsonIncoming('{"y":2}')).toEqual({ y: 2 });
    expect(normalizeChoyJsonIncoming('not-json')).toBeNull();
  });

  test('tryParseChoyJson validates object / array / null', () => {
    expect(tryParseChoyJson('{"a":1}')).toEqual({ ok: true, value: { a: 1 } });
    expect(tryParseChoyJson('[1]', { allowArray: false }).ok).toBe(false);
    expect(tryParseChoyJson('[1]', { allowArray: true })).toEqual({ ok: true, value: [1] });
    expect(tryParseChoyJson('null', { nullable: true })).toEqual({ ok: true, value: null });
    expect(tryParseChoyJson('null', { nullable: false }).ok).toBe(false);
    expect(tryParseChoyJson('42').ok).toBe(false);
    expect(tryParseChoyJson('{').ok).toBe(false);
  });
});
