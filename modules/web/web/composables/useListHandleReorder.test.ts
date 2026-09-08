// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useListHandleReorder } from '@/web/web/composables/useListHandleReorder';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function dragEvent(partial?: Partial<DragEvent> & { throwTransfer?: boolean }): DragEvent {
  const transfer: any = {
    effectAllowed: '',
    dropEffect: '',
    setData: fnRecorder(),
  };
  if (partial?.throwTransfer) {
    Object.defineProperty(transfer, 'effectAllowed', {
      set() {
        throw new Error('dt');
      },
      get() {
        return '';
      },
    });
  }
  return {
    preventDefault: fnRecorder(),
    stopPropagation: fnRecorder(),
    dataTransfer: transfer,
    ...partial,
  } as any;
}

describe('useListHandleReorder', () => {
  test('ignores drag when disabled', () => {
    const enabled = ref(false);
    const onReorder = fnRecorder();
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder,
    });
    const e = dragEvent();
    api.onDragStart(0, e);
    expect((e.preventDefault as any).calls.length).toBe(1);
    expect(api.draggingIndex.value).toBeNull();
  });

  test('reorders and renumbers with sequenceStart and getRecord', async () => {
    const enabled = ref(true);
    const rows = [
      { kind: 'record', payload: { Id: 'a', Sequence: 21 } },
      { kind: 'record', payload: { Id: 'b', Sequence: 22 } },
      { kind: 'record', payload: { Id: 'c', Sequence: 23 } },
    ];
    const onReorder = fnRecorder(async () => {});
    const api = useListHandleReorder({
      rows: () => rows,
      enabled,
      handleField: 'Sequence',
      sequenceStart: () => 21,
      getRecord: r => r.payload,
      onReorder,
    });

    const start = dragEvent();
    api.onDragStart(0, start);
    expect(api.draggingIndex.value).toBe(0);
    expect((start.dataTransfer!.setData as any).calls.length).toBe(1);

    const over = dragEvent();
    api.onDragOver(2, over);
    expect((over.preventDefault as any).calls.length).toBe(1);

    const drop = dragEvent();
    await api.onDrop(2, drop);
    expect(onReorder.calls.length).toBe(1);
    const [nextRows, changed] = onReorder.calls[0] as [any[], any[]];
    expect(nextRows.map((r: any) => r.payload.Id)).toEqual(['b', 'c', 'a']);
    expect(nextRows.map((r: any) => r.payload.Sequence)).toEqual([21, 22, 23]);
    expect(changed.map((c: any) => [c.row.payload.Id, c.previous, c.next])).toEqual([
      ['b', 22, 21],
      ['c', 23, 22],
      ['a', 21, 23],
    ]);

    api.onDragEnd();
    expect(api.draggingIndex.value).toBeNull();
  });

  test('no-ops when drop index equals drag index or out of range', async () => {
    const enabled = ref(true);
    const onReorder = fnRecorder();
    const api = useListHandleReorder({
      rows: () => [
        { Id: '1', Sequence: 1 },
        { Id: '2', Sequence: 2 },
      ],
      enabled,
      onReorder,
    });
    api.onDragStart(1, dragEvent());
    await api.onDrop(1, dragEvent());
    expect(onReorder.calls.length).toBe(0);

    api.onDragStart(0, dragEvent());
    await api.onDrop(99, dragEvent());
    expect(onReorder.calls.length).toBe(0);
  });

  test('no-ops drop when disabled or draggingIndex null', async () => {
    const enabled = ref(true);
    const onReorder = fnRecorder();
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder,
    });
    await api.onDrop(0, dragEvent());
    expect(onReorder.calls.length).toBe(0);

    enabled.value = false;
    api.draggingIndex.value = 0;
    await api.onDrop(0, dragEvent());
    expect(onReorder.calls.length).toBe(0);
  });

  test('tolerates dataTransfer assignment failures', () => {
    const enabled = ref(true);
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder: fnRecorder(),
    });
    expect(() => api.onDragStart(0, dragEvent({ throwTransfer: true }))).not.toThrow();
    api.draggingIndex.value = 0;
    expect(() => api.onDragOver(0, dragEvent({ throwTransfer: true }))).not.toThrow();
  });

  test('skips dragOver when not dragging', () => {
    const enabled = ref(true);
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder: fnRecorder(),
    });
    const e = dragEvent();
    api.onDragOver(0, e);
    expect((e.preventDefault as any).calls.length).toBe(0);
  });

  test('defaults handleField to Sequence when omitted', async () => {
    const enabled = ref(true);
    const rows = [
      { Id: 'a', Sequence: 1 },
      { Id: 'b', Sequence: 2 },
    ];
    const onReorder = fnRecorder(async () => {});
    const api = useListHandleReorder({
      rows: () => rows,
      enabled,
      onReorder,
    });
    api.onDragStart(0, dragEvent());
    await api.onDrop(1, dragEvent());
    expect(onReorder.calls.length).toBe(1);
    const [nextRows] = onReorder.calls[0] as [any[]];
    expect(nextRows.map((r: any) => r.Id)).toEqual(['b', 'a']);
    expect(nextRows.map((r: any) => r.Sequence)).toEqual([1, 2]);
  });
});
