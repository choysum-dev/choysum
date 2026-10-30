// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolveTerminologyDetailApplication } from './terminology_page_bind';

export type TerminologyDetailPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Detail page bindings: resolve application and TranslationTerm store.
 */
export function useTerminologyDetailPage(deps: TerminologyDetailPageDeps): {
  application: Ref<string>;
  termStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const application = computed(() => resolveTerminologyDetailApplication(deps.route));
  const termStore = shallowRef<WebModelStore<any> | null>(null);

  watch(
    application,
    (app) => {
      if (!app) {
        termStore.value = null;
        return;
      }
      termStore.value = createStore(`${app}.TranslationTerm`, {
        storeId: `TranslationTerm_${app}`,
        scopeManager: deps.scopeManager,
      }) as WebModelStore<any>;
    },
    { immediate: true }
  );

  return { application, termStore };
}
