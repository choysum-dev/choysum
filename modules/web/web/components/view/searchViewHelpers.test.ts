// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildChoySearchQuery,
  choySearchQueryFromPayload,
  filterRowsByKeyword,
  flattenChoySearchFilters,
  normalizeChoySearchKeyword,
} from './searchViewHelpers';

describe('searchViewHelpers', () => {
  test('normalizeChoySearchKeyword trims', () => {
    expect(normalizeChoySearchKeyword(null)).toBe('');
    expect(normalizeChoySearchKeyword(undefined)).toBe('');
    expect(normalizeChoySearchKeyword('  acme  ')).toBe('acme');
  });

  test('buildChoySearchQuery normalizes keyword and filters', () => {
    expect(buildChoySearchQuery('  hi  ')).toEqual({ keyword: 'hi', filters: [] });
    expect(
      buildChoySearchQuery('x', [{ field: ' name ', op: ' = ', value: 'a' }]),
    ).toEqual({
      keyword: 'x',
      filters: [{ field: 'name', op: '=', value: 'a' }],
    });
    expect(
      buildChoySearchQuery('x', [
        { field: '  ', op: '=', value: 'skip' },
        { field: 'code', op: '=', value: 'ok' },
      ]),
    ).toEqual({
      keyword: 'x',
      filters: [{ field: 'code', op: '=', value: 'ok' }],
    });
  });

  test('flattenChoySearchFilters walks nested ConditionGroups', () => {
    expect(flattenChoySearchFilters(null)).toEqual([]);
    expect(
      flattenChoySearchFilters([
        {
          children: [
            { field: ' Name ', operator: ' = ', value: 'a' },
            { children: [{ field: 'code', operator: 'ilike', value: 'x' }] },
            { field: '', operator: '=', value: 'skip' },
          ],
        },
      ]),
    ).toEqual([
      { field: 'Name', op: '=', value: 'a' },
      { field: 'code', op: 'ilike', value: 'x' },
    ]);
    // Singleton group object (not wrapped in an array) still flattens.
    expect(
      flattenChoySearchFilters({
        children: [{ field: 'name', operator: '=', value: 'solo' }],
      }),
    ).toEqual([{ field: 'name', op: '=', value: 'solo' }]);
    // Leaf with empty children array must not be dropped as an empty group.
    expect(
      flattenChoySearchFilters([
        { field: 'code', operator: '=', value: 'x', children: [] },
      ]),
    ).toEqual([{ field: 'code', op: '=', value: 'x' }]);
  });

  test('choySearchQueryFromPayload adapts SearchView payload', () => {
    const appliedFilters = [{ children: [{ field: 'name', operator: '=', value: 'a' }] }];
    const appliedGroups = [{ field: 'Revoked' }];
    expect(
      choySearchQueryFromPayload({
        keyword: '  hi  ',
        appliedFilters,
        appliedGroups,
      }),
    ).toEqual({
      keyword: 'hi',
      filters: [{ field: 'name', op: '=', value: 'a' }],
      appliedFilters,
      appliedGroups,
    });
    // Singleton filter tree object is preserved as a one-element appliedFilters array.
    const solo = { children: [{ field: 'code', operator: '=', value: 'x' }] };
    expect(choySearchQueryFromPayload({ keyword: '', appliedFilters: solo })).toEqual({
      keyword: '',
      filters: [{ field: 'code', op: '=', value: 'x' }],
      appliedFilters: [solo],
    });
  });

  test('filterRowsByKeyword matches listed fields', () => {
    const rows = [
      { Id: '1', Name: 'Alpha', Code: 'A1' },
      { Id: '2', Name: 'Beta', Code: 'B2' },
      { Id: '3', Name: 'Gamma', Code: 'alpha-x' },
      { Id: '4', Name: null as unknown as string, Code: 'Z' },
    ];
    expect(filterRowsByKeyword(rows, '', ['Name'])).toEqual(rows);
    expect(filterRowsByKeyword(rows, 'alpha', [])).toEqual(rows);
    expect(filterRowsByKeyword(rows, 'alpha', ['  ', ''])).toEqual(rows);
    expect(filterRowsByKeyword(rows, 'alpha', ['Name'])).toEqual([rows[0]]);
    expect(filterRowsByKeyword(rows, 'alpha', ['Name', 'Code'])).toEqual([rows[0], rows[2]]);
    expect(filterRowsByKeyword(rows, 'zzz', ['Name'])).toEqual([]);
    // Non-primitive cells are ignored (avoid matching "[object Object]" / Date strings).
    const mixed = [
      { Id: '1', Name: 'Alpha', Meta: { nested: true } },
      { Id: '2', Name: 'Beta', Meta: 'object' },
    ];
    expect(filterRowsByKeyword(mixed, 'object', ['Meta'])).toEqual([mixed[1]]);
    // Primitive number/boolean cells are matched through String() coercion.
    const primitives = [
      { Id: '1', Active: true, Qty: 12 },
      { Id: '2', Active: false, Qty: 7 },
    ];
    expect(filterRowsByKeyword(primitives, 'true', ['Active'])).toEqual([primitives[0]]);
    expect(filterRowsByKeyword(primitives, '12', ['Qty'])).toEqual([primitives[0]]);
  });
});
