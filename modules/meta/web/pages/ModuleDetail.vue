<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="moduleStore">
    <ModuleDetailView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import ModuleDetailView from '../views/ModuleDetailView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import { ChoyPage } from '@/web';

const route = useRoute();
const moduleStore = createStoreByModel<typeof MetaModuleIndex>('meta.MetaModuleIndex', {
  storeId: `MetaModuleIndex_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
