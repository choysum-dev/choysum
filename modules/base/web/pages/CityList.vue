<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="cityStore"
    :action-import="true"
    :action-export="true"
  >
    <CityListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import CityListView from '../views/CityListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type City from '@/base/service/models/city';
import { ChoyPage } from '@/web';

defineOptions({ name: 'CityListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CityList' });
const pageTitle = _t('City List');

const route = useRoute();
const cityStore = createStoreByModel<typeof City>('base.City', {
  storeId: `City_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
