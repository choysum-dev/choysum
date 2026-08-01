<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ModuleListView :store="moduleStore" />
  </OPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ModuleListView from '../views/ModuleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type MetaModuleIndex from '@/meta/service/models/module_index';

defineOptions({ name: 'MetaModuleListTablePage' });

const moduleStore = createStoreByModel<typeof MetaModuleIndex>('meta.MetaModuleIndex', {
  storeId: 'MetaModuleIndex_ListTable',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
