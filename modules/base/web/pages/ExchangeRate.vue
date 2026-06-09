<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ExchangeRateFormView
      :key="$route.fullPath"
      createAction="/base/exchange-rates/new"
      :store="exchangeRateStore"
      :record-id="recordId"
      :view-mode="viewMode"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ExchangeRateFormView from '../views/ExchangeRateFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type ExchangeRate from '@/base/service/models/exchange_rate';

defineOptions({ name: 'ExchangeRatePage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const exchangeRateStore = createStoreByModel<typeof ExchangeRate>('base.ExchangeRate', {
  storeId: `ExchangeRate_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
