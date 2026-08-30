<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="countryStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, PhonePrefix, ZipRequired, StateRequired, IsActive.')"
  >
    <CountryListView />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CountryListView from '../views/CountryListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Country from '@/base/service/models/country';

defineOptions({ name: 'CountryListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CountryList' });
const pageTitle = _t('Country List');

const route = useRoute();
const countryStore = createStoreByModel<typeof Country>('base.Country', {
  storeId: `Country_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
