// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shallowRef, type Ref } from 'vue';
import { createModelStore } from '@/web/web/stores/registry';
import type { ModelStore } from '@/web/web/stores/modelStore';

export type ExportTemplateDetailPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createModelStore;
};

/**
 * Settings → web Export Templates create/detail: fixed web.ExportTemplate store.
 */
export function useExportTemplateDetailPage(deps: ExportTemplateDetailPageDeps): {
  exportTemplateStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const exportTemplateStore = shallowRef<ModelStore<any> | null>(
    createStore('web.ExportTemplate', {
      storeId: 'ExportTemplate_web_admin',
      scopeManager: deps.scopeManager,
    }) as ModelStore<any>
  );
  return { exportTemplateStore };
}
