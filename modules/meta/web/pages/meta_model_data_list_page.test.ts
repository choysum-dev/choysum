// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useMetaModelDataListPage } from './meta_model_data_list_page';

test('useMetaModelDataListPage binds fixed meta.MetaModelData store and title', () => {
  const created: string[] = [];
  const api = useMetaModelDataListPage({
    scopeManager: {},
    createStore: ((name: string) => {
      created.push(name);
      return { id: name } as any;
    }) as any,
    translate: m => m,
  });
  expect(api.pageTitle.value).toBe('External IDs');
  expect(api.metaModelDataStore.value).toEqual({ id: 'meta.MetaModelData' });
  expect(created).toEqual(['meta.MetaModelData']);
});
