// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildChoySearchQuery,
  filterRowsByKeyword,
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
  });

  test('filterRowsByKeyword matches listed fields', () => {
    const rows = [
      { Id: '1', Name: 'Alpha', Code: 'A1' },
      { Id: '2', Name: 'Beta', Code: 'B2' },
      { Id: '3', Name: 'Gamma', Code: 'alpha-x' },
    ];
    expect(filterRowsByKeyword(rows, '', ['Name'])).toEqual(rows);
    expect(filterRowsByKeyword(rows, 'alpha', ['Name'])).toEqual([rows[0]]);
    expect(filterRowsByKeyword(rows, 'alpha', ['Name', 'Code'])).toEqual([rows[0], rows[2]]);
    expect(filterRowsByKeyword(rows, 'zzz', ['Name'])).toEqual([]);
  });
});
