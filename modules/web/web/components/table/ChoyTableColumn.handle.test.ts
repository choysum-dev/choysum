// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, provide, ref } from 'vue';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import {
  useTableProvideBuildContext,
  useTableProvideColumnRegistry,
  type ColumnRegistry,
} from '@/web/web/composables/useTable';

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
      registry = useTableProvideColumnRegistry();
      useTableProvideBuildContext({ getRows: () => rows, baseIndex: ref(1) });
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
      return () => h(ChoyTableColumn, columnProps);
    },
  });

  const mounted = mountApp(Host);
  return { handleApi: handleApi!, onReorder, columns: () => registry!.columns.value, unmount: mounted.unmount };
}

describe('ChoyTableColumn handle type', () => {
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
    const setDragImage = fnRecorder();
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('choy-list-handle');
    expect(cell.props.draggable).toBe('true');

    const handleEl = document.createElement('span');
    const rowEl = document.createElement('div');
    rowEl.setAttribute('role', 'row');
    rowEl.appendChild(handleEl);
    document.body.appendChild(rowEl);
    Object.defineProperty(rowEl, 'getBoundingClientRect', {
      value: () => ({ left: 0, top: 0, width: 80, height: 24 }),
    });
    try {
      cell.props.onDragstart({
        preventDefault,
        stopPropagation,
        clientX: 4,
        clientY: 6,
        currentTarget: handleEl,
        dataTransfer: { effectAllowed: '', setData, setDragImage },
      });
      expect(handleApi.draggingIndex.value).toBe(0);
      expect(handleApi.overIndex.value).toBe(0);
      expect(setDragImage.calls.length).toBe(1);
      expect(setDragImage.calls[0]?.[0]).toBe(rowEl);
    } finally {
      rowEl.remove();
    }

    cell.props.onDragover({ preventDefault, stopPropagation, dataTransfer: { dropEffect: '' } });
    expect(preventDefault.calls.length).toBeGreaterThan(0);
    expect(handleApi.overIndex.value).toBe(0);

    cell.props.onDrop({ preventDefault, stopPropagation });
    await nextTick();

    cell.props.onDragend({ stopPropagation });
    expect(handleApi.draggingIndex.value).toBeNull();

    cell.props.onClick({ stopPropagation });
    expect(stopPropagation.calls.length).toBeGreaterThan(0);
    unmount();
  });

  test('index column uses default width and muted tabular index class', async () => {
    let registry: ColumnRegistry | null = null;
    const Host = defineComponent({
      setup() {
        registry = useTableProvideColumnRegistry();
        useTableProvideBuildContext({ getRows: () => [{ Id: '1' }], baseIndex: ref(1) });
        return () => h(ChoyTableColumn, { type: 'index', colKey: '__index__' });
      },
    });
    const mounted = mountApp(Host);
    await nextTick();
    await flushPromises();
    const col = registry!.columns.value[0] as any;
    expect(col.width).toBe(44);
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.class).toContain('tabular-nums');
    expect(cell.props.class).toContain('text-muted-foreground');
    expect(String(cell.children)).toContain('1');
    mounted.unmount();
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
    expect(cell.props.class).toContain('choy-list-handle--disabled');
    expect(cell.props.draggable).toBe('false');
    expect(cell.props.title).toBe('');

    const preventDefault = fnRecorder();
    const setDragImage = fnRecorder();
    cell.props.onDragstart({
      preventDefault,
      stopPropagation: fnRecorder(),
      dataTransfer: { setData: fnRecorder(), setDragImage },
    });
    expect(preventDefault.calls.length).toBeGreaterThan(0);
    expect(handleApi.draggingIndex.value).toBeNull();
    expect(setDragImage.calls.length).toBe(0);
    unmount();
  });

  test('skips drag image when handle reorder api is missing', async () => {
    let registry: ColumnRegistry | null = null;
    const Host = defineComponent({
      setup() {
        registry = useTableProvideColumnRegistry();
        useTableProvideBuildContext({ getRows: () => [{ Id: '1' }], baseIndex: ref(1) });
        // Intentionally do not provide LIST_HANDLE_API_KEY.
        return () =>
          h(ChoyTableColumn, {
            type: 'handle',
            colKey: '__handle__',
            vColumnProps: { width: 36, align: 'center' },
          });
      },
    });
    const mounted = mountApp(Host);
    await nextTick();
    await flushPromises();
    const col = registry!.columns.value[0] as any;
    const cell = col.cellRenderer({ rowData: { Id: '1' }, rowIndex: 0 });
    expect(cell.props.draggable).toBe('false');
    const preventDefault = fnRecorder();
    const setDragImage = fnRecorder();
    cell.props.onDragstart({
      preventDefault,
      stopPropagation: fnRecorder(),
      currentTarget: document.createElement('span'),
      dataTransfer: { setData: fnRecorder(), setDragImage },
    });
    expect(preventDefault.calls.length).toBeGreaterThan(0);
    expect(setDragImage.calls.length).toBe(0);
    mounted.unmount();
  });

  test('skips re-registration when the column signature is unchanged', async () => {
    let registry: ColumnRegistry | null = null;
    let columnVm: any = null;
    const tick = ref(0);
    const Host = defineComponent({
      setup() {
        registry = useTableProvideColumnRegistry();
        useTableProvideBuildContext({ getRows: () => [], baseIndex: ref(1) });
        return () => [
          h('span', String(tick.value)),
          h(ChoyTableColumn, {
            prop: 'Name',
            label: 'Name',
            vColumnProps: { align: 'left', width: 80 },
            ref: (r: any) => {
              columnVm = r;
            },
          }),
        ];
      },
    });
    const mounted = mountApp(Host);
    await nextTick();
    await flushPromises();
    expect(registry!.columns.value).toHaveLength(1);
    const first = registry!.columns.value[0];
    tick.value = 1;
    await nextTick();
    await flushPromises();
    expect(registry!.columns.value[0]).toBe(first);
    columnVm?.$?.setupState?.registerOrReplace?.();
    expect(registry!.columns.value).toHaveLength(1);
    expect(registry!.columns.value[0]).toBe(first);
    mounted.unmount();
  });
});
