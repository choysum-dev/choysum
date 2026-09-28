<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="countryStore"
    :action-import="true"
    :action-export="true"
  >
    <CountryListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import CountryListView from '../views/CountryListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Country from '@/base/service/models/country';
import { ChoyPage } from '@/web';

defineOptions({ name: 'CountryListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CountryList' });
const pageTitle = _t('Country List');

const route = useRoute();
const countryStore = createStoreByModel<typeof Country>('base.Country', {
  storeId: `Country_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
