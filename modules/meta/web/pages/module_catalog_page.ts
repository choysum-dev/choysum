// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { createTranslate } from '@/web/web/i18n';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';

export type ModuleCatalogPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createModelStore;
  /** Kanban chrome uses Catalog; list chrome uses Module List. */
  pageTitleSrc?: 'Catalog' | 'Module List';
};

/**
 * Module catalog page stores: MetaModuleCatalog list + MetaModuleOperation for drawer deep links.
 */
export function useModuleCatalogPage(deps: ModuleCatalogPageDeps): {
  pageTitle: string;
  catalogStore: ModelStore<MetaModuleCatalog>;
  operationStore: ModelStore<MetaModuleOperation>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleCatalog' });
  const catalogStore = createStore('meta.MetaModuleCatalog', {
    storeId: 'MetaModuleCatalog_Catalog',
    scopeManager: deps.scopeManager,
  }) as ModelStore<MetaModuleCatalog>;
  const operationStore = createStore('meta.MetaModuleOperation', {
    storeId: 'MetaModuleOperation_CatalogDrawer',
    scopeManager: deps.scopeManager,
  }) as ModelStore<MetaModuleOperation>;
  return {
    pageTitle: _t(deps.pageTitleSrc || 'Catalog'),
    catalogStore,
    operationStore,
  };
}
