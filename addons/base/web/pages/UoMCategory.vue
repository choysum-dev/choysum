<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <UoMCategoryFormView :key="$route.fullPath" createAction="/base/uom-categories/new" :store="uomCategoryStore" :record-id="recordId" :view-mode="viewMode" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import UoMCategoryFormView from '../views/UoMCategoryFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type UoMCategory from '@/base/service/models/uom_category';

defineOptions({ name: 'UoMCategoryPage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const uomCategoryStore = createStoreByModel<typeof UoMCategory>('base.UoMCategory', {
  storeId: `UoMCategory_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
