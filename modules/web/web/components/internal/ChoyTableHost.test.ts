// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, ref } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyTableHost from './ChoyTableHost.vue';
import DataTable from './DataTable.vue';

function readBaseIndex(ss: any): number {
  const v = ss.baseIndexRef;
  return v && typeof v === 'object' && 'value' in v ? Number(v.value) : Number(v);
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
