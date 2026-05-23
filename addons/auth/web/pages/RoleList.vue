<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleListView :store="roleStore" createAction="/auth/roles/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Role from '@/auth/service/models/role';

const route = useRoute();
const roleStore = createStoreByModel<typeof Role>('auth.Role', {
  storeId: `Role_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
