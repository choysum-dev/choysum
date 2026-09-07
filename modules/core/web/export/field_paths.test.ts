// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeExportFieldPath, normalizeExportFieldPaths } from './field_paths';

test('normalizeExportFieldPath: converts dot-separated paths to slash-separated', () => {
  expect(normalizeExportFieldPath('CompanyId.Code')).toBe('CompanyId/Code');
});

test('normalizeExportFieldPath: returns empty for blank paths', () => {
  expect(normalizeExportFieldPath('')).toBe('');
  expect(normalizeExportFieldPath('   ')).toBe('');
});

test('normalizeExportFieldPaths: deduplicates and skips Id', () => {
  expect(normalizeExportFieldPaths(['Name', 'Id', 'Name', 'CompanyId.Code'])).toEqual(['Name', 'CompanyId/Code']);
});

test('normalizeExportFieldPaths: handles nullish input', () => {
  expect(normalizeExportFieldPaths(null)).toEqual([]);
  expect(normalizeExportFieldPaths(undefined)).toEqual([]);
});

test('normalizeExportFieldPaths: skips blank normalized paths', () => {
  expect(normalizeExportFieldPaths(['', '   ', 'Name'])).toEqual(['Name']);
});

test('normalizeExportFieldPaths: leaves slash paths unchanged', () => {
  expect(normalizeExportFieldPath('CompanyId/Code')).toBe('CompanyId/Code');
});

test('normalizeExportFieldPaths: normalizes null path input', () => {
  expect(normalizeExportFieldPath(null as unknown as string)).toBe('');
});

test('normalizeExportFieldPaths: skips duplicate paths after normalization', () => {
  expect(normalizeExportFieldPaths(['CompanyId.Code', 'CompanyId/Code'])).toEqual(['CompanyId/Code']);
});
