// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';

export type UserFilterDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createModelStore;
};

/**
 * Settings → web Saved Filters create/detail: fixed web.UserFilter store.
 */
export function useUserFilterDetailPage(deps: UserFilterDetailPageDeps): {
  userFilterStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const userFilterStore = shallowRef<ModelStore<any> | null>(
    createStore('web.UserFilter', {
      storeId: 'UserFilter_web_admin',
      scopeManager: deps.scopeManager,
    }) as ModelStore<any>
  );
  return { userFilterStore };
}
