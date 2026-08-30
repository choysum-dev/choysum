<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="uomCategoryStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, IsActive.')"
  >
    <UoMCategoryListView />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import UoMCategoryListView from '../views/UoMCategoryListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type UoMCategory from '@/base/service/models/uom_category';

defineOptions({ name: 'UoMCategoryListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/UoMCategoryList' });
const pageTitle = _t('Unit of Measure Category List');

const route = useRoute();
const uomCategoryStore = createStoreByModel<typeof UoMCategory>('base.UoMCategory', {
  storeId: `UoMCategory_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
