// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

vi.mock('@/core/service/utils/normalization', () => ({
  normalizeFields: (fields?: string[]) => (Array.isArray(fields) ? fields.filter(Boolean) : []),
}));

import { buildKeywordCondition, deriveKeywordFieldsFromMeta, resolveKeywordFieldsByMeta } from './builder';

describe('deriveKeywordFieldsFromMeta', () => {
  it('prefers explicit searchable flags over legacy type fallback', () => {
    const fieldsMeta = {
      Name: { type: 'Char', searchable: false },
      Description: { type: 'Char' },
      Code: { type: 'Integer', searchable: true },
    };

    const fields = deriveKeywordFieldsFromMeta(fieldsMeta, { textTypes: ['char', 'varchar'] });

    expect(fields).toContain('Description');
    expect(fields).toContain('Code');
    expect(fields).not.toContain('Name');
  });

  it('falls back to derived searchable fields when preferred fields are filtered out', () => {
    const fieldsMeta = {
      Name: { type: 'Char' },
      Title: { type: 'Varchar', searchable: true },
      Amount: { type: 'Decimal', searchable: false },
    };

    const resolved = resolveKeywordFieldsByMeta(['UnknownField'], {
      fieldsMeta,
      fallbackWhenFilteredEmpty: true,
      fallbackTextTypes: ['char', 'varchar'],
    });

    expect(resolved).toContain('Name');
    expect(resolved).toContain('Title');
    expect(resolved).not.toContain('Amount');
  });
});

describe('buildKeywordCondition', () => {
  it('returns null when metadata marks all fields as non-searchable', () => {
    const fieldsMeta = {
      Name: { type: 'Char', searchable: false },
      Notes: { type: 'Varchar', searchable: false },
    };

    const condition = buildKeywordCondition('abc', undefined, {
      fieldsMeta,
      fallbackWhenFilteredEmpty: true,
      fallbackTextTypes: ['char', 'varchar'],
    });

    expect(condition).toBeNull();
  });
});
