// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createEmptyQueryState, resetRelationPickerQueryState } from './state';

test('resetRelationPickerQueryState clears user search UI and keeps defaults', () => {
  const store = {
    state: {
      queryState: {
        ...createEmptyQueryState(),
        keyword: '启用',
        keywordFields: ['Name', 'Code'],
        appliedFilters: [{ And: [['Name', 'ilike', 'x']] }] as any,
        appliedGroups: [['Code']] as any,
        defaultFilters: [{ name: 'Active', query: ['IsActive', '=', true] }] as any,
        defaultGroups: [{ name: 'ByCode', groupby: ['Code'] }] as any,
        forcedCondition: ['IsActive', '=', true] as any,
        relationConditionSource: { model: 'sale.Order', field: 'CategoryId' },
        userClearedDefaultFilters: true,
        userClearedDefaultGroups: true,
        pagination: { limit: 25, offset: 75 },
        orderBy: [{ field: 'Name', direction: 'desc' as const }],
      },
    },
  };

  resetRelationPickerQueryState(store);

  const qs = store.state.queryState as any;
  expect(qs.keyword).toBeUndefined();
  expect(qs.appliedFilters).toEqual([]);
  expect(qs.appliedGroups).toEqual([]);
  expect(qs.forcedCondition).toBeUndefined();
  expect(qs.relationConditionSource).toBeUndefined();
  expect(qs.userClearedDefaultFilters).toBeUndefined();
  expect(qs.userClearedDefaultGroups).toBeUndefined();
  expect(qs.pagination).toEqual({ limit: 25, offset: 0 });
  expect(qs.keywordFields).toEqual(['Name', 'Code']);
  expect(qs.defaultFilters).toEqual([{ name: 'Active', query: ['IsActive', '=', true] }]);
  expect(qs.defaultGroups).toEqual([{ name: 'ByCode', groupby: ['Code'] }]);
  expect(qs.orderBy).toEqual([{ field: 'Name', direction: 'desc' }]);
});

test('resetRelationPickerQueryState no-ops without store state', () => {
  expect(() => resetRelationPickerQueryState(undefined)).not.toThrow();
  expect(() => resetRelationPickerQueryState({})).not.toThrow();
  expect(() => resetRelationPickerQueryState({ state: {} })).not.toThrow();
});
