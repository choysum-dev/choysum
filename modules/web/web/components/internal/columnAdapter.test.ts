// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { columnsToColumnDefs } from './columnAdapter';
import type { Column } from '@/web/web/composables/useVTable';

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
    expect(defs[0].id).toBe('Name');
    expect(defs[0].size).toBe(120);
    expect(defs[0].enableSorting).toBe(true);
    expect(defs[1].enableSorting).toBe(false);
    expect(defs[1].size).toBe(50);
  });
});
