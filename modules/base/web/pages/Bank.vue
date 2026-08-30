<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="bankStore">
    <BankFormView :key="$route.fullPath" createAction="/base/banks/new" :record-id="recordId" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import BankFormView from '../views/BankFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Bank from '@/base/service/models/bank';

defineOptions({ name: 'BankPage' });

withDefaults(
  defineProps<{
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const bankStore = createStoreByModel<typeof Bank>('base.Bank', {
  storeId: `Bank_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
