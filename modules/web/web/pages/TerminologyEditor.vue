<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <div class="terminology-toolbar">
      <el-select
        v-model="selectedApp"
        filterable
        clearable
        :placeholder="_t('Application')"
        style="width: 220px"
        @change="onApplicationChange"
      >
        <el-option v-for="app in applications" :key="app" :label="app" :value="app" />
      </el-select>
      <el-input
        v-model="moduleFilter"
        clearable
        :placeholder="_t('Module (required for PO)')"
        style="width: 220px"
      />
      <el-button type="primary" :disabled="!canDownloadPo" :loading="downloading" @click="onDownloadPo">
        {{ _t('Download PO') }}
      </el-button>
    </div>

    <OListView
      v-if="termStore"
      :store="termStore"
      editable
      :searchView="OSearchView"
      :action-ids="{}"
    >
      <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
      <OVarCharField :store="termStore" prop="Module" :readonly="true" />
      <OVarCharField :store="termStore" prop="Lang" :readonly="true" />
      <OVarCharField :store="termStore" prop="Scope" :readonly="true" />
      <OTextField :store="termStore" prop="Src" :readonly="true" />
      <OTextField :store="termStore" prop="Value" />
      <OVarCharField :store="termStore" prop="Kind" :readonly="true" />
      <OVarCharField :store="termStore" prop="Source" :readonly="true" />
    </OListView>
    <el-empty v-else :description="_t('Select an application to edit terminology')" />
  </OPage>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue';
import { ElMessage } from 'element-plus';
import { useRoute } from 'vue-router';
import OPage from '@/web/web/components/page/OPage.vue';
import OListView from '@/web/web/components/view/OListView.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import { createStoreByModel, listRegisteredModelNames } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { downloadTerminologyPo } from '@/web/web/stores/i18nStore/po_download';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'TerminologyEditorPage' });

const { _t } = createTranslate('web', { scope: 'web/pages/TerminologyEditor' });
const route = useRoute();
const i18nStore = useI18nStore();
const authStore = useAuthStore();
const scopeManager = useScopeManager().menuScopeManager;

const applications = ref<string[]>([]);
const selectedApp = ref('');
const moduleFilter = ref('');
const downloading = ref(false);
const termStore = shallowRef<WebModelStore<any> | null>(null);

const canDownloadPo = computed(
  () => Boolean(selectedApp.value.trim() && moduleFilter.value.trim() && i18nStore.terminologyLang)
);

const translationTermSuffix = '.TranslationTerm';

function loadApplications() {
  const names = listRegisteredModelNames()
    .filter((modelName) => modelName.endsWith(translationTermSuffix))
    .map((modelName) => modelName.slice(0, -translationTermSuffix.length).trim())
    .filter((app) => app && app !== 'core');
  applications.value = [...new Set(names)].sort();
}

function wrapStoreForReload(store: WebModelStore<any>): WebModelStore<any> {
  if ((store.UpdateById as any).__reloadsTerminology) {
    return store;
  }
  const original = store.UpdateById.bind(store) as (
    id: string,
    vals: Record<string, unknown>,
    fields?: string[]
  ) => Promise<unknown>;
  const wrapped = (async (id: string, vals: Record<string, unknown>, fields?: string[]) => {
    const out = await original(id, vals, fields);
    try {
      await i18nStore.reloadTerminology();
    } catch {
      /* reload is best-effort; save already succeeded */
    }
    return out;
  }) as typeof store.UpdateById;
  (wrapped as any).__reloadsTerminology = true;
  store.UpdateById = wrapped;
  return store;
}

function onApplicationChange(appName?: string) {
  const app = String(appName ?? selectedApp.value).trim();
  selectedApp.value = app;
  termStore.value = null;
  moduleFilter.value = '';
  if (!app) return;
  try {
    const store = createStoreByModel(`${app}.TranslationTerm`, {
      storeId: `TerminologyEditor_${app}_${route.fullPath}`,
      scopeManager,
    });
    termStore.value = wrapStoreForReload(store);
  } catch (err: any) {
    ElMessage.error(err?.message || _t('TranslationTerm store is not available for this application'));
  }
}

async function onDownloadPo() {
  if (!canDownloadPo.value) return;
  downloading.value = true;
  try {
    const blob = await downloadTerminologyPo({
      lang: i18nStore.terminologyLang,
      application: selectedApp.value.trim(),
      module: moduleFilter.value.trim(),
      accessToken: authStore.tokens?.accessToken,
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${moduleFilter.value.trim()}-${i18nStore.terminologyLang}.po`;
    a.click();
    URL.revokeObjectURL(url);
  } catch (err: any) {
    ElMessage.error(err?.message || _t('PO download failed'));
  } finally {
    downloading.value = false;
  }
}

onMounted(() => {
  loadApplications();
});
</script>

<style scoped>
.terminology-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
}
</style>
