<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleUiResourceFormView
      :key="$route.fullPath"
      createAction="/auth/ui-resource-grants/new"
      :store="uiResourceGrantStore"
      :record-id="recordId"
      :view-mode="viewMode"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleUiResourceFormView from '@/auth/web/views/RoleUiResourceFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const uiResourceGrantStore = createStoreByModel<typeof RoleUiResource>('auth.RoleUiResource', {
  storeId: `RoleUiResource_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
