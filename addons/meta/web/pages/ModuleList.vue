<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ModuleKanbanView :store="indexStore" :module-store="moduleStore" />
  </OPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ModuleKanbanView from '../views/ModuleKanbanView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type IrModuleIndex from '@/meta/service/models/ir_module_index';
import type IrModule from '@/meta/service/models/ir_module';

defineOptions({ name: 'MetaModuleListPage' });

const indexStore = createStoreByModel<typeof IrModuleIndex>('meta.IrModuleIndex', {
  storeId: 'IrModuleIndex_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});

const moduleStore = createStoreByModel<typeof IrModule>('meta.IrModule', {
  storeId: 'IrModule_ListKanban_Action',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
