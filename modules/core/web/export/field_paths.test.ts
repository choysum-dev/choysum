// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { normalizeExportFieldPath, normalizeExportFieldPaths } from './field_paths';

describe('normalizeExportFieldPath', () => {
  it('converts dot-separated paths to slash-separated', () => {
    expect(normalizeExportFieldPath('CompanyId.Code')).toBe('CompanyId/Code');
  });

  it('returns empty for blank paths', () => {
    expect(normalizeExportFieldPath('')).toBe('');
    expect(normalizeExportFieldPath('   ')).toBe('');
  });
});

describe('normalizeExportFieldPaths', () => {
  it('deduplicates and skips Id', () => {
    expect(normalizeExportFieldPaths(['Name', 'Id', 'Name', 'CompanyId.Code'])).toEqual(['Name', 'CompanyId/Code']);
  });

  it('handles nullish input', () => {
    expect(normalizeExportFieldPaths(null)).toEqual([]);
    expect(normalizeExportFieldPaths(undefined)).toEqual([]);
  });

  it('skips blank normalized paths', () => {
    expect(normalizeExportFieldPaths(['', '   ', 'Name'])).toEqual(['Name']);
  });

  it('leaves slash paths unchanged', () => {
    expect(normalizeExportFieldPath('CompanyId/Code')).toBe('CompanyId/Code');
  });
});
