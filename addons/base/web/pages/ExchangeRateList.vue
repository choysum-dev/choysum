<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ExchangeRateListView :store="exchangeRateStore" createAction="/base/exchange-rates/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ExchangeRateListView from '../views/ExchangeRateListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type ExchangeRate from '@/base/service/models/exchange_rate';

defineOptions({ name: 'ExchangeRateListPage' });

const route = useRoute();
const exchangeRateStore = createStoreByModel<typeof ExchangeRate>('base.ExchangeRate', {
  storeId: `ExchangeRate_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
