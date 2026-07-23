// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import {
  normalizeOptionalString,
  normalizeStringArray,
  readRefId,
  normalizeRefId,
  normalizeRefIdList,
  normalizeOffset,
  normalizeLimit,
  normalizeFields,
  normalizeRpcRequireKey,
  rpcServiceWildcard,
  sortStrings,
  maybeRefId,
  normalizeScopeRefId,
  normalizeUiResourceId,
  parseJsonStringArray,
  normalizeScopeId,
  uniqScopeIds,
  normalizePreferences,
  buildScopePreferences,
  asBigInt,
  toDateOnlyString,
  NormalizationError,
  parseDecimalInput,
  toPositiveDecimal,
  normalizePositiveDecimalString,
  normalizeCodeRequired,
  normalizeCodeOptional,
  normalizeName,
  requireRefId,
  normalizeNullableString,
  normalizeOptionalNonEmptyString,
  isExpiredAt,
  roundToCurrencyAmount,
  normalizeRequiredText,
  parsePositiveInt,
  parseBigInt,
  normalizeDecimalDigits,
  normalizeDateString,
  normalizeEnumValue,
  resolveModelRefId,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeOptionalText,
  normalizeOptionalRefId,
  normalizeOptionalTranslatedText,
  normalizeRequiredTranslatedText,
  translatedTextHasValue,
  normalizeNonNegativeInt,
  normalizeSequenceInt,
} from '@/core/service/utils/normalization';

test('normalizeOptionalString returns trimmed string or undefined', () => {
  expect(normalizeOptionalString('  hello  ')).toBe('hello');
  expect(normalizeOptionalString(null)).toBe(undefined);
  expect(normalizeOptionalString(undefined)).toBe(undefined);
  expect(normalizeOptionalString('')).toBe(undefined);
  expect(normalizeOptionalString('   ')).toBe(undefined);
  expect(normalizeOptionalString(123)).toBe('123');
  expect(normalizeOptionalString(0)).toBe('0');
  expect(normalizeOptionalString(false)).toBe('false');
});

test('normalizeOptionalString uppercases with upper option', () => {
  expect(normalizeOptionalString('  abc  ', { upper: true })).toBe('ABC');
  expect(normalizeOptionalString('Mixed', { upper: true })).toBe('MIXED');
});

test('normalizeOptionalString lowercases with lower option', () => {
  expect(normalizeOptionalString('  ABC  ', { lower: true })).toBe('abc');
  expect(normalizeOptionalString('Mixed', { lower: true })).toBe('mixed');
});

test('normalizeOptionalString upper takes precedence over lower', () => {
  expect(normalizeOptionalString('  abc  ', { upper: true, lower: true })).toBe('ABC');
});

test('normalizeOptionalString with options returns undefined for empty', () => {
  expect(normalizeOptionalString('', { upper: true })).toBe(undefined);
  expect(normalizeOptionalString(null, { upper: true })).toBe(undefined);
});

test('normalizeStringArray deduplicates and filters empty strings', () => {
  expect(normalizeStringArray(null)).toEqual([]);
  expect(normalizeStringArray(undefined)).toEqual([]);
  expect(normalizeStringArray('not-an-array')).toEqual([]);
  expect(normalizeStringArray(123)).toEqual([]);
  expect(normalizeStringArray([])).toEqual([]);
  expect(normalizeStringArray(['a', 'b', 'a'])).toEqual(['a', 'b']);
  expect(normalizeStringArray(['  hello ', '', ' world ', '  hello  '])).toEqual(['hello', 'world']);
  expect(normalizeStringArray([null, undefined, '  valid  '])).toEqual(['valid']);
});

