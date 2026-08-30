<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="userStore">
    <UserFormView :key="$route.fullPath" createAction="/auth/users/new" :record-id="recordId" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import UserFormView from '@/auth/web/views/UserFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type User from '@/auth/service/models/user';

const props = withDefaults(
  defineProps<{
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const userStore = createStoreByModel<typeof User>('auth.User', {
  storeId: `User_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
