// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, ref } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import {
  LIST_COLUMN_WIDTH_WRITE_DEBOUNCE_MS,
  readListColumnWidthPrefs,
  writeListColumnWidthPrefs,
  type ListColumnWidthBucket,
} from '@/web/web/composables/listColumnWidthPrefs';
import ChoyTableHost from './ChoyTableHost.vue';
import DataTable from './DataTable.vue';

function readBaseIndex(ss: any): number {
  const v = ss.baseIndexRef;
  return v && typeof v === 'object' && 'value' in v ? Number(v.value) : Number(v);
}

const listBucket: ListColumnWidthBucket = {
  kind: 'list',
  application: 'sandbox',
  modelName: 'Record',
  scopePath: '/web/sandbox/records',
};

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

describe('ChoyTableHost baseIndex', () => {
  beforeEach(() => {
    stubSfc(DataTable as any, {
      name: 'DataTable',
      props: {
        columns: { type: Array, default: () => [] },
        data: { type: Array, default: () => [] },
        rowId: { type: Function, default: undefined },
        height: { type: Number, default: 0 },
        estimateSize: { type: Number, default: 40 },
        enableRowSelection: { type: Boolean, default: false },
        enableSorting: { type: Boolean, default: false },
        showEmpty: { type: Boolean, default: false },
        sortingMode: { type: String, default: '' },
        virtualize: { type: Boolean, default: true },
      },
      emits: ['row-click', 'sort-change'],
      setup() {
        return () => h('div', { 'data-test': 'data-table-stub' });
      },
    } as any);
  });

  afterEach(() => {
    restoreSfc(DataTable as any);
  });

  test('accepts number and Ref baseIndex; pagination wins when present', async () => {
    const mNum = mountApp(ChoyTableHost as any, {
      props: { data: [{ Id: '1' }], baseIndex: 11, columns: [] },
    });
    await flushPromises();
    expect(readBaseIndex(mNum.setupState())).toBe(11);
    mNum.unmount();

    const bi = ref(21);
    const mRef = mountApp(ChoyTableHost as any, {
      props: { data: [{ Id: '1' }], baseIndex: bi, columns: [] },
    });
    await flushPromises();
    expect(readBaseIndex(mRef.setupState())).toBe(21);
    bi.value = 31;
    await flushPromises();
    expect(readBaseIndex(mRef.setupState())).toBe(31);
    mRef.unmount();

    const mPage = mountApp(ChoyTableHost as any, {
      props: {
        data: [{ Id: '1' }],
        baseIndex: 99,
        columns: [],
        store: { state: { pagination: { currentPage: 2, pageSize: 10 } } },
      },
    });
    await flushPromises();
    // page 2, size 10 → base index 11
    expect(readBaseIndex(mPage.setupState())).toBe(11);
    mPage.unmount();
  });
});

