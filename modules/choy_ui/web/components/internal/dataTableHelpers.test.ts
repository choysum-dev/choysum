// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clampDataTableVirtualWindow,
  compareDataTableValues,
  encodeDataTableRowKey,
  mapDataTableSelectionKeys,
  nextDataTableSort,
  resolveDataTableRowId,
  setDataTableSelectionAll,
  sortDataTableRows,
  toggleDataTableSelection,
} from './dataTableHelpers';

describe('dataTableHelpers', () => {
  test('cycles sort none → asc → desc → none', () => {
    expect(nextDataTableSort(null, 'name')).toEqual({ id: 'name', desc: false });
    expect(nextDataTableSort({ id: 'name', desc: false }, 'name')).toEqual({
      id: 'name',
      desc: true,
    });
    expect(nextDataTableSort({ id: 'name', desc: true }, 'name')).toBeNull();
    expect(nextDataTableSort({ id: 'name', desc: false }, 'age')).toEqual({
      id: 'age',
      desc: false,
    });
  });

  test('sorts rows by accessor', () => {
    const rows = [{ name: 'b' }, { name: 'a' }, { name: 'c' }];
    expect(sortDataTableRows(rows, (r) => r.name, 'asc').map((r) => r.name)).toEqual([
      'a',
      'b',
      'c',
    ]);
    expect(sortDataTableRows(rows, (r) => r.name, 'desc').map((r) => r.name)).toEqual([
      'c',
      'b',
      'a',
    ]);
  });

  test('compares nulls and numbers', () => {
    expect(compareDataTableValues(null, null)).toBe(0);
    expect(compareDataTableValues(null, 1)).toBeLessThan(0);
    expect(compareDataTableValues(1, null)).toBeGreaterThan(0);
    expect(compareDataTableValues(2, 1)).toBeGreaterThan(0);
  });

  test('toggles and bulk-sets selection', () => {
    const one = toggleDataTableSelection(new Set(), 'a');
    expect([...one]).toEqual(['a']);
    expect([...toggleDataTableSelection(one, 'a')]).toEqual([]);
    expect([...setDataTableSelectionAll(['a', 'b'], true)].sort()).toEqual(['a', 'b']);
    expect([...setDataTableSelectionAll(['a', 'b'], false)]).toEqual([]);
  });

  test('clamps virtual windows', () => {
    expect(clampDataTableVirtualWindow(-2, 50, 10)).toEqual({ start: 0, end: 10 });
    expect(clampDataTableVirtualWindow(3, 7, 10)).toEqual({ start: 3, end: 7 });
    expect(clampDataTableVirtualWindow(Number.NaN, Number.NaN, Number.NaN)).toEqual({
      start: 0,
      end: 0,
    });
  });

  test('encodeDataTableRowKey keeps numeric and string ids distinct', () => {
    expect(encodeDataTableRowKey(42)).toBe('n:42');
    expect(encodeDataTableRowKey('42')).toBe('s:42');
    expect(encodeDataTableRowKey(42)).not.toBe(encodeDataTableRowKey('42'));
  });

  test('resolveDataTableRowId prefers Id/id and rejects empty keys', () => {
    expect(resolveDataTableRowId({ Id: 42 })).toBe(42);
    expect(resolveDataTableRowId({ id: 'x' })).toBe('x');
    // Blank Id must not block a usable lowercase id.
    expect(resolveDataTableRowId({ Id: '', id: 'fallback' })).toBe('fallback');
    expect(resolveDataTableRowId({ name: 'a' }, (r) => String(r.name))).toBe('a');
    expect(() => resolveDataTableRowId({ name: 'a' })).toThrow(/rowId/);
    expect(() => resolveDataTableRowId({ Id: '' })).toThrow(/rowId/);
    expect(() => resolveDataTableRowId({ name: 'a' }, () => '')).toThrow(/empty id/);
    expect(() => resolveDataTableRowId({ name: 'a' }, () => ({}) as unknown as string)).toThrow(
      /empty id/,
    );
    expect(() => resolveDataTableRowId({ Id: { nested: true } as unknown as string })).toThrow(
      /rowId/,
    );
  });

  test('mapDataTableSelectionKeys restores original id types', () => {
    const registry = new Map<string, string | number>([
      [encodeDataTableRowKey(42), 42],
      [encodeDataTableRowKey('42'), '42'],
      [encodeDataTableRowKey('a'), 'a'],
    ]);
    expect(
      mapDataTableSelectionKeys(
        [encodeDataTableRowKey(42), encodeDataTableRowKey('42'), encodeDataTableRowKey('a')],
        registry,
      ),
    ).toEqual([42, '42', 'a']);
    expect(mapDataTableSelectionKeys(['missing'], registry)).toEqual(['missing']);
  });
});
