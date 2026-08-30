<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="currencyStore">
    <CurrencyFormView :key="$route.fullPath" createAction="/base/currencies/new" :record-id="recordId" :view-mode="viewMode" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CurrencyFormView from '../views/CurrencyFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type Currency from '@/base/service/models/currency';

defineOptions({ name: 'CurrencyPage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const currencyStore = createStoreByModel<typeof Currency>('base.Currency', {
  storeId: `Currency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
