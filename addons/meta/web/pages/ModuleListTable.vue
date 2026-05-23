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
import type IrModuleIndex from '@/meta/service/models/ir_module_index';

defineOptions({ name: 'MetaModuleListTablePage' });

const moduleStore = createStoreByModel<typeof IrModuleIndex>('meta.IrModuleIndex', {
  storeId: 'IrModuleIndex_ListTable',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
