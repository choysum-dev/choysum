// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, provide, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';

const columnsRef = ref<any[]>([]);

vi.mock('@/web/web/composables/useVTable', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/composables/useVTable')>('@/web/web/composables/useVTable');
  return {
    ...actual,
    useVTableUseColumnRegistry: () => ({ columns: columnsRef, register: vi.fn() }),
    useVTableUseBuildContext: () => ({ getRows: () => [], baseIndex: ref(1) }),
  };
});

function mountHandleColumn(opts?: { enabled?: boolean; width?: number; colKey?: string; type?: string }) {
  columnsRef.value = [];
  const enabled = ref(opts?.enabled ?? true);
  const rows = [
    { Id: '1', Sequence: 1 },
    { Id: '2', Sequence: 2 },
  ];
  const onReorder = vi.fn(async () => {});
  const handleApi = useListHandleReorder({
    rows: () => rows,
    enabled,
    onReorder,
  });

  const Host = defineComponent({
    setup() {
      provide(LIST_HANDLE_API_KEY, handleApi);
      const columnProps: Record<string, any> = {
        type: opts?.type ?? 'handle',
        colKey: opts?.colKey ?? '__handle__',
      };
      if (opts?.width !== undefined) {
        columnProps.vColumnProps = { width: opts.width, align: 'center' };
      } else if (opts?.type == null || opts?.type === 'handle') {
        // omit width to hit colProps.width ?? 36 fallback
        columnProps.vColumnProps = { align: 'center' };
      }
      return () => h(OVColumn, columnProps);
    },
  });

  mount(Host);
  return { handleApi, onReorder };
}

describe('OVColumn handle type', () => {
  it('registers handle column with drag handlers', async () => {
    const { handleApi } = mountHandleColumn({ width: 36 });
    await nextTick();
    expect(columnsRef.value).toHaveLength(1);
    const col = columnsRef.value[0];
    expect(col.dataKey).toBe('__handle__');

    const preventDefault = vi.fn();
    const stopPropagation = vi.fn();
    const setData = vi.fn();
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('o-list-handle');
    expect(cell.props.draggable).toBe('true');

    cell.props.onDragstart({ preventDefault, stopPropagation, dataTransfer: { effectAllowed: '', setData } });
    expect(handleApi.draggingIndex.value).toBe(0);

    cell.props.onDragover({ preventDefault, stopPropagation, dataTransfer: { dropEffect: '' } });
    expect(preventDefault).toHaveBeenCalled();

    cell.props.onDrop({ preventDefault, stopPropagation });
    await nextTick();

    cell.props.onDragend({ stopPropagation });
    expect(handleApi.draggingIndex.value).toBeNull();

    cell.props.onClick({ stopPropagation });
    expect(stopPropagation).toHaveBeenCalled();
  });

  it('uses default handle width when vColumnProps.width is omitted', async () => {
    mountHandleColumn();
    await nextTick();
    expect(columnsRef.value[0].width).toBe(36);
  });

  it('falls through handle branch when type is not handle', async () => {
    mountHandleColumn({ type: 'default', colKey: 'Name' });
    await nextTick();
    // default column still registers; ensures `type === 'handle'` false arm is hit
    expect(columnsRef.value.length).toBeGreaterThanOrEqual(1);
  });

  it('disables handle when reorder api is disabled', async () => {
    const { handleApi } = mountHandleColumn({ enabled: false, width: 36 });
    await nextTick();
    const col = columnsRef.value[0];
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('o-list-handle--disabled');
    expect(cell.props.draggable).toBe('false');
    expect(cell.props.title).toBe('');

    const preventDefault = vi.fn();
    cell.props.onDragstart({ preventDefault, stopPropagation: vi.fn(), dataTransfer: { setData: vi.fn() } });
    expect(preventDefault).toHaveBeenCalled();
    expect(handleApi.draggingIndex.value).toBeNull();
  });
});
