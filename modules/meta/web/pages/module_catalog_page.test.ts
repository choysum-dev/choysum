// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useModuleCatalogPage } from './module_catalog_page';

test('useModuleCatalogPage binds MetaModuleCatalog and MetaModuleOperation stores', () => {
  const created: string[] = [];
  const { pageTitle, catalogStore, operationStore } = useModuleCatalogPage({
    scopeManager: {},
    createStore: ((model: string) => {
      created.push(model);
      return { model } as any;
    }) as any,
  });
  expect(created).toEqual(['meta.MetaModuleCatalog', 'meta.MetaModuleOperation']);
  expect(catalogStore).toEqual({ model: 'meta.MetaModuleCatalog' });
  expect(operationStore).toEqual({ model: 'meta.MetaModuleOperation' });
  expect(String(pageTitle)).toContain('Catalog');

  const list = useModuleCatalogPage({
    scopeManager: {},
    pageTitleSrc: 'Module List',
    createStore: (() => ({})) as any,
  });
  expect(String(list.pageTitle)).toContain('Module List');
});
