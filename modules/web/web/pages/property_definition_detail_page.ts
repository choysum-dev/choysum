// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolvePropertyDefinitionDetailApplication } from './property_definition_page_bind';

export type PropertyDefinitionDetailPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Create/detail page bindings: resolve application and PropertyDefinition store.
 */
export function usePropertyDefinitionDetailPage(deps: PropertyDefinitionDetailPageDeps): {
  application: Ref<string>;
  definitionStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const application = computed(() => resolvePropertyDefinitionDetailApplication(deps.route));
  const definitionStore = shallowRef<WebModelStore<any> | null>(null);

  watch(
    application,
    (app) => {
      if (!app) {
        definitionStore.value = null;
        return;
      }
      definitionStore.value = createStore(`${app}.PropertyDefinition`, {
        storeId: `PropertyDefinition_${app}`,
        scopeManager: deps.scopeManager,
      }) as WebModelStore<any>;
    },
    { immediate: true }
  );

  return { application, definitionStore };
}
