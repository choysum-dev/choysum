// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, provide, ref } from 'vue';

import { fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import { useVTableProvideColumnRegistry, useVTableProvideBuildContext } from '@/web/web/composables/useVTable';

function mountHandleColumn(opts?: { enabled?: boolean; width?: number; colKey?: string; type?: string }) {
  const enabled = ref(opts?.enabled ?? true);
  const rows = [
    { Id: '1', Sequence: 1 },
    { Id: '2', Sequence: 2 },
  ];
  const onReorder = fnRecorder(async () => {});
  const handleApi = useListHandleReorder({
    rows: () => rows,
    enabled,
    onReorder,
  });

  let columnsRef: any = null;

  const Host = defineComponent({
    setup() {
      const reg = useVTableProvideColumnRegistry();
      columnsRef = reg.columns;
      useVTableProvideBuildContext({
        getRows: () => rows,
        baseIndex: ref(1),
      });
      provide(LIST_HANDLE_API_KEY, handleApi);

      const columnProps: Record<string, any> = {
        type: opts?.type ?? 'handle',
        colKey: opts?.colKey ?? '__handle__',
      };
      if (opts?.width !== undefined) {
        columnProps.vColumnProps = { width: opts.width, align: 'center' };
      } else if (opts?.type == null || opts?.type === 'handle') {
        columnProps.vColumnProps = { align: 'center' };
      }
      return () => h(OVColumn, columnProps);
    },
  });

  const m = mountApp(Host);
  return { handleApi, onReorder, get columns() { return columnsRef?.value ?? []; }, unmount: m.unmount };
}

describe('OVColumn handle type', () => {
  test('registers handle column with drag handlers', async () => {
    const { handleApi, columns, unmount } = mountHandleColumn({ width: 36 });
    await nextTick();
    expect(columns).toHaveLength(1);
    const col = columns[0];
    expect(col.dataKey).toBe('__handle__');

    const preventDefault = fnRecorder();
    const stopPropagation = fnRecorder();
    const setData = fnRecorder();
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('o-list-handle');
    expect(cell.props.draggable).toBe('true');

    cell.props.onDragstart({ preventDefault, stopPropagation, dataTransfer: { effectAllowed: '', setData } });
    expect(handleApi.draggingIndex.value).toBe(0);

    cell.props.onDragover({ preventDefault, stopPropagation, dataTransfer: { dropEffect: '' } });
    expect(preventDefault.calls.length).toBeGreaterThan(0);

    cell.props.onDrop({ preventDefault, stopPropagation });
    await nextTick();

    cell.props.onDragend({ stopPropagation });
    expect(handleApi.draggingIndex.value).toBeNull();

    cell.props.onClick({ stopPropagation });
    expect(stopPropagation.calls.length).toBeGreaterThan(0);
    unmount();
  });

  test('uses default handle width when vColumnProps.width is omitted', async () => {
    const { columns, unmount } = mountHandleColumn();
    await nextTick();
    expect(columns[0].width).toBe(36);
    unmount();
  });

  test('falls through handle branch when type is not handle', async () => {
    const { columns, unmount } = mountHandleColumn({ type: 'default', colKey: 'Name' });
    await nextTick();
    expect(columns.length).toBeGreaterThanOrEqual(1);
    unmount();
  });

  test('disables handle when reorder api is disabled', async () => {
    const { handleApi, columns, unmount } = mountHandleColumn({ enabled: false, width: 36 });
    await nextTick();
    const col = columns[0];
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('o-list-handle--disabled');
    expect(cell.props.draggable).toBe('false');
    expect(cell.props.title).toBe('');

    const preventDefault = fnRecorder();
    cell.props.onDragstart({
      preventDefault,
      stopPropagation: fnRecorder(),
      dataTransfer: { setData: fnRecorder() },
    });
    expect(preventDefault.calls.length).toBeGreaterThan(0);
    expect(handleApi.draggingIndex.value).toBeNull();
    unmount();
  });
});
