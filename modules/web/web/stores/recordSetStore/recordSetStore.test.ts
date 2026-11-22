// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import {
  canMoveRecordSet,
  clampRecordSetIndex,
  idAtGlobalIndex,
  resolveRecordSetAtIndex,
  resolveRecordSetNeighbor,
  useRecordSetStore,
  type RecordSetEntry,
} from './index';

function sampleEntry(over: Partial<RecordSetEntry> = {}): RecordSetEntry {
  return {
    key: 'base.Country|list',
    model: 'base.Country',
    condition: { IsActive: true },
    orderBy: [{ field: 'Name', direction: 'asc' }],
    total: 5,
    index: 1,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
    ...over,
  };
}

test('clampRecordSetIndex and idAtGlobalIndex cover window bounds', () => {
  expect(clampRecordSetIndex(-1, 5)).toBe(0);
  expect(clampRecordSetIndex(9, 5)).toBe(4);
  expect(clampRecordSetIndex(2, 0)).toBe(0);
  const entry = sampleEntry();
  expect(idAtGlobalIndex(entry, 1)).toBe('b');
  expect(idAtGlobalIndex(entry, 3)).toBeUndefined();
});

test('canMoveRecordSet respects total edges', () => {
  expect(canMoveRecordSet(sampleEntry({ index: 0 }), -1)).toBe(false);
  expect(canMoveRecordSet(sampleEntry({ index: 0 }), 1)).toBe(true);
  expect(canMoveRecordSet(sampleEntry({ index: 4, total: 5 }), 1)).toBe(false);
  expect(canMoveRecordSet(null, 1)).toBe(false);
});

test('resolveRecordSetNeighbor uses in-window ids without Search', async () => {
  const entry = sampleEntry({ index: 1 });
  const next = await resolveRecordSetNeighbor(entry, 1);
  expect(next?.id).toBe('c');
  expect(next?.entry.index).toBe(2);
});

test('resolveRecordSetNeighbor fetches and slides window when out of range', async () => {
  const entry = sampleEntry({
    total: 10,
    index: 2,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
  });
  const search = async (args: { offset: number; limit: number }) => {
    expect(args.offset).toBe(3);
    expect(args.limit).toBe(4);
    return ['d', 'e', 'f', 'g'];
  };
  const next = await resolveRecordSetNeighbor(entry, 1, search, 4);
  expect(next?.id).toBe('d');
  expect(next?.entry.index).toBe(3);
  expect(next?.entry.windowStart).toBe(3);
  expect(next?.entry.ids[0]).toBe('d');
});

test('resolveRecordSetNeighbor returns null when Search fails or empty', async () => {
  const entry = sampleEntry({ total: 10, index: 2, ids: ['a', 'b', 'c'], windowStart: 0 });
  expect(await resolveRecordSetNeighbor(entry, 1, async () => [])).toBeNull();
  expect(
    await resolveRecordSetNeighbor(entry, 1, async () => {
      throw new Error('boom');
    })
  ).toBeNull();
});

test('resolveRecordSetAtIndex jumps in-window and no-ops when already there', async () => {
  const entry = sampleEntry({ index: 1 });
  expect(await resolveRecordSetAtIndex(entry, 1)).toBeNull();
  const jumped = await resolveRecordSetAtIndex(entry, 0);
  expect(jumped?.id).toBe('a');
  expect(jumped?.entry.index).toBe(0);
  const toEndOfWindow = await resolveRecordSetAtIndex(entry, 2);
  expect(toEndOfWindow?.id).toBe('c');
  expect(toEndOfWindow?.entry.index).toBe(2);
  // Beyond total clamps to last index; outside window without Search → null.
  expect(await resolveRecordSetAtIndex(entry, 99)).toBeNull();
  const clamped = await resolveRecordSetAtIndex(entry, 99, async () => ['d', 'e']);
  expect(clamped?.id).toBe('d');
  expect(clamped?.entry.index).toBe(4);
});

test('resolveRecordSetAtIndex fetches when target is outside the window', async () => {
  const entry = sampleEntry({
    total: 10,
    index: 1,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
  });
  const search = async (args: { offset: number; limit: number }) => {
    expect(args.offset).toBe(7);
    expect(args.limit).toBe(4);
    return ['h', 'i', 'j'];
  };
  const next = await resolveRecordSetAtIndex(entry, 7, search, 4);
  expect(next?.id).toBe('h');
  expect(next?.entry.index).toBe(7);
  expect(next?.entry.windowStart).toBe(7);
});

test('useRecordSetStore seed / getActiveForModel / move / clear', async () => {
  setActivePinia(createPinia());
  const store = useRecordSetStore();
  store.seed(sampleEntry());
  expect(store.getActiveForModel('base.Country')?.ids).toEqual(['a', 'b', 'c']);
  expect(store.getActiveForModel('other.Model')).toBeNull();
  expect(store.canMove(1)).toBe(true);
  const moved = await store.move(1);
  expect(moved?.id).toBe('c');
  expect(store.active?.index).toBe(2);
  const jumped = await store.moveTo(0);
  expect(jumped?.id).toBe('a');
  expect(store.active?.index).toBe(0);
  store.clear();
  expect(store.active).toBeNull();
  expect(store.hasActive).toBe(false);
});

test('useRecordSetStore move/moveTo drop result when session key changes during await', async () => {
  setActivePinia(createPinia());
  const store = useRecordSetStore();
  store.seed(
    sampleEntry({
      total: 10,
      index: 2,
      ids: ['a', 'b', 'c'],
      windowStart: 0,
      key: 'session-a',
    })
  );
  let resolveSearch: ((ids: string[]) => void) | null = null;
  const pending = new Promise<string[]>(resolve => {
    resolveSearch = resolve;
  });
  const movePromise = store.move(1, async () => pending);
  store.clear();
  resolveSearch!(['d', 'e', 'f', 'g']);
  expect(await movePromise).toBeNull();
  expect(store.active).toBeNull();

  store.seed(
    sampleEntry({
      total: 10,
      index: 1,
      ids: ['a', 'b', 'c'],
      windowStart: 0,
      key: 'session-b',
    })
  );
  let resolveTo: ((ids: string[]) => void) | null = null;
  const pendingTo = new Promise<string[]>(resolve => {
    resolveTo = resolve;
  });
  const toPromise = store.moveTo(7, async () => pendingTo);
  store.seed(sampleEntry({ key: 'session-c', ids: ['x'], index: 0, total: 1, windowStart: 0 }));
  resolveTo!(['h', 'i', 'j']);
  expect(await toPromise).toBeNull();
  expect(store.active?.key).toBe('session-c');
});
