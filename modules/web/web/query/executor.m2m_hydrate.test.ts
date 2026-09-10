// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { execute } from './executor';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

test('execute hydrates ManyToManyRef id lists via createStoreByModel', async () => {
  const tagSearch = asyncFnRecorder(async () => [
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
    Search: asyncFnRecorder(async () => [{ Id: 'i1', Tags: ['t1', 't2'] }]),
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
  expect(store.Search.calls.length).toBe(1);
  expect(tagSearch.calls[0]?.[0]).toEqual(['Id', 'in', ['t1', 't2']]);

  const row = snapshot.rows[0] as any;
  expect(row?.payload?.Tags).toEqual([
    { Id: 't1', DisplayName: 'Red' },
    { Id: 't2', DisplayName: 'Blue' },
  ]);
});
