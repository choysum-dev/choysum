// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, shallowRef, type Ref } from 'vue';
import { createModelStore } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { ModelStore } from '@/web/web/stores/modelStore';

const { _t: defaultTranslate } = createTranslate('meta', {
  scope: 'web/pages/MetaModelDataList',
});

export type MetaModelDataListPageDeps = {
  scopeManager: unknown;
  createStore?: typeof createModelStore;
  translate?: (msgid: string, ...args: unknown[]) => string;
};

/**
 * Settings → meta External IDs list: fixed meta.MetaModelData store (Owner B/meta).
 * Read-only ledger for xml_id mappings; loaders still own writes.
 */
export function useMetaModelDataListPage(deps: MetaModelDataListPageDeps): {
  pageTitle: Ref<string>;
  metaModelDataStore: Ref<ModelStore<any> | null>;
} {
  const createStore = deps.createStore ?? createModelStore;
  const pageTitle = computed(() => {
    if (deps.translate) return deps.translate('External IDs');
    return String(defaultTranslate('External IDs'));
  });
  const metaModelDataStore = shallowRef<ModelStore<any> | null>(
    createStore('meta.MetaModelData', {
      storeId: 'MetaModelData_meta_admin',
      scopeManager: deps.scopeManager,
    }) as ModelStore<any>
  );

  return { pageTitle, metaModelDataStore };
}
