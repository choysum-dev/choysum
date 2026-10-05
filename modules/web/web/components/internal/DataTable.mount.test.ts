// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ColumnDef } from '@tanstack/vue-table';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import DataTable from './DataTable.vue';

type Row = { Id: string; Name: string };

const columns: ColumnDef<Row, unknown>[] = [
  { accessorKey: 'Name', header: 'Name', cell: (ctx) => String(ctx.getValue() ?? '') },
];

const rows: Row[] = [
  { Id: '1', Name: 'alpha' },
  { Id: '2', Name: 'beta' },
];

describe('DataTable virtualize', () => {
  test('virtualize=false renders every row in document flow', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: false,
      },
    });
    await flushPromises();
    const bodyRows = mounted.qa('[data-index]');
    expect(bodyRows.length).toBe(2);
    expect((bodyRows[0]?.getAttribute('class') || '').includes('absolute')).toBe(false);
    expect(mounted.text().includes('alpha')).toBe(true);
    expect(mounted.text().includes('beta')).toBe(true);
    mounted.unmount();
  });
});
