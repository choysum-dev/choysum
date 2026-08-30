<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="currencyStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, Symbol, DecimalDigits, Rounding, IsActive.')"
  >
    <CurrencyListView />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CurrencyListView from '../views/CurrencyListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Currency from '@/base/service/models/currency';

defineOptions({ name: 'CurrencyListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CurrencyList' });
const pageTitle = _t('Currency List');

const route = useRoute();
const currencyStore = createStoreByModel<typeof Currency>('base.Currency', {
  storeId: `Currency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
