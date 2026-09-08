// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, provide, ref } from 'vue';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import {
  useVTableProvideBuildContext,
  useVTableProvideColumnRegistry,
  type ColumnRegistry,
} from '@/web/web/composables/useVTable';

function mountHandleColumn(opts?: { enabled?: boolean; width?: number; colKey?: string; type?: string }) {
  const enabled = ref(opts?.enabled ?? true);
  const rows = [
    { Id: '1', Sequence: 1 },
    { Id: '2', Sequence: 2 },
  ];
  const onReorder = fnRecorder(async () => undefined);
  let registry: ColumnRegistry | null = null;
  let handleApi: ReturnType<typeof useListHandleReorder> | null = null;

  const Host = defineComponent({
    setup() {
      registry = useVTableProvideColumnRegistry();
      useVTableProvideBuildContext({ getRows: () => rows, baseIndex: ref(1) });
      handleApi = useListHandleReorder({
        rows: () => rows,
        enabled,
        onReorder,
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

  const mounted = mountApp(Host);
  return { handleApi: handleApi!, onReorder, columns: () => registry!.columns.value, unmount: mounted.unmount };
}

describe('OVColumn handle type', () => {
  test('registers handle column with drag handlers', async () => {
    const { handleApi, columns, unmount } = mountHandleColumn({ width: 36 });
    await nextTick();
    await flushPromises();
    expect(columns()).toHaveLength(1);
    const col = columns()[0] as any;
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
    await flushPromises();
    expect((columns()[0] as any).width).toBe(36);
    unmount();
  });

  test('falls through handle branch when type is not handle', async () => {
    const { columns, unmount } = mountHandleColumn({ type: 'default', colKey: 'Name' });
    await nextTick();
    await flushPromises();
    expect(columns().length).toBeGreaterThanOrEqual(1);
    unmount();
  });

  test('disables handle when reorder api is disabled', async () => {
    const { handleApi, columns, unmount } = mountHandleColumn({ enabled: false, width: 36 });
    await nextTick();
    await flushPromises();
    const col = columns()[0] as any;
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('o-list-handle--disabled');
    expect(cell.props.draggable).toBe('false');
    expect(cell.props.title).toBe('');

    const preventDefault = fnRecorder();
    cell.props.onDragstart({ preventDefault, stopPropagation: fnRecorder(), dataTransfer: { setData: fnRecorder() } });
    expect(preventDefault.calls.length).toBeGreaterThan(0);
    expect(handleApi.draggingIndex.value).toBeNull();
    unmount();
  });
});
