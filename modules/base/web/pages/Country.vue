<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="countryStore">
    <CountryFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import CountryFormView from '../views/CountryFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Country from '@/base/service/models/country';
import { ChoyPage } from '@/web';

defineOptions({ name: 'CountryPage' });

const route = useRoute();
const countryStore = createStoreByModel<typeof Country>('base.Country', {
  storeId: `Country_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
