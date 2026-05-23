<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <CityListView :store="cityStore" createAction="/base/cities/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CityListView from '../views/CityListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type City from '@/base/service/models/city';

defineOptions({ name: 'CityListPage' });

const route = useRoute();
const cityStore = createStoreByModel<typeof City>('base.City', {
  storeId: `City_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
