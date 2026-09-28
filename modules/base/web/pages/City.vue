<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="cityStore">
    <CityFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import CityFormView from '../views/CityFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type City from '@/base/service/models/city';
import { ChoyPage } from '@/web';

defineOptions({ name: 'CityPage' });

const route = useRoute();
const cityStore = createStoreByModel<typeof City>('base.City', {
  storeId: `City_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
