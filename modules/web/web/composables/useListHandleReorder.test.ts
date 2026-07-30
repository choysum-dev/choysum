// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { useListHandleReorder } from '@/web/web/composables/useListHandleReorder';

function dragEvent(partial?: Partial<DragEvent> & { throwTransfer?: boolean }): DragEvent {
  const transfer: any = {
    effectAllowed: '',
    dropEffect: '',
    setData: vi.fn(),
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
    preventDefault: vi.fn(),
    stopPropagation: vi.fn(),
    dataTransfer: transfer,
    ...partial,
  } as any;
}

describe('useListHandleReorder', () => {
  it('ignores drag when disabled', () => {
    const enabled = ref(false);
    const onReorder = vi.fn();
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder,
    });
    const e = dragEvent();
    api.onDragStart(0, e);
    expect(e.preventDefault).toHaveBeenCalled();
    expect(api.draggingIndex.value).toBeNull();
  });

  it('reorders and renumbers with sequenceStart and getRecord', async () => {
    const enabled = ref(true);
    const rows = [
      { kind: 'record', payload: { Id: 'a', Sequence: 21 } },
      { kind: 'record', payload: { Id: 'b', Sequence: 22 } },
      { kind: 'record', payload: { Id: 'c', Sequence: 23 } },
    ];
    const onReorder = vi.fn(async () => {});
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
    expect(start.dataTransfer!.setData).toHaveBeenCalled();

    const over = dragEvent();
    api.onDragOver(2, over);
    expect(over.preventDefault).toHaveBeenCalled();

    const drop = dragEvent();
    await api.onDrop(2, drop);
    expect(onReorder).toHaveBeenCalled();
    const [nextRows, changed] = onReorder.mock.calls[0];
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

  it('no-ops when drop index equals drag index or out of range', async () => {
    const enabled = ref(true);
    const onReorder = vi.fn();
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
    expect(onReorder).not.toHaveBeenCalled();

    api.onDragStart(0, dragEvent());
    await api.onDrop(99, dragEvent());
    expect(onReorder).not.toHaveBeenCalled();
  });

  it('no-ops drop when disabled or draggingIndex null', async () => {
    const enabled = ref(true);
    const onReorder = vi.fn();
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder,
    });
    await api.onDrop(0, dragEvent());
    expect(onReorder).not.toHaveBeenCalled();

    enabled.value = false;
    api.draggingIndex.value = 0;
    await api.onDrop(0, dragEvent());
    expect(onReorder).not.toHaveBeenCalled();
  });

  it('tolerates dataTransfer assignment failures', () => {
    const enabled = ref(true);
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder: vi.fn(),
    });
    expect(() => api.onDragStart(0, dragEvent({ throwTransfer: true }))).not.toThrow();
    api.draggingIndex.value = 0;
    expect(() => api.onDragOver(0, dragEvent({ throwTransfer: true }))).not.toThrow();
  });

  it('skips dragOver when not dragging', () => {
    const enabled = ref(true);
    const api = useListHandleReorder({
      rows: () => [{ Id: '1', Sequence: 1 }],
      enabled,
      onReorder: vi.fn(),
    });
    const e = dragEvent();
    api.onDragOver(0, e);
    expect(e.preventDefault).not.toHaveBeenCalled();
  });

  it('defaults handleField to Sequence when omitted', async () => {
    const enabled = ref(true);
    const rows = [
      { Id: 'a', Sequence: 1 },
      { Id: 'b', Sequence: 2 },
    ];
    const onReorder = vi.fn(async () => {});
    const api = useListHandleReorder({
      rows: () => rows,
      enabled,
      onReorder,
    });
    api.onDragStart(0, dragEvent());
    await api.onDrop(1, dragEvent());
    expect(onReorder).toHaveBeenCalled();
    const [nextRows] = onReorder.mock.calls[0];
    expect(nextRows.map((r: any) => r.Id)).toEqual(['b', 'a']);
    expect(nextRows.map((r: any) => r.Sequence)).toEqual([1, 2]);
  });
});
