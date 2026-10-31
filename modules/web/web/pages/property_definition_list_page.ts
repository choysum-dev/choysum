// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolvePropertyDefinitionListApplication } from './property_definition_page_bind';

const { _t: defaultTranslate } = createTranslate('web', {
  scope: 'web/pages/PropertyDefinitionList',
});

export type PropertyDefinitionListPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * List page bindings: resolve application, title, and PropertyDefinition store.
 */
export function usePropertyDefinitionListPage(deps: PropertyDefinitionListPageDeps): {
  application: Ref<string>;
  pageTitle: Ref<string>;
  definitionStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;

  const application = computed(() => resolvePropertyDefinitionListApplication(deps.route));
  const pageTitle = computed(() => {
    const appLabel = application.value || '—';
    if (deps.translate) {
      return deps.translate('Property Definitions: %s', appLabel);
    }
    return String(defaultTranslate('Property Definitions: %s', appLabel));
  });
  const definitionStore = shallowRef<WebModelStore<any> | null>(null);

  function bindStore(app: string) {
    if (!app) {
      definitionStore.value = null;
      return;
    }
    definitionStore.value = createStore(`${app}.PropertyDefinition`, {
      storeId: `PropertyDefinition_${app}`,
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>;
  }

  watch(application, (app) => bindStore(app), { immediate: true });

  return { application, pageTitle, definitionStore };
}
