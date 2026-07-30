// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import {
  buildHandleReorderWrites,
  persistHandleReorder,
  shouldDiscardInvisibleEdit,
  syncFlatRowsFromVisibleItems,
} from '@/web/web/composables/listViewHandlePersist';

describe('listViewHandlePersist', () => {
  it('buildHandleReorderWrites skips rows without id', () => {
    const writes = buildHandleReorderWrites([
      { row: { kind: 'record', payload: { Id: '1', Sequence: 2 } }, previous: 1, next: 2 },
      { row: { kind: 'record', payload: { Name: 'no-id' } }, previous: 1, next: 2 },
    ]);
    expect(writes).toEqual([{ id: '1', previous: 1, next: 2 }]);
  });

  it('persistHandleReorder updates sequences and refreshes on success', async () => {
    const updateById = vi.fn(async () => {});
    const refresh = vi.fn(async () => {});
    const rollbackFlat = vi.fn();
    const onError = vi.fn();

    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat,
      onError,
    });

    expect(updateById).toHaveBeenCalledWith('a', { Sequence: 2 });
    expect(refresh).toHaveBeenCalled();
    expect(rollbackFlat).not.toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it('persistHandleReorder rolls back, restores flat rows, and reloads on failure', async () => {
    const updateById = vi
      .fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('fail'))
      .mockResolvedValue(undefined);
    const refresh = vi.fn(async () => {});
    const rollbackFlat = vi.fn();
    const onError = vi.fn();

    await persistHandleReorder({
      writes: [
        { id: 'a', previous: 1, next: 2 },
        { id: 'b', previous: 2, next: 1 },
      ],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat,
      onError,
    });

    expect(updateById).toHaveBeenCalledWith('a', { Sequence: 2 });
    expect(updateById).toHaveBeenCalledWith('b', { Sequence: 1 });
    expect(updateById).toHaveBeenCalledWith('a', { Sequence: 1 });
    expect(rollbackFlat).toHaveBeenCalled();
    expect(onError).toHaveBeenCalledWith('write');
    // Failure path refreshes once after rollback (success path also refreshes once).
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('persistHandleReorder ignores refresh failure after rollback', async () => {
    const updateById = vi.fn(async () => {
      throw new Error('fail');
    });
    const refresh = vi.fn(async () => {
      throw new Error('reload failed');
    });
    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat: vi.fn(),
      onError: vi.fn(),
    });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('persistHandleReorder does not roll back writes when only refresh fails', async () => {
    const updateById = vi.fn(async () => {});
    const refresh = vi.fn(async () => {
      throw new Error('reload failed');
    });
    const rollbackFlat = vi.fn();
    const onError = vi.fn();
    await persistHandleReorder({
      writes: [
        { id: 'a', previous: 1, next: 2 },
        { id: 'b', previous: 2, next: 1 },
      ],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat,
      onError,
    });
    expect(updateById).toHaveBeenCalledTimes(2);
    expect(updateById).toHaveBeenCalledWith('a', { Sequence: 2 });
    expect(updateById).toHaveBeenCalledWith('b', { Sequence: 1 });
    expect(rollbackFlat).not.toHaveBeenCalled();
    expect(onError).toHaveBeenCalledWith('refresh');
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('persistHandleReorder ignores rollback UpdateById errors', async () => {
    const updateById = vi
      .fn()
      .mockRejectedValueOnce(new Error('fail'))
      .mockRejectedValueOnce(new Error('rollback fail'));
    const onError = vi.fn();
    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh: vi.fn(async () => {}),
      rollbackFlat: vi.fn(),
      onError,
    });
    expect(onError).toHaveBeenCalledWith('write');
    expect(updateById).toHaveBeenCalledTimes(2);
  });

  it('persistHandleReorder skips rollback write when previous is undefined', async () => {
    const updateById = vi.fn(async () => {
      throw new Error('fail');
    });
    await persistHandleReorder({
      writes: [{ id: 'a', previous: undefined, next: 1 }],
      handleField: 'Sequence',
      updateById,
      refresh: vi.fn(async () => {}),
      rollbackFlat: vi.fn(),
      onError: vi.fn(),
    });
    expect(updateById).toHaveBeenCalledTimes(1);
  });

  it('persistHandleReorder no-ops on empty writes', async () => {
    const updateById = vi.fn();
    await persistHandleReorder({
      writes: [],
      handleField: 'Sequence',
      updateById,
      refresh: vi.fn(),
      rollbackFlat: vi.fn(),
      onError: vi.fn(),
    });
    expect(updateById).not.toHaveBeenCalled();
  });

  it('shouldDiscardInvisibleEdit detects row leaving visible set', () => {
    const items = [{ kind: 'record', payload: { Id: '2' } }];
    expect(shouldDiscardInvisibleEdit(true, '1', items, true)).toEqual({ discard: true, warn: true });
    expect(shouldDiscardInvisibleEdit(true, '2', items, false)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(false, '1', items, true)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(true, null, items, true)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(true, '1', null as any, false)).toEqual({ discard: true, warn: false });
  });

  it('syncFlatRowsFromVisibleItems filters record rows and skips group mode', () => {
    const items = [
      { kind: 'group', key: 'g' },
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A' } },
    ];
    expect(syncFlatRowsFromVisibleItems(items, true)).toEqual([]);
    expect(syncFlatRowsFromVisibleItems(null as any, false)).toEqual([]);
    const flat = syncFlatRowsFromVisibleItems(items, false);
    expect(flat).toHaveLength(1);
    expect(flat[0].payload).toEqual({ Id: '1', Name: 'A' });
  });
});
