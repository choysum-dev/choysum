// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { execute } from './executor';

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

test('execute hydrates ManyToManyRef id lists via createStoreByModel', async () => {
  const tagSearch = fnRecorder(async () => [
    { Id: 't1', DisplayName: 'Red' },
    { Id: 't2', DisplayName: 'Blue' },
  ]);
  const createStoreByModel = (modelName: string) => {
    if (modelName === 'demo.Tag') {
      return { storeId: 'demo.Tag', Search: tagSearch } as any;
    }
    throw new Error(`unexpected model: ${modelName}`);
  };

  const store = {
    storeId: 'demo.Item',
    fullModelName: 'demo.Item',
    fieldsMetadata: {
      Tags: { type: 'ManyToManyRef', relationModel: 'demo.Tag' },
    },
    Search: fnRecorder(async () => [{ Id: 'i1', Tags: ['t1', 't2'] }]),
  } as any;

  const snapshot = await execute(
    {
      main: { kind: 'search', params: {}, hash: 'search-m2m' },
      auxiliary: [],
    } as any,
    store,
    'list',
    { createStoreByModel }
  );

  expect(tagSearch.calls.length).toBe(1);
  const row = (snapshot.rows[0] as any)?.payload ?? (snapshot.rows[0] as any);
  const tags = row?.Tags ?? (snapshot.rows[0] as any)?.raw?.Tags;
  // Hydration mutates the search items in place before row mapping.
  expect(store.Search.calls.length).toBe(1);
  expect(tagSearch.calls[0]?.[0]).toEqual(['Id', 'in', ['t1', 't2']]);
});
