<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <ModuleLogListView :store="logStore" />
  </OPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import ModuleLogListView from '../views/ModuleLogListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type ModuleManagementLog from '@/meta/service/models/module_management_log';

defineOptions({ name: 'MetaModuleHistoryPage' });

const logStore = createStoreByModel<typeof ModuleManagementLog>('meta.ModuleManagementLog', {
  storeId: 'ModuleManagementLog_List',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
