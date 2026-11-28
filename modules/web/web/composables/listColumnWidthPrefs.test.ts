// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyUserColumnWidthOverride,
  clearListColumnWidthPrefs,
  clampListColumnWidthPx,
  listColumnWidthBucketId,
  listColumnWidthStorageKey,
  readListColumnWidthPrefs,
  upsertListColumnWidthPref,
  writeListColumnWidthPrefs,
} from './listColumnWidthPrefs';

function memoryStorage(): Storage {
  const map = new Map<string, string>();
  return {
    get length() {
      return map.size;
    },
    clear() {
      map.clear();
    },
    getItem(key: string) {
      return map.has(key) ? (map.get(key) as string) : null;
    },
    key(index: number) {
      return [...map.keys()][index] ?? null;
    },
    removeItem(key: string) {
      map.delete(key);
    },
    setItem(key: string, value: string) {
      map.set(key, String(value));
    },
  } as Storage;
}

test('listColumnWidthBucketId separates list and embed scopes', () => {
  expect(
    listColumnWidthBucketId({
      kind: 'list',
      application: 'sandbox',
      modelName: 'Record',
      scopePath: '/web/sandbox/records/42?x=1',
    }),
  ).toBe('list|sandbox|Record|/web/sandbox/records/:id');

  expect(
    listColumnWidthBucketId({
      kind: 'embed',
      application: 'sandbox',
      parentModel: 'Record',
      fieldProp: 'Lines',
      relation: 'o2m',
    }),
  ).toBe('embed|sandbox|Record|Lines|o2m');

  expect(
    listColumnWidthBucketId({
      kind: 'embed',
      application: 'sandbox',
      parentModel: 'Record',
      fieldProp: 'Tags',
      relation: 'm2m',
    }),
  ).toBe('embed|sandbox|Record|Tags|m2m');
});

test('listColumnWidthStorageKey scopes by user', () => {
  expect(listColumnWidthStorageKey('usr_1')).toContain('usr_1');
  expect(listColumnWidthStorageKey('')).toContain('anon');
});

test('clampListColumnWidthPx enforces product bounds', () => {
  expect(clampListColumnWidthPx(20)).toBe(40);
  expect(clampListColumnWidthPx(160)).toBe(160);
  expect(clampListColumnWidthPx(9999)).toBe(800);
  expect(clampListColumnWidthPx(NaN)).toBeNull();
});

test('read/write/upsert/clear round-trip in memory storage', () => {
  const storage = memoryStorage();
  const bucket = {
    kind: 'list' as const,
    application: 'sandbox',
    modelName: 'Record',
    scopePath: '/web/sandbox/records',
  };
  expect(readListColumnWidthPrefs(bucket, 'u1', storage).columns).toEqual({});

  writeListColumnWidthPrefs(
    bucket,
    'u1',
    { columns: { Title: 220, Code: 120 } },
    storage,
  );
  expect(readListColumnWidthPrefs(bucket, 'u1', storage).columns).toEqual({
    Title: 220,
    Code: 120,
  });

  upsertListColumnWidthPref(bucket, 'u1', 'Title', 280, storage);
  expect(readListColumnWidthPrefs(bucket, 'u1', storage).columns.Title).toBe(280);

  clearListColumnWidthPrefs(bucket, 'u1', 'Code', storage);
  expect(readListColumnWidthPrefs(bucket, 'u1', storage).columns).toEqual({ Title: 280 });

  clearListColumnWidthPrefs(bucket, 'u1', undefined, storage);
  expect(readListColumnWidthPrefs(bucket, 'u1', storage).columns).toEqual({});
});

test('users do not share width prefs in the same storage', () => {
  const storage = memoryStorage();
  const bucket = {
    kind: 'embed' as const,
    application: 'sandbox',
    parentModel: 'Record',
    fieldProp: 'Lines',
    relation: 'o2m' as const,
  };
  upsertListColumnWidthPref(bucket, 'alice', 'Name', 200, storage);
  expect(readListColumnWidthPrefs(bucket, 'bob', storage).columns).toEqual({});
  expect(readListColumnWidthPrefs(bucket, 'alice', storage).columns.Name).toBe(200);
});

test('applyUserColumnWidthOverride prefers clamped user width', () => {
  expect(applyUserColumnWidthOverride(140, 200)).toBe(200);
  expect(applyUserColumnWidthOverride(140, 10)).toBe(40);
  expect(applyUserColumnWidthOverride(140, null)).toBe(140);
});
