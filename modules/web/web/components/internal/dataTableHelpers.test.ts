// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyDataTableParentScroll,
  applyDataTableScrollToRow,
  clampDataTableVirtualWindow,
  compareDataTableValues,
  dataTableIsVirtualized,
  dataTableMaybeMeasure,
  dataTableMeasureRow,
  dataTableScrollElement,
  dataTableSelectionIdsEqual,
  dataTableShouldMeasureRow,
  dataTableVirtualizerCount,
  decodeDataTableRowKey,
  encodeDataTableRowKey,
  mapDataTableBodyRows,
  mapDataTableSelectionKeys,
  mergeDataTableControlledSelection,
  nextDataTableSort,
  nextServerDataTableSort,
  normalizeDataTableRowId,
  pruneDataTableSelection,
  resolveDataTableEstimateSize,
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

  test('server sort emit cycles asc → desc → clear', () => {
    expect(nextServerDataTableSort(null, 'name')).toEqual({
      field: 'name',
      sorting: [{ id: 'name', desc: false }],
      direction: 'asc',
    });
    expect(nextServerDataTableSort({ id: 'name', desc: false }, 'name')).toEqual({
      field: 'name',
      sorting: [{ id: 'name', desc: true }],
      direction: 'desc',
    });
    expect(nextServerDataTableSort({ id: 'name', desc: true }, 'name')).toEqual({
      field: 'name',
      sorting: [],
      direction: undefined,
    });
    expect(nextServerDataTableSort({ id: 'name', desc: false }, 'age')).toEqual({
      field: 'age',
      sorting: [{ id: 'age', desc: false }],
      direction: 'asc',
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
    // Equal keys must keep input order (`Array.prototype.sort` stability under QuickJS).
    const ties = [
      { name: 'a', seq: 1 },
      { name: 'a', seq: 2 },
      { name: 'a', seq: 3 },
    ];
    expect(sortDataTableRows(ties, (r) => r.name, 'asc').map((r) => r.seq)).toEqual([1, 2, 3]);
    expect(sortDataTableRows(ties, (r) => r.name, 'desc').map((r) => r.seq)).toEqual([1, 2, 3]);
  });

  test('compares nulls and numbers', () => {
    expect(compareDataTableValues(null, null)).toBe(0);
    expect(compareDataTableValues(null, 1)).toBeLessThan(0);
    expect(compareDataTableValues(1, null)).toBeGreaterThan(0);
    expect(compareDataTableValues(2, 1)).toBeGreaterThan(0);
    expect(compareDataTableValues(Number.NaN, 1)).toBeGreaterThan(0);
    expect(compareDataTableValues(1, Number.NaN)).toBeLessThan(0);
    expect(compareDataTableValues(Number.NaN, Number.NaN)).toBe(0);
    const earlier = new Date('2026-01-01T00:00:00Z');
    const later = new Date('2026-12-31T00:00:00Z');
    expect(compareDataTableValues(earlier, later)).toBeLessThan(0);
    expect(compareDataTableValues(later, earlier)).toBeGreaterThan(0);
    expect(compareDataTableValues(new Date(Number.NaN), later)).toBeGreaterThan(0);
    // Mixed Date/number columns must compare on epoch millis, not localeCompare.
    expect(compareDataTableValues(earlier, earlier.getTime())).toBe(0);
    expect(compareDataTableValues(later.getTime(), earlier)).toBeGreaterThan(0);
    // ±Infinity must order via < / >, not subtraction (Infinity - Infinity is NaN).
    expect(compareDataTableValues(Number.POSITIVE_INFINITY, 1)).toBeGreaterThan(0);
    expect(compareDataTableValues(Number.NEGATIVE_INFINITY, 1)).toBeLessThan(0);
    expect(compareDataTableValues(Number.POSITIVE_INFINITY, Number.POSITIVE_INFINITY)).toBe(0);
    // Numeric-looking strings must sort numerically even without ICU localeCompare options.
    expect(compareDataTableValues('9', '10')).toBeLessThan(0);
    expect(compareDataTableValues('10', '9')).toBeGreaterThan(0);
    expect(compareDataTableValues('9', '9')).toBe(0);
    // Distinct snowflake-style ids beyond MAX_SAFE_INTEGER must not collapse as equal.
    expect(
      compareDataTableValues('9007199254740992', '9007199254740993'),
    ).not.toBe(0);
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
    // Reversed / out-of-range windows must collapse instead of inverting or overflowing.
    expect(clampDataTableVirtualWindow(5, 2, 10)).toEqual({ start: 5, end: 5 });
    expect(clampDataTableVirtualWindow(7, 9, 3)).toEqual({ start: 3, end: 3 });
  });

  test('encodeDataTableRowKey keeps numeric and string ids distinct', () => {
    expect(encodeDataTableRowKey(42)).toBe('n:42');
    expect(encodeDataTableRowKey('42')).toBe('s:42');
    expect(encodeDataTableRowKey(42)).not.toBe(encodeDataTableRowKey('42'));
    // Padded strings must match the trimmed resolveDataTableRowId key.
    expect(encodeDataTableRowKey(' 7 ')).toBe(encodeDataTableRowKey('7'));
    expect(encodeDataTableRowKey('')).toBe('s:');
    expect(encodeDataTableRowKey(Number.NaN)).toBe('s:NaN');
    expect(decodeDataTableRowKey(encodeDataTableRowKey(Number.NaN))).toBe('NaN');
    // Malformed numeric keys must not decode to 0 via Number('').
    expect(decodeDataTableRowKey('n:')).toBe('n:');
    expect(decodeDataTableRowKey('n: ')).toBe('n: ');
  });

  test('round-trips typed row ids through encode/decode', () => {
    for (const [raw, expected] of [
      [0, 0],
      [42, 42],
      [-1.5, -1.5],
      ['42', '42'],
      [' a ', 'a'],
      ['0', '0'],
    ] as const) {
      expect(decodeDataTableRowKey(encodeDataTableRowKey(raw))).toEqual(expected);
    }
  });

  test('resolveDataTableRowId prefers Id/id and rejects empty keys', () => {
    expect(resolveDataTableRowId({ Id: 42 })).toBe(42);
    expect(resolveDataTableRowId({ id: 'x' })).toBe('x');
    expect(resolveDataTableRowId({ Id: ' 7 ' })).toBe('7');
    // Blank Id must not block a usable lowercase id.
    expect(resolveDataTableRowId({ Id: '', id: 'fallback' })).toBe('fallback');
    expect(resolveDataTableRowId({ name: 'a' }, (r) => String(r.name))).toBe('a');
    expect(resolveDataTableRowId({ name: 'a' }, () => '  z  ')).toBe('z');
    expect(() => resolveDataTableRowId({ name: 'a' })).toThrow(/rowId/);
    expect(() => resolveDataTableRowId({ Id: '' })).toThrow(/rowId/);
    expect(() => resolveDataTableRowId({ Id: Number.NaN })).toThrow(/rowId/);
    expect(() => resolveDataTableRowId({ name: 'a' }, () => '')).toThrow(/empty id/);
    expect(() => resolveDataTableRowId({ name: 'a' }, () => Number.NaN)).toThrow(/empty id/);
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
    // Missing registry entries decode the internal prefix instead of leaking `n:`/`s:`.
    expect(mapDataTableSelectionKeys(['n:7', 's:7', 'plain'], new Map())).toEqual([7, '7', 'plain']);
  });

  test('pruneDataTableSelection drops missing and deselected keys', () => {
    const present = new Set(['a', 'c']);
    expect(pruneDataTableSelection({ a: true, b: true, c: false }, present)).toEqual({ a: true });
    expect(pruneDataTableSelection({ a: true }, present)).toBeNull();
    expect(pruneDataTableSelection({}, present)).toBeNull();
    // All-false leftovers must not force a no-op `{}` write.
    expect(pruneDataTableSelection({ a: false, b: false }, present)).toBeNull();
  });

  test('dataTableSelectionIdsEqual compares unique encoded ids', () => {
    expect(dataTableSelectionIdsEqual([1, 1], [1])).toBe(true);
    expect(dataTableSelectionIdsEqual([1, 1], [1, 2])).toBe(false);
    expect(dataTableSelectionIdsEqual([42, '42'], [42, '42'])).toBe(true);
    expect(dataTableSelectionIdsEqual([42], ['42'])).toBe(false);
    expect(dataTableSelectionIdsEqual([' 7 '], ['7'])).toBe(true);
    // Unusable ids must not equate to encoded string ghosts like `s:NaN`.
    expect(dataTableSelectionIdsEqual([Number.NaN], ['NaN'])).toBe(false);
  });

  test('normalizeDataTableRowId trims strings and rejects unusable ids', () => {
    expect(normalizeDataTableRowId(42)).toBe(42);
    expect(normalizeDataTableRowId(' 7 ')).toBe('7');
    expect(normalizeDataTableRowId('   ')).toBeNull();
    expect(normalizeDataTableRowId(Number.NaN)).toBeNull();
    expect(normalizeDataTableRowId(Number.POSITIVE_INFINITY)).toBeNull();
    expect(encodeDataTableRowKey(normalizeDataTableRowId(' a ') as string)).toBe('s:a');
  });

  test('mergeDataTableControlledSelection keeps off-page ids', () => {
    const present = new Set([encodeDataTableRowKey(2), encodeDataTableRowKey(3)]);
    expect(mergeDataTableControlledSelection([1, 2], [2, 3], present)).toEqual([1, 2, 3]);
    expect(mergeDataTableControlledSelection([1], [], present)).toEqual([1]);
    // Visible ids already kept off-page must not be duplicated.
    expect(mergeDataTableControlledSelection([1], [1, 2], present)).toEqual([1, 2]);
    // Duplicate off-page controlled ids must collapse to one entry.
    expect(mergeDataTableControlledSelection([1, 1], [2], present)).toEqual([1, 2]);
    // Blank / non-finite controlled ids must not survive as ghost off-page entries.
    expect(mergeDataTableControlledSelection(['', Number.NaN, '  ', 1], [2], present)).toEqual([
      1, 2,
    ]);
    // Padded off-page ids must emit in normalized form.
    expect(mergeDataTableControlledSelection([' 1 '], [2], present)).toEqual(['1', 2]);
    // Invalid visible ids are dropped the same way as controlled ghosts.
    expect(mergeDataTableControlledSelection([1], ['', Number.NaN, 2], present)).toEqual([1, 2]);
  });

  test('resolveDataTableEstimateSize prefers positive sizes', () => {
    expect(resolveDataTableEstimateSize(40)).toBe(40);
    expect(resolveDataTableEstimateSize(0)).toBe(32);
    expect(resolveDataTableEstimateSize(Number.NaN)).toBe(32);
    expect(resolveDataTableEstimateSize(undefined, 28)).toBe(28);
    expect(resolveDataTableEstimateSize(-1, 28)).toBe(28);
  });

  test('dataTable virtualization helpers cover on and off paths', () => {
    expect(dataTableIsVirtualized(undefined, 0)).toBe(false);
    expect(dataTableIsVirtualized(undefined, 2)).toBe(true);
    expect(dataTableIsVirtualized(true, 2)).toBe(true);
    expect(dataTableIsVirtualized(false, 2)).toBe(false);
    expect(dataTableVirtualizerCount(true, 5)).toBe(5);
    expect(dataTableVirtualizerCount(false, 5)).toBe(0);
    const parent = {} as Element;
    expect(dataTableScrollElement(true, parent)).toBe(parent);
    expect(dataTableScrollElement(false, parent)).toBeNull();
    expect(dataTableShouldMeasureRow(parent, true)).toBe(true);
    expect(dataTableShouldMeasureRow(null, true)).toBe(false);
    expect(dataTableShouldMeasureRow(parent, false)).toBe(false);
    let measured = 0;
    dataTableMaybeMeasure(true, { measure: () => { measured += 1; } });
    dataTableMaybeMeasure(false, { measure: () => { measured += 1; } });
    dataTableMaybeMeasure(true, null);
    expect(measured).toBe(1);
    let rowMeasured = 0;
    dataTableMeasureRow(parent, true, { measureElement: () => { rowMeasured += 1; } });
    dataTableMeasureRow(null, true, { measureElement: () => { rowMeasured += 1; } });
    dataTableMeasureRow(parent, false, { measureElement: () => { rowMeasured += 1; } });
    expect(rowMeasured).toBe(1);
  });

  test('mapDataTableBodyRows prefers virtual items when present', () => {
    const rows = [{ id: 'a' }, { id: 'b' }];
    expect(
      mapDataTableBodyRows({
        virtualized: false,
        rows,
        virtualItems: [{ index: 1, key: 'x', start: 10 }],
      }).map((item) => item.key),
    ).toEqual(['a', 'b']);
    expect(
      mapDataTableBodyRows({
        virtualized: true,
        rows,
        virtualItems: [],
      }).map((item) => item.key),
    ).toEqual(['a', 'b']);
    const mapped = mapDataTableBodyRows({
      virtualized: true,
      rows,
      virtualItems: [{ index: 1, key: 'x', start: 40 }],
    });
    expect(mapped).toEqual([{ key: 'b', index: 1, start: 40, row: rows[1] }]);
    const missing = mapDataTableBodyRows({
      virtualized: true,
      rows,
      virtualItems: [{ index: 9, key: 'ghost', start: 0 }],
    });
    expect(missing[0]?.key).toBe('ghost');
  });

  test('applyDataTableScrollToRow covers virtual, measured, and estimate paths', () => {
    const virtual: Array<[number, string | undefined]> = [];
    const tops: number[] = [];
    const virtualizer = {
      scrollToIndex: (i: number, opts?: { align?: 'start' | 'center' | 'end' | 'auto' }) => {
        virtual.push([i, opts?.align]);
      },
    };
    applyDataTableScrollToRow({
      index: Number.NaN,
      virtualized: true,
      parent: null,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: -1,
      virtualized: true,
      parent: null,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 3,
      align: 'end',
      virtualized: true,
      parent: null,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 1,
      virtualized: true,
      parent: null,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 2,
      virtualized: true,
      parent: null,
      estimateSize: 32,
      virtualizer: null,
      setScrollTop: (top) => tops.push(top),
    });
    expect(virtual).toEqual([
      [3, 'end'],
      [1, 'auto'],
    ]);
    applyDataTableScrollToRow({
      index: 1,
      virtualized: false,
      parent: null,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    const row = { offsetTop: 80, offsetHeight: 40 };
    const parent = {
      clientHeight: 100,
      querySelector: (sel: string) => (sel === '[data-index="1"]' ? row : null),
    };
    applyDataTableScrollToRow({
      index: 1,
      align: 'start',
      virtualized: false,
      parent,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 1,
      align: 'center',
      virtualized: false,
      parent,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 1,
      align: 'end',
      virtualized: false,
      parent,
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    applyDataTableScrollToRow({
      index: 4,
      virtualized: false,
      parent: {
        clientHeight: 100,
        querySelector: () => null,
      },
      estimateSize: 32,
      virtualizer,
      setScrollTop: (top) => tops.push(top),
    });
    expect(tops).toEqual([80, 50, 20, 128]);
    applyDataTableParentScroll(null, 1);
    const scroller = { scrollTop: 0 };
    applyDataTableParentScroll(scroller, 9);
    expect(scroller.scrollTop).toBe(9);
  });
});
