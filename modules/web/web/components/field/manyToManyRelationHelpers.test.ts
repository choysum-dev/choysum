// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildExcludePickedCondition,
  buildM2MEffectiveCondition,
  resolveRelationRecordLabel,
  resetM2MPickerQueryState,
  unwrapListSelectionRecords,
} from './manyToManyRelationHelpers';

test('buildExcludePickedCondition returns undefined when empty', () => {
  expect(buildExcludePickedCondition([])).toBeUndefined();
});

test('buildExcludePickedCondition builds Id not in', () => {
  expect(buildExcludePickedCondition(['a', 'a', 'b'])).toEqual(['Id', 'not in', ['a', 'b']]);
});

test('buildM2MEffectiveCondition Ands exclude, external leaf, and onchange', () => {
  expect(
    buildM2MEffectiveCondition({
      selectedIds: ['1'],
      external: ['Active', '=', true] as any,
      onchangeParts: [['Code', '=', 'X'] as any],
    })
  ).toEqual({
    And: [
      ['Id', 'not in', ['1']],
      ['Active', '=', true],
      ['Code', '=', 'X'],
    ],
  });
});

test('buildM2MEffectiveCondition treats a leaf triple as one condition, not three parts', () => {
  expect(buildM2MEffectiveCondition({ external: ['Active', '=', true] as any })).toEqual([
    'Active',
    '=',
    true,
  ]);
});

test('buildM2MEffectiveCondition returns [] when nothing to merge', () => {
  expect(buildM2MEffectiveCondition({})).toEqual([]);
});

test('resolveRelationRecordLabel prefers DisplayName then fallbacks', () => {
  expect(resolveRelationRecordLabel({ Id: '1', DisplayName: 'A', Name: 'B' })).toBe('A');
  expect(resolveRelationRecordLabel({ Id: '1', Name: 'B' })).toBe('B');
  expect(resolveRelationRecordLabel({ Id: '1' })).toBe('1');
});

test('resetM2MPickerQueryState clears keyword and appliedFilters', () => {
  const store = {
    state: {
      queryState: {
        keyword: 'x',
        appliedFilters: [{ And: [] }],
        appliedGroups: [['Code']],
        pagination: { limit: 20, offset: 40 },
        defaultFilters: [{ name: 'A' }],
      },
    },
  };
  resetM2MPickerQueryState(store);
  expect(store.state.queryState.keyword).toBeUndefined();
  expect(store.state.queryState.appliedFilters).toEqual([]);
  expect(store.state.queryState.appliedGroups).toEqual([]);
  expect(store.state.queryState.pagination).toEqual({ limit: 20, offset: 0 });
  expect(store.state.queryState.defaultFilters).toEqual([{ name: 'A' }]);
});

test('unwrapListSelectionRecords flattens wrappers', () => {
  expect(
    unwrapListSelectionRecords([{ kind: 'record', payload: { Id: '1' } }, { type: 'record', record: { Id: '2' } }, { Id: '3' }])
  ).toEqual([{ Id: '1' }, { Id: '2' }, { Id: '3' }]);
});

test('unwrapListSelectionRecords peels { value: T[] } expose stubs', () => {
  expect(
    unwrapListSelectionRecords({
      value: [{ kind: 'record', payload: { Id: '1' } }, { Id: '2' }],
    })
  ).toEqual([{ Id: '1' }, { Id: '2' }]);
});
