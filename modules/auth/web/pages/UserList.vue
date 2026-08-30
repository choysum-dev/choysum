<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="userStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Username, Email, Phone, FullName.')"
  >
    <UserListView createAction="/auth/users/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UserListView from '@/auth/web/views/UserListView.vue';
import OPage from '@/web/web/components/page/OPage.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type User from '@/auth/service/models/user';

const { _t } = createTranslate('auth', { scope: 'web/pages/UserList' });
const pageTitle = _t('User List');

const route = useRoute();

const userStore = createStoreByModel<typeof User>('auth.User', {
  storeId: `User_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
