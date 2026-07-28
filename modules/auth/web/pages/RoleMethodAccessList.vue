<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleMethodAccessListView :store="methodAccessStore" createAction="/auth/method-accesses/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleMethodAccessListView from '@/auth/web/views/RoleMethodAccessListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';

const route = useRoute();
const methodAccessStore = createStoreByModel<typeof RoleMethodAccess>('auth.RoleMethodAccess', {
  storeId: `RoleMethodAccess_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
