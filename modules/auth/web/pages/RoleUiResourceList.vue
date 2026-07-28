<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleUiResourceListView :store="uiResourceGrantStore" createAction="/auth/ui-resource-grants/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleUiResourceListView from '@/auth/web/views/RoleUiResourceListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';

const route = useRoute();
const uiResourceGrantStore = createStoreByModel<typeof RoleUiResource>('auth.RoleUiResource', {
  storeId: `RoleUiResource_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
