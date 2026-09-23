// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  findRelationOption,
  mapNameSearchRows,
  normalizeRelationQuery,
  runRelationNameSearch,
  upsertRelationOption,
} from './relationComboboxHelpers';

describe('relationComboboxHelpers', () => {
  test('normalizes query and maps NameSearch rows', () => {
    expect(normalizeRelationQuery('  alice  ')).toBe('alice');
    expect(
      mapNameSearchRows([
        { Id: 'p1', DisplayName: 'Alice' },
        { id: 'p2', label: 'Bob' },
        { Id: '', DisplayName: 'Skip' },
      ]),
    ).toEqual([
      { id: 'p1', label: 'Alice', raw: { Id: 'p1', DisplayName: 'Alice' } },
      { id: 'p2', label: 'Bob', raw: { id: 'p2', label: 'Bob' } },
    ]);
  });

  test('runs NameSearch with limit and search-more friendly empty query', async () => {
    const calls: Array<{ query: string; limit: number }> = [];
    const rows = await runRelationNameSearch(
      async (query, opts) => {
        calls.push({ query, limit: opts.limit });
        return mapNameSearchRows([{ Id: 'p1', DisplayName: `Hit:${query || '*'}` }]);
      },
      '  al  ',
      5,
    );
    expect(calls).toEqual([{ query: 'al', limit: 5 }]);
    expect(rows[0]?.label).toBe('Hit:al');
  });

  test('upserts and finds selected options', () => {
    const base = [{ id: 'a', label: 'A' }];
    expect(upsertRelationOption(base, { id: 'b', label: 'B' }).map((o) => o.id)).toEqual([
      'b',
      'a',
    ]);
    expect(upsertRelationOption(base, { id: 'a', label: 'A2' })[0]?.label).toBe('A2');
    expect(findRelationOption(base, 'a')?.label).toBe('A');
    expect(findRelationOption(base, null)).toBeNull();
    // Selection not in the current page still upserts via pinned option.
    expect(
      upsertRelationOption([{ id: 'z', label: 'Z' }], { id: 'a', label: 'A' }).map((o) => o.id),
    ).toEqual(['a', 'z']);
  });
});
