// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  patternToTranslatedTrigramPattern,
  valueToTranslatedTrigramPattern,
  fieldHasTranslatedTrigramIndex,
  resolveTranslatedTrigramPrefilterPattern,
} from '../translated_field_sql';

test('valueToTranslatedTrigramPattern JSON-escapes and wraps with wildcards', () => {
  expect(valueToTranslatedTrigramPattern('ab')).toBe('%');
  expect(valueToTranslatedTrigramPattern('abc')).toBe('%abc%');
  // JSON escape of quote, then PG wildcard escape doubles the backslash.
  expect(valueToTranslatedTrigramPattern('a"b')).toBe('%a\\\\"b%');
  expect(valueToTranslatedTrigramPattern('你好世界')).toBe('%你好世界%');
  expect(valueToTranslatedTrigramPattern('a_b%c')).toBe('%a\\_b\\%c%');
});

test('patternToTranslatedTrigramPattern keeps wildcards and JSON-escapes text', () => {
  expect(patternToTranslatedTrigramPattern('%ab%')).toBe('%');
  expect(patternToTranslatedTrigramPattern('%abc%')).toBe('%abc%');
  expect(patternToTranslatedTrigramPattern('%a"b%')).toBe('%a\\"b%');
  expect(patternToTranslatedTrigramPattern('hello')).toBe('hello');
  expect(patternToTranslatedTrigramPattern('%a\\b_c%')).not.toBe('%');
  expect(patternToTranslatedTrigramPattern('%a\\b%')).toBe('%');
});

test('fieldHasTranslatedTrigramIndex and resolveTranslatedTrigramPrefilterPattern gates', () => {
  expect(fieldHasTranslatedTrigramIndex({ translate: true, column: { index: 'trigram' } })).toBe(true);
  expect(fieldHasTranslatedTrigramIndex({ translate: true, column: { index: true } })).toBe(false);
  expect(fieldHasTranslatedTrigramIndex({ translate: true })).toBe(false);
  expect(fieldHasTranslatedTrigramIndex(undefined)).toBe(false);
  expect(fieldHasTranslatedTrigramIndex({ translate: false, column: { index: 'trigram' } })).toBe(false);
  expect(fieldHasTranslatedTrigramIndex({ column: { index: 'trigram' } })).toBe(false);

  expect(resolveTranslatedTrigramPrefilterPattern('ilike', '%ab%')).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('ilike', '%abc%')).toBe('%abc%');
  expect(resolveTranslatedTrigramPrefilterPattern('=like', '%ab%')).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('=ilike', '%abc%')).toBe('%abc%');
  expect(resolveTranslatedTrigramPrefilterPattern('in', ['hello'])).toBe('%hello%');
  expect(resolveTranslatedTrigramPrefilterPattern('in', ['ab'])).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('in', ['a', 'b'])).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('not ilike', '%abc%')).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('=', 123 as any)).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('==', 'hello')).toBe('%hello%');
});
