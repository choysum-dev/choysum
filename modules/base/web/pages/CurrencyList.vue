<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="currencyStore"
    :action-import="true"
    :action-export="true"
  >
    <CurrencyListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import CurrencyListView from '../views/CurrencyListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Currency from '@/base/service/models/currency';
import { ChoyPage } from '@/web';

defineOptions({ name: 'CurrencyListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CurrencyList' });
const pageTitle = _t('Currency List');

const route = useRoute();
const currencyStore = createStoreByModel<typeof Currency>('base.Currency', {
  storeId: `Currency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
