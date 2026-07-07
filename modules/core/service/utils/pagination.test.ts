// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';

test('normalizePagination returns safe defaults for missing or invalid input', () => {
  expect(normalizePagination()).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({})).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({ limit: null as any })).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({ offset: null as any })).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({ limit: NaN, offset: NaN })).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({ limit: -5, offset: -10 })).toEqual({ limit: undefined, offset: 0 });
  expect(normalizePagination({ limit: 0, offset: 0 })).toEqual({ limit: undefined, offset: 0 });
});

test('normalizePagination floors and returns valid limit/offset', () => {
  expect(normalizePagination({ limit: 10, offset: 5 })).toEqual({ limit: 10, offset: 5 });
  expect(normalizePagination({ limit: 10.9, offset: 5.2 })).toEqual({ limit: 10, offset: 5 });
});

test('paginateAndWrap slices with limit and offset', () => {
  const items = ['a', 'b', 'c', 'd', 'e'];
  const pagination = normalizePagination({ limit: 2, offset: 1 });

  const result = paginateAndWrap(items, 'items', pagination);

  expect(result.items).toEqual(['b', 'c']);
  expect(result.total).toBe(5);
  expect(result.filtered).toBe(5);
  expect(result.offset).toBe(1);
  expect(result.limit).toBe(2);
  expect(result.returned).toBe(2);
});

test('paginateAndWrap returns all items when limit is undefined', () => {
  const items = ['a', 'b', 'c'];
  const pagination = normalizePagination({ offset: 0 });

  const result = paginateAndWrap(items, 'data', pagination);

  expect(result.data).toEqual(['a', 'b', 'c']);
  expect(result.total).toBe(3);
  expect(result.filtered).toBe(3);
  expect(result.returned).toBe(3);
});

test('paginateAndWrap respects explicit total and extra fields', () => {
  const items = ['x', 'y'];
  const pagination = normalizePagination({ limit: 5, offset: 0 });

  const result = paginateAndWrap(items, 'results', pagination, 100, { model: 'test.Model' });

  expect(result.results).toEqual(['x', 'y']);
  expect(result.total).toBe(100);
  expect(result.filtered).toBe(2);
  expect(result.returned).toBe(2);
  expect((result as any).model).toBe('test.Model');
});

test('paginateAndWrap treats NaN total as items.length', () => {
  const items = ['a', 'b'];
  const pagination = normalizePagination({});

  const result = paginateAndWrap(items, 'items', pagination, NaN as any);

  expect(result.total).toBe(2);
});
