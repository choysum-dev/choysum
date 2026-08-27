<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <div class="partner-list-page">
      <div class="partner-list-toolbar">
        <el-button type="primary" plain @click="importWizardOpen = true">{{ importLabel }}</el-button>
        <el-button plain @click="exportPanelOpen = true">{{ exportLabel }}</el-button>
      </div>
      <PartnerListView ref="listViewRef" :store="partnerStore" createAction="/partner/partners/new" />
      <PartnerImportWizard v-model="importWizardOpen" :company-id="activeCompanyId" @imported="onImported" />
      <ExportPanel
        v-model="exportPanelOpen"
        model="partner.Partner"
        :company-id="activeCompanyId"
        :ids="exportIds"
        :domain="exportDomain"
        :default-fields="exportDefaultFields"
        :filtered-count="filteredCount"
      />
    </div>
  </OPage>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import PartnerListView from '../views/PartnerListView.vue';
import PartnerImportWizard from '../components/PartnerImportWizard.vue';
import { ExportPanel } from '@/core/web/export';
import { normalizeExportFieldPaths } from '@/core/web/export/field_paths';
import { buildUnifiedQuery } from '@/web/web/query/context';
import { exportFieldSelection } from '@/web/web/query/utils/registry/field';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { getCurrentRequestContext } from '@/core/rpc/context';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _t } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const importLabel = _t('Import CSV');
const exportLabel = _t('Export CSV');

const route = useRoute();
const importWizardOpen = ref(false);
const exportPanelOpen = ref(false);
const listViewRef = ref<{ refresh?: () => Promise<void> | void; selectedItems?: { value?: Partner[] } | Partner[] } | null>(null);

function onImported() {
  void listViewRef.value?.refresh?.();
}

const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

const activeCompanyId = computed(() => {
  const ctx = getCurrentRequestContext();
  return String(ctx?.activeCompanyId ?? ctx?.companyId ?? '').trim();
});

const exportIds = computed(() => {
  const raw = listViewRef.value?.selectedItems;
  if (Array.isArray(raw)) {
    return raw.map(row => String(row?.Id ?? '').trim()).filter(Boolean);
  }
  const items = raw?.value ?? [];
  return items.map(row => String(row?.Id ?? '').trim()).filter(Boolean);
});

const exportDomain = computed(() => {
  const ctx = buildUnifiedQuery(partnerStore, { execOptions: { skipPagination: true, skipCount: true } });
  return JSON.stringify(ctx.filters ?? { And: [] });
});

const exportDefaultFields = computed(() => {
  const paths = exportFieldSelection(partnerStore.storeId) ?? [];
  return normalizeExportFieldPaths(paths.filter(path => path !== 'Id'));
});

const filteredCount = computed(() => Number((partnerStore.state as { result?: { total?: number } }).result?.total ?? 0));
</script>

<style scoped>
.partner-list-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.partner-list-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
