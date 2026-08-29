<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :title="pageTitle">
    <template #title-actions>
      <OPageIoMenu
        :config="ioConfig"
        :store="partnerStore"
        :list-ref="listViewRef"
      />
    </template>
    <PartnerListView ref="listViewRef" :store="partnerStore" createAction="/partner/partners/new" />
  </OPage>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import OPageIoMenu from '@/web/web/components/page/OPageIoMenu.vue';
import PartnerListView from '../views/PartnerListView.vue';
import type { RecordIoConfig } from '@/web/web/composables/recordIoTypes';
import type { RecordExportListRef } from '@/web/web/composables/useRecordExportScope';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _t } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const pageTitle = _t('Partner List');

const route = useRoute();
const listViewRef = ref<(RecordExportListRef & { refresh?: () => Promise<void> | void }) | null>(null);

const ioConfig: RecordIoConfig = {
  model: 'partner.Partner',
  import: {
    enabled: true,
    uploadHint: _t('Upload a UTF-8 CSV with columns Name, Code, IsActive, CustomerRank, SupplierRank.'),
  },
  export: { enabled: true },
};

const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
