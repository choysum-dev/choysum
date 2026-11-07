// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createTranslate } from '@/web/web/i18n';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';

export type ModuleCatalogPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Module catalog page stores: MetaModuleCatalog list + MetaModuleOperation for drawer deep links.
 */
export function useModuleCatalogPage(deps: ModuleCatalogPageDeps): {
  pageTitle: string;
  catalogStore: WebModelStore<MetaModuleCatalog>;
  operationStore: WebModelStore<MetaModuleOperation>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleCatalog' });
  const catalogStore = createStore('meta.MetaModuleCatalog', {
    storeId: 'MetaModuleCatalog_Catalog',
    scopeManager: deps.scopeManager,
  }) as WebModelStore<MetaModuleCatalog>;
  const operationStore = createStore('meta.MetaModuleOperation', {
    storeId: 'MetaModuleOperation_CatalogDrawer',
    scopeManager: deps.scopeManager,
  }) as WebModelStore<MetaModuleOperation>;
  return {
    pageTitle: _t('Modules'),
    catalogStore,
    operationStore,
  };
}
