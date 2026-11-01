// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';

const { _t: defaultTranslate } = createTranslate('web', {
  scope: 'web/pages/ExportTemplateList',
});

export type ExportTemplateListPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * Settings → web Export Templates list: fixed web.ExportTemplate store (Owner B/web).
 * Primary template UX remains ExportPanel; this page is the sys.admin ledger.
 */
export function useExportTemplateListPage(deps: ExportTemplateListPageDeps): {
  pageTitle: Ref<string>;
  exportTemplateStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const pageTitle = computed(() => {
    if (deps.translate) return deps.translate('Export Templates');
    return String(defaultTranslate('Export Templates'));
  });
  const exportTemplateStore = shallowRef<WebModelStore<any> | null>(
    createStore('web.ExportTemplate', {
      storeId: 'ExportTemplate_web_admin',
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>
  );

  return { pageTitle, exportTemplateStore };
}
