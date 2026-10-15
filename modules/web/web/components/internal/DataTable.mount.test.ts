// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ColumnDef } from '@tanstack/vue-table';
import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import DataTable from './DataTable.vue';

type Row = { Id: string; Name: string };

const columns: ColumnDef<Row, unknown>[] = [
  { accessorKey: 'Name', header: 'Name', cell: (ctx) => String(ctx.getValue() ?? '') },
];

const rows: Row[] = [
  { Id: '1', Name: 'alpha' },
  { Id: '2', Name: 'beta' },
];

describe('DataTable chrome', () => {
  test('selected rows expose data-state and keep dense header/body classes', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: true,
        rowSelection: ['1'],
      },
    });
    await flushPromises();
    const bodyRows = mounted.qa('[data-index]');
    expect(bodyRows.length).toBe(2);
    const selected = bodyRows.filter((el) => el.getAttribute('data-state') === 'selected');
    expect(selected.length).toBe(1);
    expect(selected[0]?.getAttribute('aria-selected')).toBe('true');
    expect(selected[0]?.className || '').toContain('bg-muted');
    expect(selected[0]?.className || '').toContain('min-h-list-row');
    const header = mounted.q('.choy-data-table__header');
    expect(header?.className || '').toContain('text-sm');
    expect(header?.className || '').not.toContain('text-xs');
    mounted.unmount();
  });

  test('host-controlled rowSelection paints selected chrome without built-in select column', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: false,
        rowSelection: ['2'],
      },
    });
    await flushPromises();
    const bodyRows = mounted.qa('[data-index]');
    const selected = bodyRows.filter((el) => el.getAttribute('data-state') === 'selected');
    expect(selected.length).toBe(1);
    expect(selected[0]?.getAttribute('data-index')).toBe('1');
    expect(selected[0]?.getAttribute('aria-selected')).toBe('true');
    expect(selected[0]?.className || '').toContain('bg-muted');
    mounted.unmount();
  });

  test('select column cell click toggles selection without emitting row-click', async () => {
    const onRowClick = fnRecorder();
    const onUpdateRowSelection = fnRecorder();
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: true,
        rowSelection: [],
      },
      on: {
        onRowClick,
        'onUpdate:rowSelection': onUpdateRowSelection,
      },
    });
    await flushPromises();
    const selectCell = mounted.qa('[role=gridcell]').find((el) => el.getAttribute('data-no-row-click') != null);
    expect(selectCell).toBeTruthy();
    (selectCell as HTMLElement).click();
    await flushPromises();
    expect(onUpdateRowSelection.calls.length).toBeGreaterThan(0);
    const next = onUpdateRowSelection.calls.at(-1)?.[0] as unknown[];
    expect(Array.isArray(next) && next.length > 0).toBe(true);
    expect(onRowClick.calls.length).toBe(0);
    mounted.unmount();
  });

  test('highlightColumnOnHover marks matching header and body cells', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: false,
        highlightColumnOnHover: true,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as {
      setHoveredColumn?: (id: string) => void;
      clearHoveredColumn?: () => void;
    };
    expect(typeof state.setHoveredColumn).toBe('function');
    state.setHoveredColumn!('Name');
    await flushPromises();
    const hovered = mounted.qa('[data-col-hover]');
    expect(hovered.length).toBe(3); // 1 header + 2 body cells
    expect(hovered.every((el) => (el.className || '').includes('bg-muted/40'))).toBe(true);
    state.clearHoveredColumn!();
    await flushPromises();
    expect(mounted.qa('[data-col-hover]').length).toBe(0);
    mounted.unmount();
  });

  test('highlightColumnOnHover=false ignores column hover updates', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: false,
        highlightColumnOnHover: false,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as { setHoveredColumn?: (id: string) => void };
    state.setHoveredColumn!('Name');
    await flushPromises();
    expect(mounted.qa('[data-col-hover]').length).toBe(0);
    mounted.unmount();
  });

  test('group and more rows render a single full-width group cell', async () => {
    type GroupRow = {
      kind: 'group' | 'more' | 'record';
      key: string;
      label?: string;
      count?: number;
      remain?: number;
      Name?: string;
    };
    const groupColumns: ColumnDef<GroupRow, unknown>[] = [
      {
        id: '__index__',
        header: '#',
        cell: () => '1',
        size: 40,
      },
      {
        id: '__group_label',
        header: '',
        cell: (ctx) => {
          const row = ctx.row.original;
          if (row.kind === 'group') return `${row.label} (${row.count ?? 0})`;
          if (row.kind === 'more') return `more:${row.remain ?? 0}`;
          return '';
        },
        size: 200,
      },
      {
        accessorKey: 'Name',
        header: 'Name',
        cell: (ctx) => String(ctx.getValue() ?? ''),
        size: 120,
      },
    ];
    const groupRows: GroupRow[] = [
      { kind: 'group', key: 'g1', label: 'Available', count: 3 },
      { kind: 'more', key: 'm1', remain: 2 },
      { kind: 'record', key: 'r1', Name: 'alpha' },
    ];
    const mounted = mountApp(DataTable as any, {
      props: {
        columns: groupColumns,
        data: groupRows,
        height: 200,
        virtualize: false,
        enableRowSelection: false,
        rowId: (row: GroupRow) => row.key,
      },
    });
    await flushPromises();
    const groupEl = mounted.q('[data-row-kind=group]');
    const moreEl = mounted.q('[data-row-kind=more]');
    const recordEl = mounted.q('[data-row-kind=record]');
    expect(groupEl).toBeTruthy();
    expect(moreEl).toBeTruthy();
    expect(recordEl).toBeTruthy();
    expect(groupEl!.querySelector('[data-slot=group-row]')).toBeTruthy();
    expect(moreEl!.querySelector('[data-slot=group-row]')).toBeTruthy();
    expect(recordEl!.querySelector('[data-slot=group-row]')).toBeFalsy();
    expect(groupEl!.textContent || '').toContain('Available');
    expect(groupEl!.textContent || '').toContain('(3)');
    expect(groupEl!.textContent || '').not.toContain('alpha');
    expect(moreEl!.textContent || '').toContain('more:2');
    expect(recordEl!.textContent || '').toContain('alpha');
    // Group row is one cell spanning the grid; record row keeps per-column cells.
    expect(groupEl!.querySelectorAll('[role=gridcell]').length).toBe(1);
    expect(recordEl!.querySelectorAll('[role=gridcell]').length).toBe(3);
    mounted.unmount();
  });

  test('sorted column header shows a chevron icon instead of text arrows', async () => {
    const mounted = mountApp(DataTable as any, {
      props: {
        columns,
        data: rows,
        height: 120,
        virtualize: false,
        enableRowSelection: false,
        enableSorting: true,
        sortingMode: 'client',
      },
    });
    await flushPromises();
    const state = mounted.setupState() as { onHeaderClick?: (id: string, canSort: boolean) => void };
    expect(typeof state.onHeaderClick).toBe('function');
    state.onHeaderClick!('Name', true);
    await flushPromises();
    const chevrons =
      mounted.qa('[data-lucide=ChevronUp]').length + mounted.qa('[data-lucide=ChevronDown]').length;
    expect(chevrons).toBeGreaterThan(0);
    expect(mounted.text()).not.toContain('↑');
    expect(mounted.text()).not.toContain('↓');
    mounted.unmount();
  });
});

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
        estimateSize: 40,
        enableRowSelection: false,
      },
    });
    await flushPromises();
    const body = mounted.q('.choy-data-table__body') as HTMLElement | null;
    const row = mounted.qa('[data-index]')[1] as HTMLElement | undefined;
    expect(body).toBeTruthy();
    expect(row).toBeTruthy();
    Object.defineProperty(row, 'offsetTop', { configurable: true, value: 80 });
    Object.defineProperty(row, 'offsetHeight', { configurable: true, value: 40 });
    Object.defineProperty(body, 'clientHeight', { configurable: true, value: 100 });
    mounted.root?.scrollToRow?.(1, 'start');
    expect(body!.scrollTop).toBe(80);
    mounted.root?.scrollToRow?.(1, 'center');
    expect(body!.scrollTop).toBe(50);
    mounted.root?.scrollToRow?.(1, 'end');
    expect(body!.scrollTop).toBe(20);
    mounted.root?.scrollToRow?.(99, 'start');
    mounted.setupState()?.measureRowElement?.(mounted.qa('[data-index]')[0] || null);
    mounted.setupState()?.measureRowElement?.(null);
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
