<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="exchangeRateStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Date, Rate.')"
  >
    <ExchangeRateListView createAction="/base/exchange-rates/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ExchangeRateListView from '../views/ExchangeRateListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type ExchangeRate from '@/base/service/models/exchange_rate';

defineOptions({ name: 'ExchangeRateListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/ExchangeRateList' });
const pageTitle = _t('Exchange Rate List');

const route = useRoute();
const exchangeRateStore = createStoreByModel<typeof ExchangeRate>('base.ExchangeRate', {
  storeId: `ExchangeRate_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
