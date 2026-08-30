<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="methodAccessStore">
    <RoleMethodAccessFormView
      :key="$route.fullPath"
      createAction="/auth/method-accesses/new"
      :record-id="recordId"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleMethodAccessFormView from '@/auth/web/views/RoleMethodAccessFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';

const props = withDefaults(
  defineProps<{
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const methodAccessStore = createStoreByModel<typeof RoleMethodAccess>('auth.RoleMethodAccess', {
  storeId: `RoleMethodAccess_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
