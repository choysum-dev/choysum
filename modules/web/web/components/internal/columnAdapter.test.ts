// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  columnsToColumnDefs,
  FALLBACK_COLUMN_SIZE,
  listColumnIsResizable,
  orderListTableColumns,
  resolveColumnSizePx,
  resolveColumnWidthSpec,
  resolveIndexDisplayMaxText,
} from './columnAdapter';
import { LIST_INDEX_COLUMN_MIN_WIDTH } from '@/web/web/components/field/listColumnSizeDefaults';
import { setTableColumnMeta, type Column } from '@/web/web/composables/useTable';

describe('columnsToColumnDefs', () => {
  test('maps width, sortable, and cell renderer', () => {
    const cols: Column[] = [
      {
        key: 'Name',
        dataKey: 'Name',
        title: 'Name',
        width: 120,
        sortable: true,
        cellRenderer: ({ rowData }) => String((rowData as any).Name ?? ''),
      },
      {
        key: '__index__',
        dataKey: '__index__',
        title: '#',
        width: 50,
        sortable: false,
      },
    ];
    const defs = columnsToColumnDefs(cols);
    expect(defs).toHaveLength(2);
    // Index is ordered before field columns.
    expect(defs[0].id).toBe('__index__');
    expect(defs[0].enableSorting).toBe(false);
    expect(defs[0].size).toBe(50);
    expect(defs[1].id).toBe('Name');
    expect(defs[1].size).toBe(120);
    expect(defs[1].enableSorting).toBe(true);
  });

  test('orderListTableColumns puts selection, handle, index, group label first', () => {
    const cols: Column[] = [
      { key: 'Name', dataKey: 'Name', title: 'Name' },
      { key: '__group_label', title: '' },
      { key: '__index__', title: '#' },
      { key: '__handle__', title: '' },
      { key: '__selection__', title: '' },
    ];
    const ordered = orderListTableColumns(cols).map((c) => String(c.key));
    expect(ordered).toEqual(['__selection__', '__handle__', '__index__', '__group_label', 'Name']);
  });

  test('selection column type marks meta.selectColumn even with a custom key', () => {
    const cols: Column[] = [
      { key: 'pick', title: '', type: 'selection' },
      { key: 'Name', dataKey: 'Name', title: 'Name' },
    ];
    const ordered = orderListTableColumns(cols).map((c) => String(c.key));
    expect(ordered[0]).toBe('pick');
    const defs = columnsToColumnDefs(cols);
    expect((defs[0].meta as { selectColumn?: boolean } | undefined)?.selectColumn).toBe(true);
  });

  test('minWidth alone is the track size (type defaults must not be raised to FALLBACK)', () => {
    expect(FALLBACK_COLUMN_SIZE).toBe(140);
    expect(resolveColumnSizePx({ key: 'Active', minWidth: 80 }).size).toBe(80);
    expect(resolveColumnSizePx({ key: 'State', minWidth: 100 }).size).toBe(100);
    const wide = resolveColumnSizePx({ key: 'Value', minWidth: 220 });
    expect(wide.minSize).toBe(220);
    expect(wide.size).toBe(220);
    const defs = columnsToColumnDefs([{ key: 'Value', dataKey: 'Value', minWidth: 220 }]);
    expect(defs[0].size).toBe(220);
    expect(defs[0].minSize).toBe(220);
  });

  test('width and minWidth take the max', () => {
    const { size } = resolveColumnSizePx({ key: 'A', width: 100, minWidth: 180 });
    expect(size).toBe(180);
    const defs = columnsToColumnDefs([{ key: 'A', width: 100, minWidth: 180 }]);
    expect(defs[0].size).toBe(180);
  });

  test('neither width nor minWidth uses FALLBACK', () => {
    const defs = columnsToColumnDefs([{ key: 'Name', dataKey: 'Name' }]);
    expect(defs[0].size).toBe(FALLBACK_COLUMN_SIZE);
  });

  test('explicit width below FALLBACK stays when no minWidth', () => {
    const defs = columnsToColumnDefs([{ key: 'idx', width: 44, type: 'index' }]);
    expect(defs[0].size).toBe(44);
  });

  test('soft fieldType gets capped widthSpec; hard and system stay fixed', () => {
    const name: Column = { key: 'Name', dataKey: 'Name', minWidth: 140 };
    const qty: Column = { key: 'Qty', dataKey: 'Qty', minWidth: 80 };
    const idx: Column = { key: '__index__', type: 'index', width: 44 };
    setTableColumnMeta(name, { fieldType: 'varchar' });
    setTableColumnMeta(qty, { fieldType: 'int' });

    expect(resolveColumnWidthSpec(name, 140)).toEqual({ type: 'capped', max: 350 });
    expect(resolveColumnWidthSpec(qty, 80)).toBeUndefined();
    expect(resolveColumnWidthSpec(idx, 44)).toBeUndefined();

    const defs = columnsToColumnDefs([idx, name, qty]);
    expect((defs[0].meta as { widthSpec?: unknown } | undefined)?.widthSpec).toBeUndefined();
    expect((defs[1].meta as { widthSpec?: { type: string; max: number } } | undefined)?.widthSpec).toEqual({
      type: 'capped',
      max: 350,
    });
    expect(defs[1].maxSize).toBe(350);
    expect((defs[2].meta as { widthSpec?: unknown } | undefined)?.widthSpec).toBeUndefined();
  });

  test('Code leaf is semi-hard capped at 160; Title stays soft 350', () => {
    const code: Column = { key: 'Code', dataKey: 'Code', minWidth: 100 };
    const title: Column = { key: 'Title', dataKey: 'Title', minWidth: 140 };
    setTableColumnMeta(code, { fieldType: 'varchar' });
    setTableColumnMeta(title, { fieldType: 'varchar' });
    expect(resolveColumnWidthSpec(code, 100)).toEqual({ type: 'capped', max: 160 });
    expect(resolveColumnWidthSpec(title, 140)).toEqual({ type: 'capped', max: 350 });
  });

  test('growth cap uses author min, not post-sample size', () => {
    const title: Column = { key: 'Title', dataKey: 'Title', minWidth: 140 };
    setTableColumnMeta(title, { fieldType: 'text' });
    // Sample already raised size to the original cap (140*3 → 420); do not recompute to 560.
    expect(resolveColumnWidthSpec(title, 420)).toBeUndefined();
    expect(resolveColumnWidthSpec(title, 300)).toEqual({ type: 'capped', max: 420 });
  });

  test('explicit minWidth 180 on Code is not crushed below 180', () => {
    const code: Column = { key: 'Code', dataKey: 'Code', minWidth: 180 };
    setTableColumnMeta(code, { fieldType: 'varchar' });
    const { size } = resolveColumnSizePx(code);
    expect(size).toBe(180);
    // max <= size → fixed track (no cap below author min).
    expect(resolveColumnWidthSpec(code, 180)).toBeUndefined();
  });

  test('hard column header budget raises size for long titles', () => {
    const col: Column = {
      key: 'State',
      dataKey: 'State',
      minWidth: 80,
      title: '非常长的状态标签名',
    };
    setTableColumnMeta(col, { fieldType: 'boolean' });
    const { size } = resolveColumnSizePx(col, { textMeasure: null });
    expect(size).toBeGreaterThan(80);
    expect(size % 20).toBe(0);
  });

  test('soft sample rows raise min up to type max', () => {
    const title: Column = { key: 'Title', dataKey: 'Title', title: '标题', minWidth: 140 };
    setTableColumnMeta(title, { fieldType: 'varchar' });
    const measure = (text: string) => String(text).length * 10;
    const defs = columnsToColumnDefs([title], {
      sampleRows: [{ Title: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXX' }],
      textMeasure: measure,
    });
    expect(defs[0].size).toBeGreaterThan(140);
    expect(defs[0].size).toBeLessThanOrEqual(350);
    expect(defs[0].minSize).toBe(defs[0].size);
  });

  test('soft sample does not exceed semi-hard Code max 160', () => {
    const code: Column = { key: 'Code', dataKey: 'Code', title: '编码', minWidth: 100 };
    setTableColumnMeta(code, { fieldType: 'varchar' });
    const defs = columnsToColumnDefs([code], {
      sampleRows: [{ Code: 'SKU-VERY-LONG-CODE-VALUE-99999' }],
      textMeasure: (text) => String(text).length * 10,
    });
    expect(defs[0].size).toBeLessThanOrEqual(160);
    expect(defs[0].size).toBeGreaterThanOrEqual(100);
  });

  test('userWidths force fixed tracks and beat soft sample', () => {
    const title: Column = { key: 'Title', dataKey: 'Title', title: '标题', minWidth: 140 };
    setTableColumnMeta(title, { fieldType: 'varchar' });
    const defs = columnsToColumnDefs([title], {
      sampleRows: [{ Title: 'short' }],
      userWidths: { Title: 240 },
      textMeasure: null,
    });
    expect(defs[0].size).toBe(240);
    expect(defs[0].minSize).toBe(240);
    expect((defs[0].meta as { widthSpec?: unknown; userSized?: boolean; resizable?: boolean }).widthSpec).toBeUndefined();
    expect((defs[0].meta as { userSized?: boolean }).userSized).toBe(true);
    expect((defs[0].meta as { resizable?: boolean }).resizable).toBe(true);
  });

  test('author explicit width blocks userWidths and is not resizable', () => {
    const title: Column = { key: 'Title', dataKey: 'Title', width: 180, minWidth: 140 };
    setTableColumnMeta(title, { fieldType: 'varchar' });
    const defs = columnsToColumnDefs([title], { userWidths: { Title: 300 } });
    expect(defs[0].size).toBe(180);
    expect((defs[0].meta as { resizable?: boolean }).resizable).toBe(false);
  });

  test('listColumnIsResizable rejects narrow chrome without dataKey', () => {
    expect(listColumnIsResizable({ key: 'del', minWidth: 36 })).toBe(false);
    expect(listColumnIsResizable({ key: 'Name', dataKey: 'Name', minWidth: 140 })).toBe(true);
    expect(listColumnIsResizable({ key: 'actions', dataKey: 'x', minWidth: 140 })).toBe(false);
  });

  test('explicit numeric width skips soft cap even for varchar', () => {
    const col: Column = { key: 'Code', dataKey: 'Code', width: 100, minWidth: 100 };
    setTableColumnMeta(col, { fieldType: 'varchar' });
    expect(resolveColumnWidthSpec(col, 100)).toBeUndefined();
  });

  test('author flex/auto wins over soft cap', () => {
    const col: Column = { key: 'Title', dataKey: 'Title', minWidth: 140 };
    setTableColumnMeta(col, { fieldType: 'varchar', widthSpec: { type: 'flex', weight: 2 } });
    expect(resolveColumnWidthSpec(col, 140)).toEqual({ type: 'flex', weight: 2 });
  });

  test('forwards author widthSpec from column meta into ColumnDef.meta', () => {
    const col: Column = { key: 'Desc', dataKey: 'Desc', minWidth: 140 };
    setTableColumnMeta(col, { widthSpec: { type: 'auto' } });
    const defs = columnsToColumnDefs([col]);
    expect((defs[0].meta as { widthSpec?: { type: string } } | undefined)?.widthSpec).toEqual({
      type: 'auto',
    });
  });

  test('index digit budget raises from current-page max display text', () => {
    const idx: Column = { key: '__index__', type: 'index', width: LIST_INDEX_COLUMN_MIN_WIDTH };
    expect(listColumnIsResizable(idx)).toBe(false);

    const narrow = resolveColumnSizePx(idx, {
      indexDisplayMaxText: '20',
      textMeasure: null,
    });
    expect(narrow.size).toBe(44);

    const wide = resolveColumnSizePx(idx, {
      indexDisplayMaxText: '10000',
      textMeasure: null,
    });
    expect(wide.size).toBeGreaterThan(44);
    expect(wide.size).toBeLessThanOrEqual(80);

    expect(
      resolveIndexDisplayMaxText({
        baseIndex: 981,
        sampleRows: [{ Id: 'a' }, { Id: 'b' }],
      }),
    ).toBe('982');

    const defs = columnsToColumnDefs([idx], {
      baseIndex: 9990,
      sampleRows: Array.from({ length: 10 }, (_, i) => ({ Id: String(i) })),
      textMeasure: null,
    });
    expect(defs[0].size).toBeGreaterThan(44);
    expect((defs[0].meta as { resizable?: boolean }).resizable).toBe(false);

    // userWidths must not resize index chrome.
    const blocked = columnsToColumnDefs([idx], {
      indexDisplayMaxText: '9',
      userWidths: { __index__: 120 },
      textMeasure: null,
    });
    expect(blocked[0].size).toBe(44);

    // Per-row groupIndex (mirrors cell renderer) widens when larger than base+i.
    const grouped = columnsToColumnDefs([idx], {
      baseIndex: 1,
      sampleRows: [
        { Id: 'a', groupIndex: 2 },
        { Id: 'b', groupIndex: 12345 },
      ],
      textMeasure: null,
    });
    expect(grouped[0].size).toBeGreaterThan(44);
    expect(grouped[0].size).toBeLessThanOrEqual(80);
    expect(
      resolveIndexDisplayMaxText({
        baseIndex: 100,
        sampleRows: [{ Id: 'x' }, { Id: 'y', groupIndex: 3 }],
      }),
    ).toBe('100'); // row0 → 100, row1 groupIndex 3 → max stays 100

    // Author wider floor wins over a short digit budget.
    const custom: Column = { key: '__index__', type: 'index', width: 72 };
    const floored = resolveColumnSizePx(custom, {
      indexDisplayMaxText: '9',
      textMeasure: null,
    });
    expect(floored.size).toBe(72);
  });
});

