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

  test('non-virtualized scrollToRow uses measured row offsets', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        estimateSize: 32,
        enableRowSelection: false,
      },
    });
    await flushPromises();
    const body = mounted.q('.choy-data-table__body') as HTMLElement | null;
    expect(body).toBeTruthy();
    mounted.root?.scrollToRow?.(1, 'start');
    mounted.root?.scrollToRow?.(1, 'center');
    mounted.root?.scrollToRow?.(1, 'end');
    mounted.root?.scrollToRow?.(99, 'start');
    expect(mounted.qa('[data-index]').length).toBe(2);
    mounted.unmount();
  });

  test('empty non-virtual table shows rows after data arrives', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: [],
        height: 120,
        virtualize: false,
        enableRowSelection: false,
      },
      reactiveProps: true,
    });
    await flushPromises();
    expect(mounted.qa('[data-index]').length).toBe(0);
    mounted.props.data = rows;
    mounted.props.estimateSize = 40;
    await flushPromises();
    expect(mounted.qa('[data-index]').length).toBe(2);
    expect(mounted.text().includes('alpha')).toBe(true);
    mounted.root?.scrollToRow?.(-1);
    mounted.root?.scrollToRow?.(Number.NaN);
    mounted.unmount();
  });
});
