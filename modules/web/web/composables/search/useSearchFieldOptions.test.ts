// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createTermReference } from '@/core/service/i18n';
import {
  isFilterableSearchField,
  isGroupableSearchField,
  listSearchFieldOptions,
  resolveSearchFieldLabel,
  sortSearchFieldOptions,
  useFilterableSearchFields,
} from './useSearchFieldOptions';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    getGlobalComposer: () => ({
      t: (_key: string, fallback: string) => (fallback === 'Status' ? '状态' : fallback),
    }),
  };
});

describe('useSearchFieldOptions', () => {
  it('lists filterable fields with resolved labels and excludes collections', () => {
    const statusText = createTermReference('demo', 'Status', { scope: 'demo.model.Widget.fields' });
    const store = {
      fieldsMetadata: {
        Status: { id: '1', type: 'selection', string: 'Status', stringText: statusText },
        Lines: { id: '2', type: 'onetomany' },
        PartnerId: { id: '3', type: 'manytoone', relationModel: 'base.Partner' },
        DeletedAt: { id: '9', type: 'datetime' },
      },
      getFieldsGetTranslatedString: () => undefined,
    } as any;

    expect(isFilterableSearchField('Lines', store.fieldsMetadata.Lines)).toBe(false);
    expect(isFilterableSearchField('PartnerId', store.fieldsMetadata.PartnerId)).toBe(true);
    expect(isFilterableSearchField('Id', undefined)).toBe(true);
    expect(isFilterableSearchField('DeletedAt', store.fieldsMetadata.DeletedAt)).toBe(false);
    expect(isGroupableSearchField('Lines', store.fieldsMetadata.Lines)).toBe(false);
    expect(isGroupableSearchField('Status', store.fieldsMetadata.Status)).toBe(true);
    expect(isGroupableSearchField('DeletedAt', store.fieldsMetadata.DeletedAt)).toBe(false);

    const filterable = listSearchFieldOptions(store, isFilterableSearchField);
    expect(filterable.map(f => f.prop)).toEqual(['Status', 'PartnerId']);
    expect(filterable[0]?.label).toBe('状态');
  });

  it('sorts by id then label and resolves labels via helper APIs', () => {
    const sorted = sortSearchFieldOptions([
      { prop: 'B', label: 'Beta', id: '2' },
      { prop: 'A', label: 'Alpha', id: '1' },
      { prop: 'C', label: 'Charlie' },
    ]);
    expect(sorted.map(i => i.prop)).toEqual(['C', 'A', 'B']);

    const byIdOf = sortSearchFieldOptions(
      [
        { prop: 'Z', label: 'Zed' },
        { prop: 'A', label: 'Aye' },
      ],
      prop => (prop === 'A' ? '1' : '2')
    );
    expect(byIdOf.map(i => i.prop)).toEqual(['A', 'Z']);

    const store = {
      fieldsMetadata: {
        Name: { id: '1', type: 'varchar', string: 'Name' },
        Ref: { id: '2', type: 'manytooneref', relationModel: 'base.X' },
        Tags: { id: '3', type: 'manytomany' },
        Blob: { id: '4', type: 'jsonobject' },
      },
      getFieldsGetTranslatedString: (p: string) => (p === 'Name' ? '名称' : undefined),
    } as any;
    expect(isFilterableSearchField('Ref', store.fieldsMetadata.Ref)).toBe(true);
    expect(isGroupableSearchField('Tags', store.fieldsMetadata.Tags)).toBe(false);
    expect(isGroupableSearchField('Blob', store.fieldsMetadata.Blob)).toBe(false);

    const fields = useFilterableSearchFields(store);
    expect(fields.value.find(f => f.prop === 'Name')?.label).toBe('名称');
    expect(resolveSearchFieldLabel(undefined, 'Name')).toBe('Name');
    expect(resolveSearchFieldLabel(store, '')).toBe('');
    expect(resolveSearchFieldLabel(store, 'Name')).toBe('名称');
  });
});
