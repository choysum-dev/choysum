<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <UserListView :store="userStore" createAction="/auth/users/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UserListView from '@/auth/web/views/UserListView.vue';
import OPage from '@/web/web/components/page/OPage.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type User from '@/auth/service/models/user';

const route = useRoute();

const userStore = createStoreByModel<typeof User>('auth.User', {
  storeId: `User_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
