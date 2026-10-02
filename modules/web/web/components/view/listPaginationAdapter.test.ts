// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  canMountListPagination,
  listPageSizeToPaginateState,
  listPageToPaginateState,
  resolveListPaginationEffective,
  sameListPaginateState,
} from './listPaginationAdapter';

describe('listPaginationAdapter', () => {
  test('maps limit/offset to page models and paginateState', () => {
    const effective = resolveListPaginationEffective({ total: 100, limit: 20, offset: 20 });
    expect(effective.currentPage).toBe(2);
    expect(effective.pageSize).toBe(20);
    expect(canMountListPagination(effective)).toBe(true);
    expect(listPageToPaginateState(1, effective)).toEqual({ limit: 20, offset: 0 });
    expect(listPageSizeToPaginateState(50, effective)).toEqual({ limit: 50, offset: 0 });
    expect(listPageSizeToPaginateState(20, effective)).toBeNull();
    expect(listPageSizeToPaginateState(0, effective)).toBeNull();
    expect(listPageSizeToPaginateState(Number.NaN, effective)).toBeNull();
    // Out-of-range pages clamp to the last page instead of emitting an invalid offset.
    expect(listPageToPaginateState(99, effective)).toEqual({ limit: 20, offset: 80 });
    // Controlled page/pageSize take precedence over limit/offset.
    const controlled = resolveListPaginationEffective({ total: 100, page: 3, pageSize: 50 });
    expect(controlled.currentPage).toBe(3);
    expect(controlled.pageSize).toBe(50);
    expect(sameListPaginateState({ limit: 20, offset: 0 }, { limit: 20, offset: 0 })).toBe(true);
    expect(sameListPaginateState({ limit: 20, offset: 0 }, { limit: 50, offset: 0 })).toBe(false);
  });

  test('normalizes non-positive or non-finite limits to 20', () => {
    expect(resolveListPaginationEffective({ total: 10, limit: -5, offset: 0 }).pageSize).toBe(20);
    expect(resolveListPaginationEffective({ total: 10, limit: 0, offset: 0 }).pageSize).toBe(20);
    expect(resolveListPaginationEffective({ total: 10, limit: Number.NaN, offset: 0 }).pageSize).toBe(20);
    expect(resolveListPaginationEffective({ total: 10, limit: Number.POSITIVE_INFINITY, offset: 0 }).pageSize).toBe(
      20,
    );
  });

  test('defers mount while total is unknown and offset is restored', () => {
    const pending = resolveListPaginationEffective({ total: 0, limit: 20, offset: 40 });
    expect(canMountListPagination(pending)).toBe(false);
    const ready = resolveListPaginationEffective({ total: 100, limit: 20, offset: 40 });
    expect(canMountListPagination(ready)).toBe(true);
    expect(ready.currentPage).toBe(3);
  });
});
