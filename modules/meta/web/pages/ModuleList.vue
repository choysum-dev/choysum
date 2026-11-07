<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="catalogStore"
  >
    <ModuleKanbanView :module-store="operationStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import ModuleKanbanView from '../views/ModuleKanbanView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { ChoyPage } from '@/web';

defineOptions({ name: 'MetaModuleListPage' });

const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleList' });
const pageTitle = _t('Module Board');

const catalogStore = createStoreByModel<typeof MetaModuleCatalog>('meta.MetaModuleCatalog', {
  storeId: 'MetaModuleCatalog_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});

const operationStore = createStoreByModel<typeof MetaModuleOperation>('meta.MetaModuleOperation', {
  storeId: 'MetaModuleOperation_ListKanban_Action',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
