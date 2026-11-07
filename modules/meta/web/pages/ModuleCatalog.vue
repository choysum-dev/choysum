<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="catalogStore">
    <ModuleCatalogView :store="catalogStore" :initial-operation-id="initialOperationId" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import ModuleCatalogView from '../views/ModuleCatalogView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useModuleCatalogPage } from './module_catalog_page';

defineOptions({ name: 'MetaModuleCatalogPage' });

const route = useRoute();
const { pageTitle, catalogStore } = useModuleCatalogPage({
  scopeManager: useScopeManager().menuScopeManager,
});

const initialOperationId = computed(() => {
  if (String(route.name || '') !== 'MetaModuleOperationDeepLink') return '';
  return String(route.params.id || '').trim();
});
</script>
