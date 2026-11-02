// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';

const { _t: defaultTranslate } = createTranslate('web', {
  scope: 'web/pages/UserFilterList',
});

export type UserFilterListPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * Settings → web Saved Filters list: fixed web.UserFilter store (Owner B/web).
 * Primary Favorites UX remains OSearch; this page is the sys.admin ledger.
 */
export function useUserFilterListPage(deps: UserFilterListPageDeps): {
  pageTitle: Ref<string>;
  userFilterStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const pageTitle = computed(() => {
    if (deps.translate) return deps.translate('Saved Filters');
    return String(defaultTranslate('Saved Filters'));
  });
  const userFilterStore = shallowRef<WebModelStore<any> | null>(
    createStore('web.UserFilter', {
      storeId: 'UserFilter_web_admin',
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>
  );

  return { pageTitle, userFilterStore };
}
