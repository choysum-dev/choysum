// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import {
  setHtml5DragImage,
  setListHandleDragImage,
  useListHandleReorder,
} from '@/web/web/composables/useListHandleReorder';
import { asyncFnRecorder, fnRecorder } from '@/web/web/__tests__/mountApp';

function dragEvent(partial?: Partial<DragEvent> & { throwTransfer?: boolean }): DragEvent {
  const transfer: any = {
    effectAllowed: '',
    dropEffect: '',
    setData: fnRecorder(),
    setDragImage: fnRecorder(),
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
    const onReorder = asyncFnRecorder(async () => {});
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
    expect(api.overIndex.value).toBe(0);
    expect((start.dataTransfer!.setData as any).calls.length).toBe(1);

    const over = dragEvent();
    api.onDragOver(2, over);
    expect((over.preventDefault as any).calls.length).toBe(1);
    expect(api.overIndex.value).toBe(2);

    const drop = dragEvent();
    await api.onDrop(2, drop);
    expect(api.overIndex.value).toBeNull();
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
    expect(api.overIndex.value).toBeNull();
  });

  test('overIndex clears on dragEnd without drop', () => {
    const enabled = ref(true);
    const api = useListHandleReorder({
      rows: () => [
        { Id: '1', Sequence: 1 },
        { Id: '2', Sequence: 2 },
      ],
      enabled,
      onReorder: fnRecorder(),
    });
    api.onDragStart(0, dragEvent());
    api.onDragOver(1, dragEvent());
    expect(api.overIndex.value).toBe(1);
    api.onDragEnd();
    expect(api.draggingIndex.value).toBeNull();
    expect(api.overIndex.value).toBeNull();
  });

  test('setHtml5DragImage / setListHandleDragImage tolerate missing transfer', () => {
    const el = document.createElement('div');
    el.setAttribute('role', 'row');
    const handle = document.createElement('span');
    el.appendChild(handle);
    document.body.appendChild(el);
    try {
      expect(() => setHtml5DragImage(el, { clientX: 0, clientY: 0 } as DragEvent)).not.toThrow();
      const setDragImage = fnRecorder();
      const e = {
        clientX: 10,
        clientY: 12,
        currentTarget: handle,
        dataTransfer: { setDragImage },
      } as unknown as DragEvent;
      Object.defineProperty(el, 'getBoundingClientRect', {
        value: () => ({ left: 0, top: 0, width: 100, height: 40 }),
      });
      setHtml5DragImage(el, e);
      expect(setDragImage.calls.length).toBe(1);
      expect(setDragImage.calls[0]).toEqual([el, 10, 12]);

      const setDragImage2 = fnRecorder();
      setListHandleDragImage({
        clientX: 5,
        clientY: 6,
        currentTarget: handle,
        dataTransfer: { setDragImage: setDragImage2 },
      } as unknown as DragEvent);
      expect(setDragImage2.calls.length).toBe(1);
      expect(setDragImage2.calls[0]?.[0]).toBe(el);

      // No matching ancestor → no-op.
      const orphan = document.createElement('span');
      expect(() =>
        setListHandleDragImage({
          currentTarget: orphan,
          dataTransfer: { setDragImage: fnRecorder() },
        } as unknown as DragEvent),
      ).not.toThrow();
    } finally {
      el.remove();
    }
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
    const onReorder = asyncFnRecorder(async () => {});
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
