<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <CountryListView :store="countryStore" createAction="/base/countries/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CountryListView from '../views/CountryListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Country from '@/base/service/models/country';

defineOptions({ name: 'CountryListPage' });

const route = useRoute();
const countryStore = createStoreByModel<typeof Country>('base.Country', {
  storeId: `Country_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
