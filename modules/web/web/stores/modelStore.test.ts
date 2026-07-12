// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { getFieldMetadataCompatibility, type FieldMetadata } from './modelStore';

describe('getFieldMetadataCompatibility', () => {
  it('exposes resolved contract keys when present', () => {
    const meta: FieldMetadata = {
      id: 'f1',
      type: 'ManyToOne',
      typeAnnotation: 'Company',
      relationModel: 'Company',
      relation: 'company_id',
      storageKind: 'column',
      shouldCreateColumn: true,
      resolvedColumnType: 'INTEGER',
      reasonCode: 'LEGACY_COLUMN',
      computedKind: 'runtime',
      relatedPath: 'CompanyId.Name',
      relatedStore: true,
      searchable: true,
      runAs: 'system',
    };

    const normalized = getFieldMetadataCompatibility(meta);

    expect(normalized.isRelation).toBe(true);
    expect(normalized.storageKind).toBe('column');
    expect(normalized.shouldCreateColumn).toBe(true);
    expect(normalized.resolvedColumnType).toBe('INTEGER');
    expect(normalized.reasonCode).toBe('LEGACY_COLUMN');
    expect(normalized.computedKind).toBe('runtime');
    expect(normalized.relatedPath).toBe('CompanyId.Name');
    expect(normalized.relatedStore).toBe(true);
    expect(normalized.searchable).toBe(true);
    expect(normalized.runAs).toBe('system');
  });

  it('keeps relation fallback based on legacy type metadata', () => {
    const meta: FieldMetadata = {
      id: 'f2',
      type: 'ManyToMany',
      typeAnnotation: 'Tag[]',
    };

    const normalized = getFieldMetadataCompatibility(meta);

    expect(normalized.isRelation).toBe(true);
    expect(normalized.relationModel).toBeUndefined();
    expect(normalized.searchable).toBeUndefined();
  });
});