describe('ChoyTableHost column width prefs', () => {
  beforeEach(() => {
    stubSfc(DataTable as any, {
      name: 'DataTable',
      props: {
        columns: { type: Array, default: () => [] },
        data: { type: Array, default: () => [] },
        enableColumnResize: { type: Boolean, default: false },
        showResetColumnWidths: { type: Boolean, default: false },
      },
      emits: [
        'column-resize-start',
        'column-resize',
        'column-resize-end',
        'column-resize-reset',
        'reset-column-widths',
      ],
      setup() {
        return () => h('div', { 'data-test': 'data-table-stub' });
      },
    } as any);
    try {
      localStorage.clear();
    } catch {
      // ignore
    }
  });

  afterEach(() => {
    restoreSfc(DataTable as any);
  });

  test('loads, drags, persists, resets, and skips no-op end', async () => {
    writeListColumnWidthPrefs(listBucket, 'u1', { columns: { Name: 200 } });
    const mounted = mountApp(ChoyTableHost as any, {
      props: {
        data: [{ Id: '1', Name: 'a' }],
        columns: [],
        enableColumnResize: true,
        columnWidthBucket: listBucket,
        prefsUserId: 'u1',
      },
    });
    await flushPromises();
    const ss = mounted.setupState() as any;
    expect(ss.userWidths?.Name ?? ss.userWidths?.value?.Name).toBe(200);

    const userWidths = () =>
      (ss.userWidths && 'value' in ss.userWidths ? ss.userWidths.value : ss.userWidths) as Record<
        string,
        number
      >;

    ss.onColumnResizeStart?.({
      columnId: 'Name',
      width: 200,
      snapshot: { Name: 200, Code: 120 },
    });
    expect(userWidths().Name).toBe(200);
    // Force effectiveUserWidths computed (merge freeze + live overrides).
    void ss.columnDefs;
    void ss.effectiveUserWidths;

    ss.onColumnResize?.({ columnId: 'Name', width: 260 });
    expect(userWidths().Name).toBe(260);

    ss.onColumnResizeEnd?.({ columnId: 'Name', width: 260 });
    await sleep(LIST_COLUMN_WIDTH_WRITE_DEBOUNCE_MS + 50);
    await flushPromises();
    expect(readListColumnWidthPrefs(listBucket, 'u1').columns.Name).toBe(260);

    // No-op end restores pre-drag prefs (seed without move).
    ss.onColumnResizeStart?.({
      columnId: 'Name',
      width: 260,
      snapshot: { Name: 260 },
    });
    ss.onColumnResizeEnd?.({ columnId: 'Name', width: 260 });
    expect(userWidths().Name).toBe(260);

    // No-op on a column that had no prior override drops the seed.
    ss.onColumnResizeStart?.({
      columnId: 'Fresh',
      width: 140,
      snapshot: { Fresh: 140 },
    });
    ss.onColumnResizeEnd?.({ columnId: 'Fresh', width: 140 });
    expect(userWidths().Fresh).toBeUndefined();

    ss.onColumnResizeReset?.({ columnId: 'Name' });
    expect(userWidths().Name).toBeUndefined();
    expect(readListColumnWidthPrefs(listBucket, 'u1').columns.Name).toBeUndefined();

    ss.onColumnResizeStart?.({
      columnId: 'Title',
      width: 140,
      snapshot: { Title: 140 },
    });
    ss.onColumnResize?.({ columnId: 'Title', width: 220 });
    ss.onColumnResizeEnd?.({ columnId: 'Title', width: 220 });
    await sleep(LIST_COLUMN_WIDTH_WRITE_DEBOUNCE_MS + 50);
    expect(readListColumnWidthPrefs(listBucket, 'u1').columns.Title).toBe(220);

    ss.resetColumnWidths?.();
    expect(Object.keys(userWidths()).length).toBe(0);
    expect(readListColumnWidthPrefs(listBucket, 'u1').columns).toEqual({});

    mounted.unmount();
  });

  test('flush pending persist on unmount and cancel on bucket switch', async () => {
    const mounted = mountApp(ChoyTableHost as any, {
      props: {
        data: [{ Id: '1' }],
        columns: [],
        enableColumnResize: true,
        columnWidthBucket: listBucket,
        prefsUserId: 'u2',
      },
      reactiveProps: true,
    });
    await flushPromises();
    const ss = mounted.setupState() as any;

    ss.onColumnResizeStart?.({
      columnId: 'Name',
      width: 140,
      snapshot: { Name: 140 },
    });
    ss.onColumnResize?.({ columnId: 'Name', width: 300 });
    ss.onColumnResizeEnd?.({ columnId: 'Name', width: 300 });
    // Unmount before debounce fires → flushPendingPersist.
    mounted.unmount();
    expect(readListColumnWidthPrefs(listBucket, 'u2').columns.Name).toBe(300);

    const m2 = mountApp(ChoyTableHost as any, {
      props: {
        data: [{ Id: '1' }],
        columns: [],
        enableColumnResize: true,
        columnWidthBucket: listBucket,
        prefsUserId: 'u3',
      },
      reactiveProps: true,
    });
    await flushPromises();
    const ss2 = m2.setupState() as any;
    ss2.onColumnResizeStart?.({
      columnId: 'Name',
      width: 140,
      snapshot: { Name: 140 },
    });
    ss2.onColumnResize?.({ columnId: 'Name', width: 310 });
    ss2.onColumnResizeEnd?.({ columnId: 'Name', width: 310 });
    // Switch bucket before debounce — flush old then load empty new bucket.
    const other: ListColumnWidthBucket = {
      kind: 'list',
      application: 'sandbox',
      modelName: 'Other',
      scopePath: '/web/sandbox/other',
    };
    m2.props.columnWidthBucket = other;
    await flushPromises();
    expect(readListColumnWidthPrefs(listBucket, 'u3').columns.Name).toBe(310);
    m2.unmount();
  });
});
