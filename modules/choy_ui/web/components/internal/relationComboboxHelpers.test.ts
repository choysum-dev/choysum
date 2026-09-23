// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  findRelationOption,
  mapNameSearchRows,
  normalizeRelationQuery,
  runRelationNameSearch,
  upsertRelationOption,
  type RelationOption,
} from './relationComboboxHelpers';

describe('relationComboboxHelpers', () => {
  test('normalizes query and maps NameSearch rows', () => {
    expect(normalizeRelationQuery('  alice  ')).toBe('alice');
    expect(
      mapNameSearchRows([
        { Id: 'p1', DisplayName: 'Alice' },
        { id: 'p2', label: 'Bob' },
        { Id: '', DisplayName: 'Skip' },
        { Id: '', id: 'from-lower', DisplayName: 'Lower' },
        { Id: { nested: true }, DisplayName: 'Object' },
        { Id: 'dup', DisplayName: 'First' },
        { Id: 'dup', DisplayName: 'Second' },
        { Id: 9, DisplayName: 'Nine' },
      ]),
    ).toEqual([
      { id: 'p1', label: 'Alice', raw: { Id: 'p1', DisplayName: 'Alice' } },
      { id: 'p2', label: 'Bob', raw: { id: 'p2', label: 'Bob' } },
      { id: 'from-lower', label: 'Lower', raw: { Id: '', id: 'from-lower', DisplayName: 'Lower' } },
      { id: 'dup', label: 'First', raw: { Id: 'dup', DisplayName: 'First' } },
      { id: '9', label: 'Nine', raw: { Id: 9, DisplayName: 'Nine' } },
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

    // Opening the combobox with no keyword must forward an empty query.
    const blank = await runRelationNameSearch(
      async (query, opts) => {
        calls.push({ query, limit: opts.limit });
        return mapNameSearchRows([{ Id: 'p2', DisplayName: `Hit:${query || '*'}` }]);
      },
      '',
      5,
    );
    expect(calls[1]).toEqual({ query: '', limit: 5 });
    expect(blank[0]?.label).toBe('Hit:*');
  });

  test('propagates search failures and normalizes non-array results', async () => {
    let rejected: unknown;
    try {
      await runRelationNameSearch(async () => {
        throw new Error('search failed');
      }, 'a');
    } catch (err) {
      rejected = err;
    }
    expect(rejected instanceof Error && rejected.message).toBe('search failed');

    const empty = await runRelationNameSearch(async () => null as unknown as RelationOption[], 'a');
    expect(empty).toEqual([]);

    const limited = await runRelationNameSearch(
      async () => [
        { id: '1', label: 'One' },
        { id: '2', label: 'Two' },
      ],
      'x',
      Number.NaN,
    );
    expect(limited).toHaveLength(2);
  });

  test('upserts and finds selected options', () => {
    const base = [{ id: 'a', label: 'A' }];
    expect(upsertRelationOption(base, null)).toEqual(base);
    expect(upsertRelationOption(base, { id: '', label: 'Empty' })).toEqual(base);
    expect(upsertRelationOption(base, { id: 'b', label: 'B' }).map((o) => o.id)).toEqual([
      'b',
      'a',
    ]);
    expect(upsertRelationOption(base, { id: 'a', label: 'A2' })[0]?.label).toBe('A2');
    expect(findRelationOption(base, 'a')?.label).toBe('A');
    expect(findRelationOption(base, null)).toBeNull();
    expect(findRelationOption(base, '   ')).toBeNull();
    // Selection not in the current page still upserts via pinned option.
    expect(
      upsertRelationOption([{ id: 'z', label: 'Z' }], { id: 'a', label: 'A' }).map((o) => o.id),
    ).toEqual(['a', 'z']);
    // Blank DisplayName falls back to id.
    expect(mapNameSearchRows([{ Id: 'p3', DisplayName: '   ' }])).toEqual([
      { id: 'p3', label: 'p3', raw: { Id: 'p3', DisplayName: '   ' } },
    ]);
  });
});
