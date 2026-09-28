<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="indexStore"
  >
    <ModuleKanbanView :module-store="moduleStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import ModuleKanbanView from '../views/ModuleKanbanView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import type MetaModule from '@/meta/service/models/module';
import { ChoyPage } from '@/web';

defineOptions({ name: 'MetaModuleListPage' });

const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleList' });
const pageTitle = _t('Module List');

const indexStore = createStoreByModel<typeof MetaModuleIndex>('meta.MetaModuleIndex', {
  storeId: 'MetaModuleIndex_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});

const moduleStore = createStoreByModel<typeof MetaModule>('meta.MetaModule', {
  storeId: 'MetaModule_ListKanban_Action',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
