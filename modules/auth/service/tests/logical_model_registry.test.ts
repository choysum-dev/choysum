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
  expect(normalizeLogicalMethods('["Update"]')).toEqual(['Update']);
  expect(() => normalizeLogicalMethods('not-json')).toThrow(/must be a JSON string array/);
});

test('logicalMethodsAllow treats null/empty as all methods', () => {
  expect(logicalMethodsAllow(null, 'Search')).toBe(true);
  expect(logicalMethodsAllow([], 'Search')).toBe(true);
  expect(logicalMethodsAllow(['Search', 'Browse'], 'update')).toBe(false);
  expect(logicalMethodsAllow(['Search', 'Browse'], 'SEARCH')).toBe(true);
});
