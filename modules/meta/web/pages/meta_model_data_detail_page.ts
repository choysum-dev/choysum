// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';

export type MetaModelDataDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createModelStore;
};

/**
 * Settings → meta External IDs detail: fixed meta.MetaModelData store (display-only Form).
 */
export function useMetaModelDataDetailPage(deps: MetaModelDataDetailPageDeps): {
  metaModelDataStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const metaModelDataStore = shallowRef<ModelStore<any> | null>(
    createStore('meta.MetaModelData', {
      storeId: 'MetaModelData_meta_admin',
      scopeManager: deps.scopeManager,
    }) as ModelStore<any>
  );
  return { metaModelDataStore };
}
