// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';

export type UserFilterDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Settings → web Saved Filters create/detail: fixed web.UserFilter store.
 */
export function useUserFilterDetailPage(deps: UserFilterDetailPageDeps): {
  userFilterStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const userFilterStore = shallowRef<WebModelStore<any> | null>(
    createStore('web.UserFilter', {
      storeId: 'UserFilter_web_admin',
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>
  );
  return { userFilterStore };
}
