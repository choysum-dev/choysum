<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="partnerStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, IsActive, CustomerRank, SupplierRank.')"
  >
    <PartnerListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { ChoyPage } from '@/web';
import PartnerListView from '../views/PartnerListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const { _t } = createTranslate('partner', { scope: 'web/pages/PartnerList' });
const pageTitle = _t('Partner List');

const route = useRoute();

const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
