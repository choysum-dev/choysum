// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { resolveFieldDefaultDetailApplication } from './field_default_page_bind';

export type FieldDefaultDetailPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createModelStore;
};

/**
 * Create/detail page bindings: resolve application and FieldDefault store.
 */
export function useFieldDefaultDetailPage(deps: FieldDefaultDetailPageDeps): {
  application: Ref<string>;
  fieldDefaultStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const application = computed(() => resolveFieldDefaultDetailApplication(deps.route));
  const fieldDefaultStore = shallowRef<ModelStore<any> | null>(null);

  watch(
    application,
    (app) => {
      if (!app) {
        fieldDefaultStore.value = null;
        return;
      }
      fieldDefaultStore.value = createStore(`${app}.FieldDefault`, {
        storeId: `FieldDefault_${app}`,
        scopeManager: deps.scopeManager,
      }) as ModelStore<any>;
    },
    { immediate: true }
  );

  return { application, fieldDefaultStore };
}
