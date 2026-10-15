// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { columnsToColumnDefs, orderListTableColumns } from './columnAdapter';
import type { Column } from '@/web/web/composables/useTable';

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
});
