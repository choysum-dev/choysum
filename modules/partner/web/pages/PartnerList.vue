<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <div class="partner-list-page">
      <div class="partner-list-toolbar">
        <el-button type="primary" plain @click="importWizardOpen = true">{{ importLabel }}</el-button>
      </div>
      <PartnerListView ref="listViewRef" :store="partnerStore" createAction="/partner/partners/new" />
      <PartnerImportWizard v-model="importWizardOpen" :company-id="activeCompanyId" @imported="onImported" />
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
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { getCurrentRequestContext } from '@/core/rpc/context';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _lt } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const importLabel = _lt('Import CSV');

const route = useRoute();
const importWizardOpen = ref(false);
const listViewRef = ref<{ refresh?: () => Promise<void> | void } | null>(null);

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

<style scoped>
.partner-list-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.partner-list-toolbar {
  display: flex;
  justify-content: flex-end;
}
</style>
