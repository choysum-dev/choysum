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
});

test('patternToTranslatedTrigramPattern keeps wildcards and JSON-escapes text', () => {
  expect(patternToTranslatedTrigramPattern('%ab%')).toBe('%');
  expect(patternToTranslatedTrigramPattern('%abc%')).toBe('%abc%');
  expect(patternToTranslatedTrigramPattern('%a"b%')).toBe('%a\\"b%');
  expect(patternToTranslatedTrigramPattern('hello')).toBe('hello');
});

test('fieldHasTranslatedTrigramIndex and resolveTranslatedTrigramPrefilterPattern gates', () => {
  expect(fieldHasTranslatedTrigramIndex({ translate: true, column: { index: 'trigram' } })).toBe(true);
  expect(fieldHasTranslatedTrigramIndex({ translate: true, column: { index: true } })).toBe(false);
  expect(fieldHasTranslatedTrigramIndex({ translate: true })).toBe(false);

  expect(resolveTranslatedTrigramPrefilterPattern('ilike', '%ab%')).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('ilike', '%abc%')).toBe('%abc%');
  expect(resolveTranslatedTrigramPrefilterPattern('in', ['hello'])).toBe('%hello%');
  expect(resolveTranslatedTrigramPrefilterPattern('in', ['a', 'b'])).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('not ilike', '%abc%')).toBeNull();
});
