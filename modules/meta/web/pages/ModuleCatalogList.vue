<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="catalogStore">
    <ModuleCatalogListView :store="catalogStore" :operation-store="operationStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import ModuleCatalogListView from '../views/ModuleCatalogListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useModuleCatalogPage } from './module_catalog_page';

defineOptions({ name: 'MetaModuleCatalogListPage' });

const { pageTitle, catalogStore, operationStore } = useModuleCatalogPage({
  scopeManager: useScopeManager().menuScopeManager,
  pageTitleSrc: 'Module List',
});
</script>
