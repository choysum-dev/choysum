<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="methodAccessStore">
    <RoleMethodAccessFormView
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import RoleMethodAccessFormView from '@/auth/web/views/RoleMethodAccessFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import { ChoyPage } from '@/web';

const route = useRoute();
const methodAccessStore = createStoreByModel<typeof RoleMethodAccess>('auth.RoleMethodAccess', {
  storeId: `RoleMethodAccess_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
