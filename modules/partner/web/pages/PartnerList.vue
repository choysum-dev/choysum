<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :title="pageTitle">
    <template #title-actions>
      <OPageIoMenu :items="ioMenuItems" />
    </template>
    <PartnerListView ref="listViewRef" :store="partnerStore" createAction="/partner/partners/new" />
    <RecordImportShell
      v-model:open="importOpen"
      :model="ioConfig.model"
      :company-id="activeCompanyId"
      :upload-hint="ioConfig.import?.uploadHint"
      @imported="onImported"
    />
    <RecordExportShell
      v-model:open="exportOpen"
      :model="ioConfig.model"
      :store="partnerStore"
      :list-ref="listViewRef"
      :company-id="activeCompanyId"
    />
  </OPage>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import OPageIoMenu from '@/web/web/components/page/OPageIoMenu.vue';
import PartnerListView from '../views/PartnerListView.vue';
import { RecordImportShell } from '@/web/web/import';
import { RecordExportShell } from '@/web/web/export';
import { useRecordIoMenu } from '@/web/web/composables/useRecordIoMenu';
import type { RecordIoConfig } from '@/web/web/composables/recordIoTypes';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { getCurrentRequestContext } from '@/core/rpc/context';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _t } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const pageTitle = _t('Partner List');

const route = useRoute();
const importOpen = ref(false);
const exportOpen = ref(false);
const listViewRef = ref<{ refresh?: () => Promise<void> | void; selectedItems?: { value?: Partner[] } | Partner[] } | null>(null);

const ioConfig: RecordIoConfig = {
  model: 'partner.Partner',
  import: {
    enabled: true,
    uploadHint: _t('Upload a UTF-8 CSV with columns Name, Code, IsActive, CustomerRank, SupplierRank.'),
  },
  export: { enabled: true },
};

const { items: ioMenuItems } = useRecordIoMenu({
  config: ioConfig,
  openImport: () => {
    importOpen.value = true;
  },
  openExport: () => {
    exportOpen.value = true;
  },
});

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
</script>
