// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  canonicalizeLogicalMethodName,
  isRegisteredLogicalModelName,
  listLogicalModelNames,
  logicalMethodsAllow,
  normalizeLogicalMethods,
  normalizeLogicalModelName,
} from '@/auth/service/models/_logical_model_registry';

test('logical model names come from core platform-inject self-registration', () => {
  expect(listLogicalModelNames()).toEqual(['AppSetting', 'FieldDefault', 'TranslationTerm']);
  expect(isRegisteredLogicalModelName('TranslationTerm')).toBe(true);
  expect(isRegisteredLogicalModelName('Partner')).toBe(false);
  expect(isRegisteredLogicalModelName('')).toBe(false);
});

test('normalizeLogicalModelName rejects unregistered names', () => {
  expect(normalizeLogicalModelName(null)).toBe(null);
  expect(normalizeLogicalModelName('  ')).toBe(null);
  expect(normalizeLogicalModelName('FieldDefault')).toBe('FieldDefault');
  expect(() => normalizeLogicalModelName('Partner')).toThrow(/not a registered logical model/);
});

test('normalizeLogicalMethods canonicalizes PascalCase and dedupes case-insensitively', () => {
  expect(normalizeLogicalMethods(null)).toBe(null);
  expect(normalizeLogicalMethods([])).toBe(null);
  expect(normalizeLogicalMethods(['search', 'Browse', 'SEARCH'])).toEqual(['Search', 'Browse']);
  expect(canonicalizeLogicalMethodName('getEffective')).toBe('GetEffective');
  expect(canonicalizeLogicalMethodName('   ')).toBe('');
  expect(canonicalizeLogicalMethodName(null)).toBe('');
  expect(canonicalizeLogicalMethodName(undefined)).toBe('');
  expect(normalizeLogicalMethods('["Update"]')).toEqual(['Update']);
  expect(normalizeLogicalMethods('  ')).toBe(null);
  expect(normalizeLogicalMethods(['Search', '  '])).toEqual(['Search']);
  expect(() => normalizeLogicalMethods('not-json')).toThrow(/must be a JSON string array/);
  expect(() => normalizeLogicalMethods('{}')).toThrow(/must be a JSON string array/);
  expect(() => normalizeLogicalMethods(42 as any)).toThrow(/must be a string array/);
  expect(() => normalizeLogicalMethods([1 as any])).toThrow(/each entry must be a string/);
});

test('logicalMethodsAllow treats null/empty as all methods', () => {
  expect(logicalMethodsAllow(null, 'Search')).toBe(true);
  expect(logicalMethodsAllow([], 'Search')).toBe(true);
  expect(logicalMethodsAllow(['Search', 'Browse'], 'update')).toBe(false);
  expect(logicalMethodsAllow(['Search', 'Browse'], 'SEARCH')).toBe(true);
  expect(logicalMethodsAllow(null, '')).toBe(false);
  expect(logicalMethodsAllow(['Search'], '   ')).toBe(false);
  expect(() => logicalMethodsAllow('{}', 'Search')).toThrow(/must be a JSON string array/);
});

test('listLogicalModelSelection is re-exported for admin FieldsGet', async () => {
  const { listLogicalModelSelection } = await import('@/auth/service/models/_logical_model_registry');
  expect(listLogicalModelSelection()).toEqual([
    { value: 'AppSetting', label: 'AppSetting' },
    { value: 'FieldDefault', label: 'FieldDefault' },
    { value: 'TranslationTerm', label: 'TranslationTerm' },
  ]);
});
