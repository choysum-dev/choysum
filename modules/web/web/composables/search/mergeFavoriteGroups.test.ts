// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mergeFavoriteGroups, removeFavoriteGroups } from './mergeFavoriteGroups';

function fieldsOf(groups: Array<{ field: string; granularity?: string }>) {
  return groups.map(g => (g.granularity ? `${g.field}:${g.granularity}` : g.field));
}

describe('mergeFavoriteGroups', () => {
  test('merges favorite groups onto current and overwrites same field', () => {
    const out = mergeFavoriteGroups(
      [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
      [{ field: 'CreatedAt', granularity: 'year' }, { field: 'Name' }]
    );
    expect(fieldsOf(out)).toEqual(['Status', 'CreatedAt:year', 'Name']);
  });

  test('handles empty sides', () => {
    expect(fieldsOf(mergeFavoriteGroups([], [{ field: 'A' }]))).toEqual(['A']);
    expect(fieldsOf(mergeFavoriteGroups([{ field: 'A' }], []))).toEqual(['A']);
    expect(mergeFavoriteGroups(null, null)).toEqual([]);
  });
});

describe('removeFavoriteGroups', () => {
  test('removes exact favorite contributions only', () => {
    const out = removeFavoriteGroups(
      [
        { field: 'CreatedAt', granularity: 'year' },
        { field: 'Status' },
        { field: 'CreatedAt', granularity: 'month' },
      ],
      [{ field: 'CreatedAt', granularity: 'year' }]
    );
    expect(fieldsOf(out)).toEqual(['Status', 'CreatedAt:month']);
  });
});
