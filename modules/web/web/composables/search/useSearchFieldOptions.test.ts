// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createTermReference } from '@/core/service/i18n';
import { isFilterableSearchField, isGroupableSearchField, listSearchFieldOptions } from './useSearchFieldOptions';

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
    expect(isGroupableSearchField('Lines', store.fieldsMetadata.Lines)).toBe(false);

    const filterable = listSearchFieldOptions(store, isFilterableSearchField);
    expect(filterable.map(f => f.prop)).toEqual(['Status', 'PartnerId']);
    expect(filterable[0]?.label).toBe('状态');
  });
});
