<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ModuleDetailView :key="$route.fullPath" :store="moduleStore" :record-id="recordId" :view-mode="viewMode" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ModuleDetailView from '../views/ModuleDetailView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type IrModuleIndex from '@/meta/service/models/ir_module_index';

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const moduleStore = createStoreByModel<typeof IrModuleIndex>('meta.IrModuleIndex', {
  storeId: `IrModuleIndex_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
