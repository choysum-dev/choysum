// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, watch, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { resolveTerminologyListApplication } from './terminology_page_bind';

const { _t: defaultTranslate } = createTranslate('web', { scope: 'web/pages/TranslationTermList' });

export type TerminologyListPageDeps = {
  route: RouteLocationNormalizedLoaded;
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * List page bindings: resolve application, title, and TranslationTerm store.
 */
export function useTerminologyListPage(deps: TerminologyListPageDeps): {
  application: Ref<string>;
  pageTitle: Ref<string>;
  termStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;

  const application = computed(() => resolveTerminologyListApplication(deps.route));
  const pageTitle = computed(() => {
    const appLabel = application.value || '—';
    if (deps.translate) {
      return deps.translate('Translations: %s', appLabel);
    }
    return String(defaultTranslate('Translations: %s', appLabel));
  });
  const termStore = shallowRef<WebModelStore<any> | null>(null);

  function bindStore(app: string) {
    if (!app) {
      termStore.value = null;
      return;
    }
    termStore.value = createStore(`${app}.TranslationTerm`, {
      storeId: `TranslationTerm_${app}`,
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>;
  }

  watch(application, (app) => bindStore(app), { immediate: true });

  return { application, pageTitle, termStore };
}
