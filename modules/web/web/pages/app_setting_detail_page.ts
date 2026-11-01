// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolveAppSettingDetailApplication } from './app_setting_page_bind';

export type AppSettingDetailPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Create/detail page bindings: resolve application and AppSetting store.
 */
export function useAppSettingDetailPage(deps: AppSettingDetailPageDeps): {
  application: Ref<string>;
  appSettingStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const application = computed(() => resolveAppSettingDetailApplication(deps.route));
  const appSettingStore = shallowRef<WebModelStore<any> | null>(null);

  watch(
    application,
    (app) => {
      if (!app) {
        appSettingStore.value = null;
        return;
      }
      appSettingStore.value = createStore(`${app}.AppSetting`, {
        storeId: `AppSetting_${app}`,
        scopeManager: deps.scopeManager,
      }) as WebModelStore<any>;
    },
    { immediate: true }
  );

  return { application, appSettingStore };
}
