// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  canonicalizeLogicalMethodName,
  isRegisteredLogicalModelName,
  listLogicalModelNames,
  logicalMethodsAllow,
  assertLogicalMethods,
  assertLogicalModelName,
} from '@/auth/service/models/_logical_model_registry';

test('logical model names come from core platform-inject self-registration', () => {
  expect(listLogicalModelNames()).toEqual(['AppSetting', 'FieldDefault', 'PropertyDefinition', 'TranslationTerm']);
  expect(isRegisteredLogicalModelName('TranslationTerm')).toBe(true);
  expect(isRegisteredLogicalModelName('PropertyDefinition')).toBe(true);
  expect(isRegisteredLogicalModelName('Partner')).toBe(false);
  expect(isRegisteredLogicalModelName('')).toBe(false);
});

test('assertLogicalModelName rejects unregistered names', () => {
  expect(assertLogicalModelName(null)).toBe(null);
  expect(assertLogicalModelName('  ')).toBe(null);
  expect(assertLogicalModelName('FieldDefault')).toBe('FieldDefault');
  expect(() => assertLogicalModelName('Partner')).toThrow(/not a registered logical model/);
});

test('assertLogicalMethods canonicalizes PascalCase and dedupes case-insensitively', () => {
  expect(assertLogicalMethods(null)).toBe(null);
  expect(assertLogicalMethods([])).toBe(null);
  expect(assertLogicalMethods(['search', 'Browse', 'SEARCH'])).toEqual(['Search', 'Browse']);
  expect(canonicalizeLogicalMethodName('getEffective')).toBe('GetEffective');
  expect(canonicalizeLogicalMethodName('   ')).toBe('');
  expect(canonicalizeLogicalMethodName(null)).toBe('');
  expect(canonicalizeLogicalMethodName(undefined)).toBe('');
  expect(assertLogicalMethods('["Update"]')).toEqual(['Update']);
  expect(assertLogicalMethods('  ')).toBe(null);
  expect(assertLogicalMethods(['Search', '  '])).toEqual(['Search']);
  expect(() => assertLogicalMethods('not-json')).toThrow(/must be a JSON string array/);
  expect(() => assertLogicalMethods('{}')).toThrow(/must be a JSON string array/);
  expect(() => assertLogicalMethods(42 as any)).toThrow(/must be a string array/);
  expect(() => assertLogicalMethods([1 as any])).toThrow(/each entry must be a string/);
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
    { value: 'PropertyDefinition', label: 'PropertyDefinition' },
    { value: 'TranslationTerm', label: 'TranslationTerm' },
  ]);
});
