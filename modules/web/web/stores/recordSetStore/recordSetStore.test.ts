// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import {
  canMoveRecordSet,
  clampRecordSetIndex,
  idAtGlobalIndex,
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
  store.clear();
  expect(store.active).toBeNull();
  expect(store.hasActive).toBe(false);
});
