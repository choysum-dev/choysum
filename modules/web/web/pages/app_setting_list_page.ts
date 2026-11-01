// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolveAppSettingListApplication } from './app_setting_page_bind';

const { _t: defaultTranslate } = createTranslate('web', {
  scope: 'web/pages/AppSettingList',
});

export type AppSettingListPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * List page bindings: resolve application, title, and AppSetting store.
 */
export function useAppSettingListPage(deps: AppSettingListPageDeps): {
  application: Ref<string>;
  pageTitle: Ref<string>;
  appSettingStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;

  const application = computed(() => resolveAppSettingListApplication(deps.route));
  const pageTitle = computed(() => {
    const appLabel = application.value || '—';
    if (deps.translate) {
      return deps.translate('Parameters: %s', appLabel);
    }
    return String(defaultTranslate('Parameters: %s', appLabel));
  });
  const appSettingStore = shallowRef<WebModelStore<any> | null>(null);

  function bindStore(app: string) {
    if (!app) {
      appSettingStore.value = null;
      return;
    }
    appSettingStore.value = createStore(`${app}.AppSetting`, {
      storeId: `AppSetting_${app}`,
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>;
  }

  watch(application, (app) => bindStore(app), { immediate: true });

  return { application, pageTitle, appSettingStore };
}
