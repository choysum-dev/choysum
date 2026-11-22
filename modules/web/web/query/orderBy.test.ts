// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { metadataOrderByToState, orderByToSortingState, toRpcOrderBy } from './orderBy';

test('toRpcOrderBy maps FE direction onto RPC order', () => {
  expect(toRpcOrderBy([{ field: 'Name', direction: 'desc' }])).toEqual([
    { field: 'Name', order: 'desc' },
  ]);
  expect(toRpcOrderBy([{ field: 'Code', direction: 'ASC' }])).toEqual([
    { field: 'Code', order: 'asc' },
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
