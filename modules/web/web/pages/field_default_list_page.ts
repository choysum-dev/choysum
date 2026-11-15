// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createModelStore } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { resolveFieldDefaultListApplication } from './field_default_page_bind';

const { _t: defaultTranslate } = createTranslate('web', {
  scope: 'web/pages/FieldDefaultList',
});

export type FieldDefaultListPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createModelStore;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * List page bindings: resolve application, title, and FieldDefault store.
 */
export function useFieldDefaultListPage(deps: FieldDefaultListPageDeps): {
  application: Ref<string>;
  pageTitle: Ref<string>;
  fieldDefaultStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;

  const application = computed(() => resolveFieldDefaultListApplication(deps.route));
  const pageTitle = computed(() => {
    const appLabel = application.value || '—';
    if (deps.translate) {
      return deps.translate('Field Defaults: %s', appLabel);
    }
    return String(defaultTranslate('Field Defaults: %s', appLabel));
  });
  const fieldDefaultStore = shallowRef<ModelStore<any> | null>(null);

  function bindStore(app: string) {
    if (!app) {
      fieldDefaultStore.value = null;
      return;
    }
    fieldDefaultStore.value = createStore(`${app}.FieldDefault`, {
      storeId: `FieldDefault_${app}`,
      scopeManager: deps.scopeManager,
    }) as ModelStore<any>;
  }

  watch(application, (app) => bindStore(app), { immediate: true });

  return { application, pageTitle, fieldDefaultStore };
}
