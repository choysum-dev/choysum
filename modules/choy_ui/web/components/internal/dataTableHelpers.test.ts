// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clampDataTableVirtualWindow,
  compareDataTableValues,
  nextDataTableSort,
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
    expect(compareDataTableValues(null, 1)).toBeLessThan(0);
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
  });
});
