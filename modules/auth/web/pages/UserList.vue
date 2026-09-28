<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="userStore"
    :action-import="true"
    :action-export="true"
  >
    <UserListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UserListView from '@/auth/web/views/UserListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type User from '@/auth/service/models/user/user';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/UserList' });
const pageTitle = _t('User List');

const route = useRoute();

const userStore = createStoreByModel<typeof User>('auth.User', {
  storeId: `User_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
