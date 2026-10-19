<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="bankStore"
    :action-import="true"
    :action-export="true"
  >
    <BankListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import BankListView from '../views/BankListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Bank from '@/base/service/models/bank';
import { ChoyPage } from '@/web';

defineOptions({ name: 'BankListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/BankList' });
const pageTitle = _t('Bank List');

const route = useRoute();
const bankStore = createStoreByModel<typeof Bank>('base.Bank', {
  storeId: `Bank_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
