// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';

export type MetaModelDataDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Settings → meta External IDs detail: fixed meta.MetaModelData store (display-only Form).
 */
export function useMetaModelDataDetailPage(deps: MetaModelDataDetailPageDeps): {
  metaModelDataStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const metaModelDataStore = shallowRef<WebModelStore<any> | null>(
    createStore('meta.MetaModelData', {
      storeId: 'MetaModelData_meta_admin',
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>
  );
  return { metaModelDataStore };
}
