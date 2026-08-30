<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="uomCategoryStore">
    <UoMCategoryFormView :key="$route.fullPath" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import UoMCategoryFormView from '../views/UoMCategoryFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type UoMCategory from '@/base/service/models/uom_category';

defineOptions({ name: 'UoMCategoryPage' });

const route = useRoute();
const uomCategoryStore = createStoreByModel<typeof UoMCategory>('base.UoMCategory', {
  storeId: `UoMCategory_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
