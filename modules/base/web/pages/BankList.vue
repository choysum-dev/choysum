<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="bankStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, BIC, IsActive.')"
  >
    <BankListView createAction="/base/banks/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import BankListView from '../views/BankListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Bank from '@/base/service/models/bank';

defineOptions({ name: 'BankListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/BankList' });
const pageTitle = _t('Bank List');

const route = useRoute();
const bankStore = createStoreByModel<typeof Bank>('base.Bank', {
  storeId: `Bank_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
