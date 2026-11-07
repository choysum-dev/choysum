<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="moduleStore"
  >
    <ModuleListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import ModuleListView from '../views/ModuleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import { ChoyPage } from '@/web';

defineOptions({ name: 'MetaModuleListTablePage' });

const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleListTable' });
const pageTitle = _t('Module List');

const moduleStore = createStoreByModel<typeof MetaModuleCatalog>('meta.MetaModuleCatalog', {
  storeId: 'MetaModuleCatalog_ListTable',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
