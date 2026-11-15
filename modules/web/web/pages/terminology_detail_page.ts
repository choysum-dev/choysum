// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { resolveTerminologyDetailApplication } from './terminology_page_bind';

export type TerminologyDetailPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createModelStore;
};

/**
 * Detail page bindings: resolve application and TranslationTerm store.
 */
export function useTerminologyDetailPage(deps: TerminologyDetailPageDeps): {
  application: Ref<string>;
  termStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const application = computed(() => resolveTerminologyDetailApplication(deps.route));
  const termStore = shallowRef<ModelStore<any> | null>(null);

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
      }) as ModelStore<any>;
    },
    { immediate: true }
  );

  return { application, termStore };
}