test('readRefId extracts id from string or object', () => {
  expect(readRefId(null)).toBe(undefined);
  expect(readRefId(undefined)).toBe(undefined);
  expect(readRefId('')).toBe(undefined);
  expect(readRefId(0)).toBe(undefined);
  expect(readRefId('  id123  ')).toBe('id123');
  expect(readRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(readRefId({ Id: '' })).toBe(undefined);
  expect(readRefId({ Id: null })).toBe(undefined);
  expect(readRefId({ id: 'lowercase' })).toBe(undefined);
  expect(readRefId(true)).toBe(undefined);
});

test('normalizeRefId returns trimmed string or null', () => {
  expect(normalizeRefId(null)).toBe(null);
  expect(normalizeRefId(undefined)).toBe(null);
  expect(normalizeRefId('')).toBe(null);
  expect(normalizeRefId(0)).toBe('0');
  expect(normalizeRefId('  id123  ')).toBe('id123');
  expect(normalizeRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(normalizeRefId({ id: 'lowercase' })).toBe('lowercase');
  expect(normalizeRefId({ Id: '', id: 'fallback' })).toBe(null);
  expect(normalizeRefId({ Id: '' })).toBe(null);
  expect(normalizeRefId(true)).toBe('true');
});

test('normalizeRefIdList wraps singleton, extracts Ids, filters null, and deduplicates', () => {
  expect(normalizeRefIdList(null)).toEqual([]);
  expect(normalizeRefIdList(undefined)).toEqual([]);
  expect(normalizeRefIdList([])).toEqual([]);
  expect(normalizeRefIdList('  id1  ')).toEqual(['id1']);
  expect(normalizeRefIdList({ Id: 'obj1' })).toEqual(['obj1']);
  expect(normalizeRefIdList(['a', 'b', 'a'])).toEqual(['a', 'b']);
  expect(normalizeRefIdList(['', '  ', 'valid'])).toEqual(['valid']);
  expect(normalizeRefIdList([{ Id: 'x' }, { Id: 'y' }, { Id: 'x' }])).toEqual(['x', 'y']);
  expect(normalizeRefIdList([null, undefined, { Id: 'z' }])).toEqual(['z']);
});

test('normalizeOffset returns non-negative floored integer', () => {
  expect(normalizeOffset(null)).toBe(0);
  expect(normalizeOffset(undefined)).toBe(0);
  expect(normalizeOffset(NaN)).toBe(0);
  expect(normalizeOffset(-1)).toBe(0);
  expect(normalizeOffset(-100)).toBe(0);
  expect(normalizeOffset(0)).toBe(0);
  expect(normalizeOffset(5)).toBe(5);
  expect(normalizeOffset(5.7)).toBe(5);
  expect(normalizeOffset('10')).toBe(10);
});

test('normalizeLimit returns positive floored integer or null', () => {
  expect(normalizeLimit(null)).toBe(null);
  expect(normalizeLimit(undefined)).toBe(null);
  expect(normalizeLimit(NaN)).toBe(null);
  expect(normalizeLimit(0)).toBe(null);
  expect(normalizeLimit(-1)).toBe(null);
  expect(normalizeLimit(-100)).toBe(null);
  expect(normalizeLimit(5)).toBe(5);
  expect(normalizeLimit(5.7)).toBe(5);
  expect(normalizeLimit('10')).toBe(10);
});

test('normalizeFields trims deduplicates and filters empty strings', () => {
  expect(normalizeFields(null)).toEqual([]);
  expect(normalizeFields(undefined)).toEqual([]);
  expect(normalizeFields('not-array')).toEqual([]);
  expect(normalizeFields([' Name ', '', ' Code ', ' name ', ''])).toEqual(['Name', 'Code', 'name']);
  expect(normalizeFields([])).toEqual([]);
});

test('normalizeRpcRequireKey keeps rpc and converts service prefix', () => {
  expect(normalizeRpcRequireKey('')).toBe('');
  expect(normalizeRpcRequireKey('   ')).toBe('');
  expect(normalizeRpcRequireKey('rpc:/auth.User/Browse')).toBe('rpc:/auth.User/Browse');
  expect(normalizeRpcRequireKey('service:/auth.User/Browse')).toBe('rpc:/auth.User/Browse');
  expect(normalizeRpcRequireKey('http:/auth.User/Browse')).toBe('');
});

test('rpcServiceWildcard returns service wildcard for rpc keys', () => {
  expect(rpcServiceWildcard('')).toBe('');
  expect(rpcServiceWildcard('service:/auth.User/Browse')).toBe('');
  expect(rpcServiceWildcard('rpc:/auth.User/Browse')).toBe('rpc:/auth.User/*');
  expect(rpcServiceWildcard('rpc:/auth.User/*')).toBe('rpc:/auth.User/*');
  expect(rpcServiceWildcard('rpc:/auth.User')).toBe('');
});

test('sortStrings returns stable sorted copy', () => {
  expect(sortStrings([])).toEqual([]);
  expect(sortStrings(['c', 'a', 'b'])).toEqual(['a', 'b', 'c']);
  expect(sortStrings(['a', 'a', 'b'])).toEqual(['a', 'a', 'b']);
});

test('maybeRefId extracts id from string or object with Id/id', () => {
  expect(maybeRefId(null)).toBe(undefined);
  expect(maybeRefId(undefined)).toBe(undefined);
  expect(maybeRefId('')).toBe(undefined);
  expect(maybeRefId(0)).toBe(undefined);
  expect(maybeRefId('  id123  ')).toBe('id123');
  expect(maybeRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(maybeRefId({ id: 'lowercase' })).toBe('lowercase');
  expect(maybeRefId({ Id: '', id: 'fallback' })).toBe('fallback');
  expect(maybeRefId(true)).toBe(undefined);
});

test('normalizeScopeRefId returns trimmed string or empty', () => {
  expect(normalizeScopeRefId(null)).toBe('');
  expect(normalizeScopeRefId(undefined)).toBe('');
  expect(normalizeScopeRefId('')).toBe('');
  expect(normalizeScopeRefId('  app123  ')).toBe('app123');
  expect(normalizeScopeRefId({ Id: '  obj456  ' })).toBe('obj456');
  expect(normalizeScopeRefId({ id: 'lowercase' })).toBe('lowercase');
  expect(normalizeScopeRefId(0)).toBe('0');
});

test('normalizeUiResourceId returns trimmed string or empty', () => {
  expect(normalizeUiResourceId(null)).toBe('');
  expect(normalizeUiResourceId(undefined)).toBe('');
  expect(normalizeUiResourceId('')).toBe('');
  expect(normalizeUiResourceId('  res123  ')).toBe('res123');
  expect(normalizeUiResourceId({ Id: '  res456  ' })).toBe('res456');
  expect(normalizeUiResourceId({ id: 'res789' })).toBe('res789');
});

test('parseJsonStringArray handles various input shapes', () => {
  expect(parseJsonStringArray(null)).toEqual([]);
  expect(parseJsonStringArray(undefined)).toEqual([]);
  expect(parseJsonStringArray([])).toEqual([]);
  expect(parseJsonStringArray(['a', 'b', 'a'])).toEqual(['a', 'b']);
  expect(parseJsonStringArray('["x","y"]')).toEqual(['x', 'y']);
  expect(parseJsonStringArray('single')).toEqual(['single']);
  expect(parseJsonStringArray({ value: ['v1', 'v2'] })).toEqual(['v1', 'v2']);
  expect(parseJsonStringArray({ values: ['w1'] })).toEqual(['w1']);
  expect(parseJsonStringArray({ items: ['i1', 'i2'] })).toEqual(['i1', 'i2']);
  expect(parseJsonStringArray({ '0': 'a', '1': 'b' })).toEqual(['a', 'b']);
  expect(parseJsonStringArray('')).toEqual([]);
  expect(parseJsonStringArray('   ')).toEqual([]);
});

test('normalizeScopeId normalizes company id-like values', () => {
  expect(normalizeScopeId(null)).toBe('');
  expect(normalizeScopeId(undefined)).toBe('');
  expect(normalizeScopeId('  C1  ')).toBe('C1');
  expect(normalizeScopeId({ Id: 'C2' })).toBe('C2');
  expect(normalizeScopeId({ id: 'C3' })).toBe('C3');
  expect(normalizeScopeId(123)).toBe('123');
});

test('uniqScopeIds preserves first-seen order', () => {
  expect(uniqScopeIds([])).toEqual([]);
  expect(uniqScopeIds(['C1', 'C2', 'C1', '', '  C3  '])).toEqual(['C1', 'C2', 'C3']);
  expect(uniqScopeIds(['  ', null as any, undefined as any, 'valid'])).toEqual(['valid']);
});

test('normalizePreferences normalizes various shapes to plain object', () => {
  expect(normalizePreferences(null)).toEqual({});
  expect(normalizePreferences(undefined)).toEqual({});
  expect(normalizePreferences('')).toEqual({});
  expect(normalizePreferences('{"key":"val"}')).toEqual({ key: 'val' });
  expect(normalizePreferences({ a: 1 })).toEqual({ a: 1 });
  expect(normalizePreferences([1, 2])).toEqual([1, 2] as any); // arrays pass through
  expect(normalizePreferences('   ')).toEqual({});
  // non-JSON string returns empty
  expect(normalizePreferences('not-json')).toEqual({});
});

test('buildScopePreferences merges active/enabled into base', () => {
  expect(buildScopePreferences({}, 'C1', ['C1', 'C2'])).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C1', 'C2'],
  });
  expect(buildScopePreferences({ theme: 'dark' }, 'C1', ['C1'])).toEqual({
    theme: 'dark',
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C1'],
  });
  expect(buildScopePreferences(null as any, 'X', [])).toEqual({
    activeCompanyId: 'X',
    enabledCompanyIds: [],
  });
});

// ---------------------------------------------------------------------------
// asBigInt
// ---------------------------------------------------------------------------

test('asBigInt returns bigint for bigint input', () => {
  expect(asBigInt(42n)).toBe(42n);
});

test('asBigInt converts number to bigint', () => {
  expect(asBigInt(7)).toBe(7n);
  expect(asBigInt(3.9)).toBe(3n);
});

test('asBigInt handles $bigint wrapper', () => {
  expect(asBigInt({ $bigint: '99' })).toBe(99n);
});

test('asBigInt handles string', () => {
  expect(asBigInt(' 123 ')).toBe(123n);
});

test('asBigInt returns 0n for empty/falsy', () => {
  expect(asBigInt('')).toBe(0n);
  expect(asBigInt(null)).toBe(0n);
  expect(asBigInt(undefined)).toBe(0n);
  expect(asBigInt('   ')).toBe(0n);
});

// ---------------------------------------------------------------------------
// toDateOnlyString
// ---------------------------------------------------------------------------

test('toDateOnlyString from Date', () => {
  expect(toDateOnlyString(new Date('2024-06-15T12:00:00Z'))).toBe('2024-06-15');
});

test('toDateOnlyString from ISO string', () => {
  expect(toDateOnlyString('2024-06-15')).toBe('2024-06-15');
  expect(toDateOnlyString('2024-06-15T12:00:00Z')).toBe('2024-06-15');
});

test('toDateOnlyString returns empty for falsy', () => {
  expect(toDateOnlyString('')).toBe('');
  expect(toDateOnlyString(null)).toBe('');
  expect(toDateOnlyString(undefined)).toBe('');
});

test('toDateOnlyString returns empty for short input', () => {
  expect(toDateOnlyString('abc')).toBe('');
});

// ---------------------------------------------------------------------------
// lifted parsers and normalizers
// ---------------------------------------------------------------------------

test('parseDecimalInput parses string/object/decimal and can reject number input', () => {
  expect(parseDecimalInput('3.14').eq(new Decimal('3.14'))).toBe(true);
  expect(parseDecimalInput({ $bigdecimal: '2.50' }).eq(new Decimal('2.5'))).toBe(true);
  expect(parseDecimalInput(new Decimal('9')).eq(new Decimal('9'))).toBe(true);
  expect(() => parseDecimalInput(1, { allowNumber: false })).toThrow('number_not_allowed');
});

test('parseDecimalInput reports required and invalid decimal codes', () => {
  try {
    parseDecimalInput('');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    parseDecimalInput('not-a-number');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_decimal');
  }
});

test('toPositiveDecimal and normalizePositiveDecimalString work for positive decimals', () => {
  expect(toPositiveDecimal('1.25').eq(new Decimal('1.25'))).toBe(true);
  expect(normalizePositiveDecimalString('2.50')).toBe('2.5');
});

test('toPositiveDecimal reports non_positive_decimal', () => {
  try {
    toPositiveDecimal('0');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('non_positive_decimal');
  }
});

test('normalizeRequiredText trims and rejects empty', () => {
  expect(normalizeRequiredText('  hello  ')).toBe('hello');
  try {
    normalizeRequiredText('   ');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }
});

test('normalizeCodeRequired trims and uppercases by default', () => {
  expect(normalizeCodeRequired('  abc  ')).toBe('ABC');
  expect(normalizeCodeRequired('  MiXeD  ', { uppercase: false })).toBe('MiXeD');
});

test('normalizeCodeRequired reports required for empty values', () => {
  try {
    normalizeCodeRequired('  ');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }
});

test('normalizeCodeOptional preserves undefined/null and normalizes strings', () => {
  expect(normalizeCodeOptional(undefined)).toBeUndefined();
  expect(normalizeCodeOptional(null)).toBeNull();
  expect(normalizeCodeOptional('  ')).toBeNull();
  expect(normalizeCodeOptional('  abc  ')).toBe('ABC');
  expect(normalizeCodeOptional('  MiXeD  ', { uppercase: false })).toBe('MiXeD');
});

test('normalizeName and requireRefId report required', () => {
  expect(normalizeName('  Name  ')).toBe('Name');
  expect(requireRefId({ Id: 'id_1' })).toBe('id_1');

  try {
    normalizeName('');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    requireRefId(null);
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }
});

test('normalizeNullableString converts nullish/empty to null', () => {
  expect(normalizeNullableString(undefined)).toBeNull();
  expect(normalizeNullableString(null)).toBeNull();
  expect(normalizeNullableString('  ')).toBeNull();
  expect(normalizeNullableString('  x  ')).toBe('x');
});

test('normalizeOptionalNonEmptyString handles optional/required/max-length semantics', () => {
  expect(normalizeOptionalNonEmptyString(undefined)).toBeUndefined();
  expect(normalizeOptionalNonEmptyString(null)).toBeUndefined();
  expect(normalizeOptionalNonEmptyString('  ok  ', { maxLength: 10 })).toBe('ok');

  try {
    normalizeOptionalNonEmptyString('   ');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    normalizeOptionalNonEmptyString('toolong', { maxLength: 3 });
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('string_too_long');
  }
});

test('isExpiredAt treats missing/invalid as expired and compares timestamps', () => {
  const now = new Date('2026-01-01T00:00:00.000Z').getTime();
  expect(isExpiredAt(undefined, now)).toBe(true);
  expect(isExpiredAt('not-a-date', now)).toBe(true);
  expect(isExpiredAt('2025-12-31T23:59:59.000Z', now)).toBe(true);
  expect(isExpiredAt('2026-01-01T00:00:01.000Z', now)).toBe(false);
});

test('roundToCurrencyAmount supports step rounding and digits fallback', () => {
  expect(roundToCurrencyAmount(new Decimal('1.23'), { DecimalDigits: 2, Rounding: '0.05' }).toString()).toBe('1.25');
  expect(roundToCurrencyAmount(new Decimal('1.2349'), { DecimalDigits: 3, Rounding: null }).toString()).toBe('1.235');
  expect(roundToCurrencyAmount(new Decimal('1.2'), { DecimalDigits: 2, Rounding: 'bad' }, 4).toString()).toBe('1.2');
});

test('parsePositiveInt parses and validates integer >= 1', () => {
  expect(parsePositiveInt('3')).toBe(3);
  try {
    parsePositiveInt('3.5');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_integer');
  }
  try {
    parsePositiveInt('0');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('integer_too_small');
  }
});

test('parseBigInt parses bigint-like values and reports invalid inputs', () => {
  expect(parseBigInt(7n)).toBe(7n);
  expect(parseBigInt(7.9)).toBe(7n);
  expect(parseBigInt({ $bigint: '8' })).toBe(8n);
  expect(parseBigInt('9')).toBe(9n);

  try {
    parseBigInt('');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    parseBigInt('x');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_bigint');
  }
});

test('normalizeDecimalDigits validates required and non-negative integer', () => {
  expect(normalizeDecimalDigits('2')).toBe(2);

  try {
    normalizeDecimalDigits('');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    normalizeDecimalDigits(-1);
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_integer');
  }
});

test('normalizeDateString validates date-only format and calendar value', () => {
  expect(normalizeDateString('2024-07-08')).toBe('2024-07-08');

  try {
    normalizeDateString('');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }

  try {
    normalizeDateString('2024/07/08');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_date_format');
  }

  try {
    normalizeDateString('2024-02-30');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_date_value');
  }
});

// ---------------------------------------------------------------------------
// normalizeEnumValue
// ---------------------------------------------------------------------------

test('normalizeEnumValue returns default for nullish/empty', () => {
  expect(normalizeEnumValue(undefined, ['a', 'b'] as const, 'a')).toBe('a');
  expect(normalizeEnumValue(null, ['a', 'b'] as const, 'b')).toBe('b');
  expect(normalizeEnumValue('', ['a', 'b'] as const, 'a')).toBe('a');
});

test('normalizeEnumValue returns matching value', () => {
  expect(normalizeEnumValue('a', ['a', 'b'] as const, 'b')).toBe('a');
  expect(normalizeEnumValue('  b  ', ['a', 'b'] as const, 'a')).toBe('b');
});

test('normalizeEnumValue throws invalid_enum_value for unknown values', () => {
  try {
    normalizeEnumValue('c', ['a', 'b'] as const, 'a');
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_enum_value');
  }
});

// ---------------------------------------------------------------------------
// resolveModelRefId
// ---------------------------------------------------------------------------

test('resolveModelRefId returns Id from relation object', () => {
  expect(resolveModelRefId({ CompanyId: { Id: 'c1' } }, 'CompanyId')).toBe('c1');
});

test('resolveModelRefId returns raw field when not an object', () => {
  expect(resolveModelRefId({ CompanyId: 'c2' }, 'CompanyId')).toBe('c2');
  expect(resolveModelRefId({ CompanyId: 42 }, 'CompanyId')).toBe(42);
});

test('resolveModelRefId returns undefined for falsy/missing input', () => {
  expect(resolveModelRefId(null, 'Field')).toBeUndefined();
  expect(resolveModelRefId(undefined, 'Field')).toBeUndefined();
  expect(resolveModelRefId({}, 'Field')).toBeUndefined();
});

test('resolveModelRefId falls back to raw field when Id property is missing', () => {
  expect(resolveModelRefId({ CompanyId: { Name: 'x' } }, 'CompanyId')).toEqual({ Name: 'x' });
});

// ---------------------------------------------------------------------------
// roundToCurrencyAmount nullable
// ---------------------------------------------------------------------------

test('roundToCurrencyAmount handles null or undefined currency by defaulting digits to 0', () => {
  const amount = new Decimal('3.14159');
  expect(roundToCurrencyAmount(amount, null).eq(new Decimal('3'))).toBe(true);
  expect(roundToCurrencyAmount(amount, undefined).eq(new Decimal('3'))).toBe(true);
});

// ---------------------------------------------------------------------------
// asRecord
// ---------------------------------------------------------------------------

test('asRecord returns null for null', () => {
  expect(asRecord(null)).toBe(null);
});

test('asRecord returns null for undefined', () => {
  expect(asRecord(undefined)).toBe(null);
});

test('asRecord returns null for arrays', () => {
  expect(asRecord([1, 2, 3])).toBe(null);
  expect(asRecord([])).toBe(null);
});

test('asRecord returns null for functions', () => {
  expect(asRecord(() => {})).toBe(null);
});

test('asRecord returns null for primitives', () => {
  expect(asRecord(42)).toBe(null);
  expect(asRecord('hello')).toBe(null);
  expect(asRecord(true)).toBe(null);
  expect(asRecord(false)).toBe(null);
});

test('asRecord returns the object for plain objects', () => {
  const obj = { a: 1 };
  expect(asRecord(obj)).toBe(obj);
  expect(asRecord({})).toEqual({});
});

test('asRecord returns the object for class instances (non-array)', () => {
  const d = new Date();
  expect(asRecord(d)).toBe(d);
});

// ---------------------------------------------------------------------------
// normalizeOptionalNonNegativeInt
// ---------------------------------------------------------------------------

test('normalizeOptionalNonNegativeInt returns undefined for null/undefined/empty', () => {
  expect(normalizeOptionalNonNegativeInt(null)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt(undefined)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt('')).toBe(undefined);
});

test('normalizeOptionalNonNegativeInt returns truncated positive integer', () => {
  expect(normalizeOptionalNonNegativeInt(42)).toBe(42);
  expect(normalizeOptionalNonNegativeInt('99')).toBe(99);
  expect(normalizeOptionalNonNegativeInt(3.9)).toBe(3);
  expect(normalizeOptionalNonNegativeInt(0)).toBe(0);
});

test('normalizeOptionalNonNegativeInt returns undefined for negative values', () => {
  expect(normalizeOptionalNonNegativeInt(-1)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt('-5')).toBe(undefined);
});

test('normalizeOptionalNonNegativeInt returns undefined for NaN/Infinity', () => {
  expect(normalizeOptionalNonNegativeInt(NaN)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt(Infinity)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt(-Infinity)).toBe(undefined);
  expect(normalizeOptionalNonNegativeInt('not a number')).toBe(undefined);
});

// ---------------------------------------------------------------------------
// optional text / translated helpers (lifted from module bridges)
// ---------------------------------------------------------------------------

test('normalizeOptionalText preserves undefined/null and clears empty', () => {
  expect(normalizeOptionalText(undefined)).toBeUndefined();
  expect(normalizeOptionalText(null)).toBeNull();
  expect(normalizeOptionalText('')).toBeNull();
  expect(normalizeOptionalText('   ')).toBeNull();
  expect(normalizeOptionalText('  abc  ')).toBe('abc');
  expect(normalizeOptionalText('abc', { upper: true })).toBe('ABC');
  expect(normalizeOptionalText('ABC', { lower: true })).toBe('abc');
});

test('normalizeOptionalRefId preserves undefined/null and resolves ids', () => {
  expect(normalizeOptionalRefId(undefined)).toBeUndefined();
  expect(normalizeOptionalRefId(null)).toBeNull();
  expect(normalizeOptionalRefId('  id1  ')).toBe('id1');
  expect(normalizeOptionalRefId({ Id: '  id2  ' })).toBe('id2');
  expect(normalizeOptionalRefId('')).toBeNull();
});

test('normalizeOptionalTranslatedText accepts scalars and lang maps', () => {
  expect(normalizeOptionalTranslatedText(undefined)).toBeUndefined();
  expect(normalizeOptionalTranslatedText(null)).toBeNull();
  expect(normalizeOptionalTranslatedText('  A  ')).toBe('A');
  expect(normalizeOptionalTranslatedText({ en_US: ' A ', zh_CN: '  ' })).toEqual({ en_US: 'A', zh_CN: '' });
});

test('normalizeRequiredTranslatedText accepts lang maps and rejects empty maps', () => {
  expect(normalizeRequiredTranslatedText({ en_US: ' Acme ', zh_CN: '' })).toEqual({
    en_US: 'Acme',
    zh_CN: '',
  });
  expect(normalizeRequiredTranslatedText(' Solo ')).toBe('Solo');
  try {
    normalizeRequiredTranslatedText({ en_US: '  ', zh_CN: '' });
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('required');
  }
});

test('translatedTextHasValue detects non-empty scalars and maps', () => {
  expect(translatedTextHasValue('x')).toBe(true);
  expect(translatedTextHasValue({ en_US: '', zh_CN: '甲' })).toBe(true);
  expect(translatedTextHasValue({ en_US: '', zh_CN: '' })).toBe(false);
  expect(translatedTextHasValue(null)).toBe(false);
});

test('normalizeNonNegativeInt and normalizeSequenceInt validate integers', () => {
  expect(normalizeNonNegativeInt(undefined)).toBeUndefined();
  expect(normalizeNonNegativeInt(null)).toBe(0);
  expect(normalizeNonNegativeInt(5)).toBe(5);
  expect(normalizeSequenceInt(undefined)).toBeUndefined();
  expect(normalizeSequenceInt(null)).toBe(10);
  expect(normalizeSequenceInt(-3)).toBe(-3);
  try {
    normalizeNonNegativeInt(-1);
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_integer');
  }
  try {
    normalizeSequenceInt(2.5);
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof NormalizationError).toBe(true);
    expect((err as NormalizationError).code).toBe('invalid_integer');
  }
});

test('normalizeOptionalTranslatedText skips blank langs and applies case options', () => {
  expect(
    normalizeOptionalTranslatedText(
      { '': 'skip', en_US: 'abc', zh_CN: undefined, fr_FR: 'x' },
      { upper: true }
    )
  ).toEqual({ en_US: 'ABC', fr_FR: 'X' });
  expect(translatedTextHasValue(42)).toBe(false);
  expect(translatedTextHasValue({ en_US: 1 as any })).toBe(false);
});
