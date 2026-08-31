<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="cityStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, IsActive.')"
  >
    <CityListView />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CityListView from '../views/CityListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type City from '@/base/service/models/city';

defineOptions({ name: 'CityListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CityList' });
const pageTitle = _t('City List');

const route = useRoute();
const cityStore = createStoreByModel<typeof City>('base.City', {
  storeId: `City_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
