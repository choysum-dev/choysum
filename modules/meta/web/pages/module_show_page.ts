// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, watch, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';

export type ModuleShowPageDeps = {
  scopeManager: unknown;
  moduleName: Ref<string> | (() => string);
  createStore?: typeof createStoreByModel;
};

/**
 * Resolve `/meta/modules/:name` to a catalog row Id for the read-only Form.
 */
export function useModuleShowPage(deps: ModuleShowPageDeps): {
  catalogStore: WebModelStore<MetaModuleCatalog>;
  recordId: Ref<string>;
  loading: Ref<boolean>;
  missing: Ref<boolean>;
  moduleName: Ref<string>;
} {
  const createStore = deps.createStore ?? createStoreByModel;
  const moduleName = computed(() => {
    const raw = typeof deps.moduleName === 'function' ? deps.moduleName() : deps.moduleName.value;
    return String(raw || '').trim();
  });
  const catalogStore = createStore('meta.MetaModuleCatalog', {
    storeId: `MetaModuleCatalog_Show_${moduleName.value || 'unknown'}`,
    scopeManager: deps.scopeManager,
  }) as WebModelStore<MetaModuleCatalog>;
  const recordId = ref('');
  const loading = ref(false);
  const missing = ref(false);

  async function resolveRecord() {
    const name = moduleName.value;
    recordId.value = '';
    missing.value = false;
    if (!name) {
      missing.value = true;
      return;
    }
    loading.value = true;
    try {
      const rows = (await (catalogStore as any).Search(['ModuleName', '=', name], {
        limit: 1,
        fields: ['Id', 'ModuleName'],
      })) as Array<{ Id?: string }> | null;
      const id = String(rows?.[0]?.Id || '').trim();
      if (!id) {
        missing.value = true;
        return;
      }
      recordId.value = id;
    } catch {
      missing.value = true;
    } finally {
      loading.value = false;
    }
  }

  watch(moduleName, () => {
    void resolveRecord();
  }, { immediate: true });

  return { catalogStore, recordId, loading, missing, moduleName };
}
