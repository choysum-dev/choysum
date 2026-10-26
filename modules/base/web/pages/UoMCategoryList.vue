<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="uomCategoryStore"
    :action-import="true"
    :action-export="true"
  >
    <UoMCategoryListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UoMCategoryListView from '../views/UoMCategoryListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type UoMCategory from '@/base/service/models/uom_category';
import { ChoyPage } from '@/web';

defineOptions({ name: 'UoMCategoryListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/UoMCategoryList' });
const pageTitle = _t('Unit of Measure Category List');

const route = useRoute();
const uomCategoryStore = createStoreByModel<typeof UoMCategory>('base.UoMCategory', {
  storeId: `UoMCategory_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
