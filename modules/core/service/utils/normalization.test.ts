// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  normalizeStringArray,
  readRefId,
  normalizeRefId,
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
} from '@/core/service/utils/normalization';

test('normalizeOptionalString returns trimmed string or undefined', () => {
  expect(normalizeOptionalString('  hello  ')).toBe('hello');
  expect(normalizeOptionalString(null)).toBe(undefined);
  expect(normalizeOptionalString(undefined)).toBe(undefined);
  expect(normalizeOptionalString('')).toBe(undefined);
  expect(normalizeOptionalString('   ')).toBe(undefined);
  expect(normalizeOptionalString(123)).toBe('123');
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
