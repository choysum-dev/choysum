// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';

export type ExportTemplateDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createStoreByModel;
};

/**
 * Settings → web Export Templates create/detail: fixed web.ExportTemplate store.
 */
export function useExportTemplateDetailPage(deps: ExportTemplateDetailPageDeps): {
  exportTemplateStore: Ref<WebModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const exportTemplateStore = shallowRef<WebModelStore<any> | null>(
    createStore('web.ExportTemplate', {
      storeId: 'ExportTemplate_web_admin',
      scopeManager: deps.scopeManager,
    }) as WebModelStore<any>
  );
  return { exportTemplateStore };
}
