// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useSearchGrouping } from './useSearchGrouping';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

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
  test('normalizes string and object group specs from props', () => {
    const groups = ref<any[]>(['Status', { field: 'CreatedAt', granularity: 'month' }, { name: 'Code' }, { prop: 'X', gran: 'year' }]);
    const onGroupsChange = fnRecorder();
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

  test('toggles plain and temporal groupby and notifies parent', () => {
    const groups = ref<any[]>(['Status']);
    const onGroupsChange = fnRecorder((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });

    api.togglePlainGroupby('Status');
    expect(onGroupsChange.calls.length).toBeGreaterThan(0);
    expect(groups.value).not.toContain('Status');

    api.togglePlainGroupby('Status');
    expect(groups.value.some((g: any) => g === 'Status' || g?.field === 'Status')).toBe(true);

    api.toggleTemporalGroupby('CreatedAt', 'month');
    expect(groups.value.some((g: any) => typeof g === 'object' && g.field === 'CreatedAt' && g.granularity === 'month')).toBe(true);

    api.toggleTemporalGroupby('CreatedAt', 'month');
    expect(groups.value.some((g: any) => typeof g === 'object' && g.field === 'CreatedAt' && g.granularity === 'month')).toBe(false);
  });

  test('toggles temporal groupby encoded as legacy string', () => {
    const groups = ref<any[]>(['CreatedAt:week']);
    const onGroupsChange = fnRecorder((next: any[]) => {
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

  test('builds appliedGroupItems for plain and temporal entries', () => {
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

  test('handles tree select change for plain, temporal, and dummy root', () => {
    const groups = ref<any[]>([]);
    const onGroupsChange = fnRecorder((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });

    api.onTreeSelectChange(undefined);
    expect(onGroupsChange.calls.length).toBe(0);

    api.onTreeSelectChange(`d:CreatedAt:${api.DUMMY_ROOT_SUFFIX}`);
    expect(onGroupsChange.calls.length).toBe(0);

    api.onTreeSelectChange('f:Status');
    expect(groups.value.some((g: any) => g === 'Status' || g?.field === 'Status')).toBe(true);

    api.onTreeSelectChange('d:CreatedAt:year');
    expect(groups.value.some((g: any) => g?.field === 'CreatedAt' && g?.granularity === 'year')).toBe(true);
  });

  test('exposes group tree data from useGroupingOptions', () => {
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

  test('skips empty group specs and labels unknown plain object fields', () => {
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => [null, {}, { field: 'Ghost' }, { field: 'Status' }] as any,
      onGroupsChange: () => {},
    });
    expect(api.currentAppliedGroups.value).toEqual(['Ghost', 'Status']);
    expect(api.appliedGroupItems.value.find(i => i.field === 'Ghost')?.label).toBe('Ghost');
    expect(api.appliedGroupItems.value.find(i => i.field === 'Status')?.label).toBe('Status');
  });

  test('toggles plain group when current list already uses object form', () => {
    const groups = ref<any[]>([{ field: 'Status' }]);
    const onGroupsChange = fnRecorder((next: any[]) => {
      groups.value = next;
    });
    const api = useSearchGrouping({
      store: makeStore(),
      currentAppliedGroups: () => groups.value,
      onGroupsChange,
    });
    api.togglePlainGroupby('Status');
    expect(groups.value.some((g: any) => g === 'Status' || g?.field === 'Status')).toBe(false);
    api.setGroupbyLocal(['Status', { field: 'CreatedAt', granularity: 'day' }]);
    expect(onGroupsChange.calls.length).toBeGreaterThan(0);
  });
});
