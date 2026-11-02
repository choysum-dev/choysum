// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useMetaModelDataDetailPage } from './meta_model_data_detail_page';

test('useMetaModelDataDetailPage binds fixed meta.MetaModelData store', () => {
  const created: Array<{ name: string; opts: Record<string, unknown> }> = [];
  const api = useMetaModelDataDetailPage({
    scopeManager: { id: 'menu' },
    createStore: ((name: string, opts: Record<string, unknown>) => {
      created.push({ name, opts });
      return { id: name } as any;
    }) as any,
  });
  expect(api.metaModelDataStore.value).toEqual({ id: 'meta.MetaModelData' });
  expect(created).toEqual([
    {
      name: 'meta.MetaModelData',
      opts: { storeId: 'MetaModelData_meta_admin', scopeManager: { id: 'menu' } },
    },
  ]);
});
