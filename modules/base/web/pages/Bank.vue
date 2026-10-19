<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="bankStore">
    <BankFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import BankFormView from '../views/BankFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Bank from '@/base/service/models/bank';
import { ChoyPage } from '@/web';

defineOptions({ name: 'BankPage' });

const route = useRoute();
const bankStore = createStoreByModel<typeof Bank>('base.Bank', {
  storeId: `Bank_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
