// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';
import { useSearchGrouping } from './useSearchGrouping';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
    }),
    getGlobalComposer: () => ({
      t: (_key: string, fallback: string) => fallback,
    }),
  };
});

function makeStore() {
  return {
    fieldsMetadata: {
      Status: { id: '1', type: 'selection', string: 'Status' },
      CreatedAt: { id: '2', type: 'datetime', string: 'Created At' },
      Lines: { id: '3', type: 'onetomany', string: 'Lines' },
      DeletedAt: { id: '9', type: 'datetime', string: 'Deleted At' },
    },
    getFieldsGetTranslatedString: () => undefined,
  } as any;
}

describe('useSearchGrouping', () => {
  it('normalizes string and object group specs from props', () => {
    const groups = ref<any[]>(['Status', { field: 'CreatedAt', granularity: 'month' }, { name: 'Code' }, { prop: 'X', gran: 'year' }]);
    const onGroupsChange = vi.fn();
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });
    expect(api.currentAppliedGroups.value).toEqual([
      'Status',
      { field: 'CreatedAt', granularity: 'month' },
      'Code',
      { field: 'X', granularity: 'year' },
    ]);
  });

  it('toggles plain and temporal groupby and notifies parent', () => {
    const groups = ref<any[]>(['Status']);
    const onGroupsChange = vi.fn((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });

    api.togglePlainGroupby('Status');
    expect(onGroupsChange).toHaveBeenCalled();
    expect(groups.value).not.toContain('Status');

    api.togglePlainGroupby('Status');
    expect(groups.value.some((g: any) => g === 'Status' || g?.field === 'Status')).toBe(true);

    api.toggleTemporalGroupby('CreatedAt', 'month');
    expect(groups.value.some((g: any) => typeof g === 'object' && g.field === 'CreatedAt' && g.granularity === 'month')).toBe(true);

    api.toggleTemporalGroupby('CreatedAt', 'month');
    expect(groups.value.some((g: any) => typeof g === 'object' && g.field === 'CreatedAt' && g.granularity === 'month')).toBe(false);
  });

  it('toggles temporal groupby encoded as legacy string', () => {
    const groups = ref<any[]>(['CreatedAt:week']);
    const onGroupsChange = vi.fn((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });
    api.toggleTemporalGroupby('CreatedAt', 'week');
    expect(groups.value.some((g: any) => g === 'CreatedAt:week' || (g?.field === 'CreatedAt' && g?.granularity === 'week'))).toBe(false);
  });

  it('builds appliedGroupItems for plain and temporal entries', () => {
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => ['Status', { field: 'CreatedAt', granularity: 'day' }, 'CreatedAt:month'] as any,
      onGroupsChange: () => {},
    });
    const keys = api.appliedGroupItems.value.map(i => i.key);
    expect(keys).toContain('cur:plain:Status');
    expect(keys).toContain('cur:temp:CreatedAt:day');
    expect(keys).toContain('cur:temp:CreatedAt:month');
    expect(api.appliedGroupItems.value.find(i => i.field === 'Status')?.label).toBe('Status');
  });

  it('handles tree select change for plain, temporal, and dummy root', () => {
    const groups = ref<any[]>([]);
    const onGroupsChange = vi.fn((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });

    api.onTreeSelectChange(undefined);
    expect(onGroupsChange).not.toHaveBeenCalled();

    api.onTreeSelectChange(`d:CreatedAt:${api.DUMMY_ROOT_SUFFIX}`);
    expect(onGroupsChange).not.toHaveBeenCalled();

    api.onTreeSelectChange('f:Status');
    expect(groups.value.some((g: any) => g === 'Status' || g?.field === 'Status')).toBe(true);

    api.onTreeSelectChange('d:CreatedAt:year');
    expect(groups.value.some((g: any) => g?.field === 'CreatedAt' && g?.granularity === 'year')).toBe(true);
  });

  it('exposes group tree data from useGroupingOptions', () => {
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => [],
      onGroupsChange: () => {},
    });
    expect(api.groupTreeData.value.some(n => n.id === 'f:Status')).toBe(true);
    expect(api.groupTreeData.value.some(n => n.id === 'd:CreatedAt')).toBe(true);
    expect(api.treeProps.value).toBe('value');
    expect(api.temporalComboLabel('CreatedAt', 'month')).toContain('Month');
  });
});
