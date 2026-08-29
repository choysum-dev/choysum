<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :title="pageTitle" :store="partnerStore">
    <template #title-actions>
      <OPageIoMenu
        import
        export
        :import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, IsActive, CustomerRank, SupplierRank.')"
        :list-ref="listViewRef"
      />
    </template>
    <PartnerListView ref="listViewRef" createAction="/partner/partners/new" />
  </OPage>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import OPageIoMenu from '@/web/web/components/page/OPageIoMenu.vue';
import PartnerListView from '../views/PartnerListView.vue';
import type { RecordExportListRef } from '@/web/web/composables/useRecordExportScope';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _t } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const pageTitle = _t('Partner List');

const route = useRoute();
const listViewRef = ref<(RecordExportListRef & { refresh?: () => Promise<void> | void }) | null>(null);

const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
