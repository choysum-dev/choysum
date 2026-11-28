// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  filterOrderByForGroupKeys,
  isHandleReorderSortAllowed,
  metadataOrderByToState,
  orderByToSortingState,
  resolveTableOrderBy,
  syncControlledOrderByToQueryState,
  toRpcOrderBy,
  writeQueryStateOrderBy,
} from './orderBy';

test('toRpcOrderBy maps FE direction onto RPC order', () => {
  expect(toRpcOrderBy([{ field: 'Name', direction: 'desc' }])).toEqual([
    { field: 'Name', order: 'desc' },
  ]);
  expect(toRpcOrderBy([{ field: 'Code', direction: 'ASC' }])).toEqual([
    { field: 'Code', order: 'asc' },
  ]);
  expect(toRpcOrderBy([{ field: 'Name', direction: ' DESC ' }])).toEqual([
    { field: 'Name', order: 'desc' },
  ]);
});

test('toRpcOrderBy keeps order when already present and prefers order over direction', () => {
  expect(toRpcOrderBy([{ field: 'Name', order: 'desc' }])).toEqual([
    { field: 'Name', order: 'desc' },
  ]);
  expect(toRpcOrderBy([{ field: 'Name', order: 'asc', direction: 'desc' }])).toEqual([
    { field: 'Name', order: 'asc' },
  ]);
});

test('toRpcOrderBy drops empty fields and returns undefined for empty input', () => {
  expect(toRpcOrderBy(undefined)).toBeUndefined();
  expect(toRpcOrderBy([])).toBeUndefined();
  expect(toRpcOrderBy([{ field: '', direction: 'desc' }, null, 'x'])).toBeUndefined();
  expect(toRpcOrderBy({ field: 'Id', direction: 'desc' })).toEqual([{ field: 'Id', order: 'desc' }]);
});

test('orderByToSortingState mirrors primary orderBy for DataTable headers', () => {
  expect(orderByToSortingState(undefined)).toEqual([]);
  expect(orderByToSortingState([{ field: 'Name', direction: 'asc' }])).toEqual([
    { id: 'Name', desc: false },
  ]);
  expect(orderByToSortingState([{ field: 'CreatedAt', order: 'desc' }])).toEqual([
    { id: 'CreatedAt', desc: true },
  ]);
});

test('metadataOrderByToState maps order onto direction', () => {
  expect(metadataOrderByToState(undefined)).toBeUndefined();
  expect(metadataOrderByToState([{ field: 'Name', order: 'ASC' }])).toEqual([
    { field: 'Name', direction: 'asc' },
  ]);
  expect(metadataOrderByToState({ field: 'CreatedAt', order: 'desc' })).toEqual([
    { field: 'CreatedAt', direction: 'desc' },
  ]);
});

test('filterOrderByForGroupKeys keeps group-key rows and drops the rest', () => {
  expect(filterOrderByForGroupKeys(undefined, ['Name'])).toBeUndefined();
  expect(filterOrderByForGroupKeys([], ['Name'])).toBeUndefined();
  expect(
    filterOrderByForGroupKeys(
      [
        { field: 'CreatedAt', direction: 'asc' },
        { field: 'Name', direction: 'desc' },
      ],
      ['Name'],
    ),
  ).toEqual([{ field: 'Name', direction: 'desc' }]);
  expect(
    filterOrderByForGroupKeys([{ field: 'CreatedAt', direction: 'asc' }], ['StageId']),
  ).toBeUndefined();
  expect(
    filterOrderByForGroupKeys([{ field: 'Name', direction: 'asc' }], ['', null, undefined]),
  ).toBeUndefined();
});

test('resolveTableOrderBy prefers queryState.orderBy including explicit clear', () => {
  expect(resolveTableOrderBy(undefined, [{ field: 'Name', direction: 'desc' }])).toEqual([
    { field: 'Name', direction: 'desc' },
  ]);
  expect(resolveTableOrderBy({ keyword: '' }, [{ field: 'Name', direction: 'desc' }])).toEqual([
    { field: 'Name', direction: 'desc' },
  ]);
  expect(
    resolveTableOrderBy({ orderBy: [{ field: 'Seq', direction: 'asc' }] }, [{ field: 'Name', direction: 'desc' }])
  ).toEqual([{ field: 'Seq', direction: 'asc' }]);
  expect(resolveTableOrderBy({ orderBy: undefined }, [{ field: 'Name', direction: 'desc' }])).toBeUndefined();
  expect(resolveTableOrderBy({ orderBy: [] }, [{ field: 'Name', direction: 'desc' }])).toEqual([]);
  expect(resolveTableOrderBy({ orderBy: { field: 'Only', direction: 'asc' } }, undefined)).toEqual([
    { field: 'Only', direction: 'asc' },
  ]);
  // RPC-shaped `{ order }` rows normalize onto FE `{ direction }`.
  expect(resolveTableOrderBy({ orderBy: [{ field: 'Code', order: 'DESC' }] }, undefined)).toEqual([
    { field: 'Code', direction: 'desc' },
  ]);
});

test('syncControlledOrderByToQueryState / writeQueryStateOrderBy cover mount and clear', () => {
  const state: { queryState?: Record<string, unknown> } = { queryState: { keyword: '' } };
  syncControlledOrderByToQueryState(state, undefined);
  expect(state.queryState).toEqual({ keyword: '' });

  syncControlledOrderByToQueryState(state, [{ field: 'Name', direction: 'desc' }]);
  expect(state.queryState?.orderBy).toEqual([{ field: 'Name', direction: 'desc' }]);

  // Single-object controlled sort (ChoyListView prop union) becomes a one-item array.
  writeQueryStateOrderBy(state, { field: 'Code', order: 'DESC' });
  expect(state.queryState?.orderBy).toEqual([{ field: 'Code', direction: 'desc' }]);

  writeQueryStateOrderBy(state, []);
  expect(state.queryState?.orderBy).toEqual([]);

  writeQueryStateOrderBy(state, undefined);
  expect('orderBy' in (state.queryState || {})).toBe(true);
  expect(state.queryState?.orderBy).toBeUndefined();

  writeQueryStateOrderBy(undefined as any, [{ field: 'X', direction: 'asc' }]);
});

test('isHandleReorderSortAllowed requires sole handleField asc', () => {
  expect(isHandleReorderSortAllowed(undefined, 'Sequence')).toBe(true);
  expect(isHandleReorderSortAllowed([], 'Sequence')).toBe(true);
  expect(isHandleReorderSortAllowed([{ field: 'Sequence', direction: 'asc' }], 'Sequence')).toBe(true);
  expect(isHandleReorderSortAllowed([{ field: 'Sequence', order: 'asc' }], 'Sequence')).toBe(true);
  expect(isHandleReorderSortAllowed([{ field: 'Sequence', order: ' ASC ' }], 'Sequence')).toBe(true);
  expect(isHandleReorderSortAllowed([{ field: 'Sequence', direction: 'desc' }], 'Sequence')).toBe(false);
  // Prefer `order` over `direction` (same as toRpcOrderBy).
  expect(
    isHandleReorderSortAllowed([{ field: 'Sequence', order: 'desc', direction: 'asc' }], 'Sequence')
  ).toBe(false);
  expect(isHandleReorderSortAllowed([{ field: 'Name', direction: 'asc' }], 'Sequence')).toBe(false);
  expect(
    isHandleReorderSortAllowed(
      [
        { field: 'Sequence', direction: 'asc' },
        { field: 'Name', direction: 'asc' },
      ],
      'Sequence'
    )
  ).toBe(false);
});
