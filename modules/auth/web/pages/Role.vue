<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="roleStore">
    <RoleFormView :key="$route.fullPath" createAction="/auth/roles/new" :record-id="recordId" :view-mode="viewMode" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleFormView from '@/auth/web/views/RoleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type Role from '@/auth/service/models/role';

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const roleStore = createStoreByModel<typeof Role>('auth.Role', {
  storeId: `Role_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
