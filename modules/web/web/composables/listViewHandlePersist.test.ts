// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildHandleReorderWrites,
  persistHandleReorder,
  shouldDiscardInvisibleEdit,
  syncFlatRowsFromVisibleItems,
} from '@/web/web/composables/listViewHandlePersist';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

describe('listViewHandlePersist', () => {
  test('buildHandleReorderWrites skips rows without id', () => {
    const writes = buildHandleReorderWrites([
      { row: { kind: 'record', payload: { Id: '1', Sequence: 2 } }, previous: 1, next: 2 },
      { row: { kind: 'record', payload: { Name: 'no-id' } }, previous: 1, next: 2 },
    ]);
    expect(writes).toEqual([{ id: '1', previous: 1, next: 2 }]);
  });

  test('persistHandleReorder updates sequences and refreshes on success', async () => {
    const updateById = asyncFnRecorder(async () => {});
    const refresh = asyncFnRecorder(async () => {});
    const rollbackFlat = asyncFnRecorder();
    const onError = asyncFnRecorder();

    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat,
      onError,
    });

    expect(updateById.calls).toEqual([['a', { Sequence: 2 }]]);
    expect(refresh.calls.length).toBe(1);
    expect(rollbackFlat.calls.length).toBe(0);
    expect(onError.calls.length).toBe(0);
  });

  test('persistHandleReorder rolls back, restores flat rows, and reloads on failure', async () => {
    let n = 0;
    const updateById = asyncFnRecorder(async () => {
      n += 1;
      if (n === 2) throw new Error('fail');
    });
    const refresh = asyncFnRecorder(async () => {});
    const rollbackFlat = asyncFnRecorder();
    const onError = asyncFnRecorder();

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

    expect(updateById.calls[0]).toEqual(['a', { Sequence: 2 }]);
    expect(updateById.calls[1]).toEqual(['b', { Sequence: 1 }]);
    expect(updateById.calls[2]).toEqual(['a', { Sequence: 1 }]);
    expect(rollbackFlat.calls.length).toBe(1);
    expect(onError.calls).toEqual([['write']]);
    // Failure path refreshes once after rollback (success path also refreshes once).
    expect(refresh.calls.length).toBe(1);
  });

  test('persistHandleReorder ignores refresh failure after rollback', async () => {
    const updateById = asyncFnRecorder(async () => {
      throw new Error('fail');
    });
    const refresh = asyncFnRecorder(async () => {
      throw new Error('reload failed');
    });
    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh,
      rollbackFlat: asyncFnRecorder(),
      onError: asyncFnRecorder(),
    });
    expect(refresh.calls.length).toBe(1);
  });

  test('persistHandleReorder does not roll back writes when only refresh fails', async () => {
    const updateById = asyncFnRecorder(async () => {});
    const refresh = asyncFnRecorder(async () => {
      throw new Error('reload failed');
    });
    const rollbackFlat = asyncFnRecorder();
    const onError = asyncFnRecorder();
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
    expect(updateById.calls.length).toBe(2);
    expect(updateById.calls[0]).toEqual(['a', { Sequence: 2 }]);
    expect(updateById.calls[1]).toEqual(['b', { Sequence: 1 }]);
    expect(rollbackFlat.calls.length).toBe(0);
    expect(onError.calls).toEqual([['refresh']]);
    expect(refresh.calls.length).toBe(1);
  });

  test('persistHandleReorder ignores rollback UpdateById errors', async () => {
    let n = 0;
    const updateById = asyncFnRecorder(async () => {
      n += 1;
      if (n === 1) throw new Error('fail');
      throw new Error('rollback fail');
    });
    const onError = asyncFnRecorder();
    await persistHandleReorder({
      writes: [{ id: 'a', previous: 1, next: 2 }],
      handleField: 'Sequence',
      updateById,
      refresh: asyncFnRecorder(async () => {}),
      rollbackFlat: asyncFnRecorder(),
      onError,
    });
    expect(onError.calls).toEqual([['write']]);
    expect(updateById.calls.length).toBe(2);
  });

  test('persistHandleReorder skips rollback write when previous is undefined', async () => {
    const updateById = asyncFnRecorder(async () => {
      throw new Error('fail');
    });
    await persistHandleReorder({
      writes: [{ id: 'a', previous: undefined, next: 1 }],
      handleField: 'Sequence',
      updateById,
      refresh: asyncFnRecorder(async () => {}),
      rollbackFlat: asyncFnRecorder(),
      onError: asyncFnRecorder(),
    });
    expect(updateById.calls.length).toBe(1);
  });

  test('persistHandleReorder no-ops on empty writes', async () => {
    const updateById = asyncFnRecorder();
    await persistHandleReorder({
      writes: [],
      handleField: 'Sequence',
      updateById,
      refresh: asyncFnRecorder(),
      rollbackFlat: asyncFnRecorder(),
      onError: asyncFnRecorder(),
    });
    expect(updateById.calls.length).toBe(0);
  });

  test('shouldDiscardInvisibleEdit detects row leaving visible set', () => {
    const items = [{ kind: 'record', payload: { Id: '2' } }];
    expect(shouldDiscardInvisibleEdit(true, '1', items, true)).toEqual({ discard: true, warn: true });
    expect(shouldDiscardInvisibleEdit(true, '2', items, false)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(false, '1', items, true)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(true, null, items, true)).toEqual({ discard: false, warn: false });
    expect(shouldDiscardInvisibleEdit(true, '1', null as any, false)).toEqual({ discard: true, warn: false });
  });

  test('syncFlatRowsFromVisibleItems filters record rows and skips group mode', () => {
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
