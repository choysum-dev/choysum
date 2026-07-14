// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { getFieldMetadataView, type FieldMetadata } from './modelStore';

describe('getFieldMetadataView', () => {
  it('exposes resolved contract keys when present', () => {
    const meta: FieldMetadata = {
      id: 'f1',
      type: 'ManyToOne',
      typeAnnotation: 'Company',
      relationModel: 'Company',
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

    const normalized = getFieldMetadataView(meta);

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

  it('marks relation only by declared relation field types', () => {
    const meta: FieldMetadata = {
      id: 'f2',
      type: 'ManyToMany',
      typeAnnotation: 'Tag[]',
    };

    const normalized = getFieldMetadataView(meta);

    expect(normalized.isRelation).toBe(true);
    expect(normalized.relationModel).toBeUndefined();
    expect(normalized.searchable).toBeUndefined();
  });

  it('does not infer relation from relationModel alone without relation type', () => {
    const meta: FieldMetadata = {
      id: 'f3',
      type: 'String',
      typeAnnotation: 'string',
      relationModel: 'Company',
    };

    const normalized = getFieldMetadataView(meta);

    expect(normalized.relationModel).toBe('Company');
    expect(normalized.isRelation).toBe(false);
  });
});
