<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="userFilterStore" :title="pageTitle" :store="userFilterStore">
    <UserFilterListView :store="userFilterStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import UserFilterListView from '../views/UserFilterListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useUserFilterListPage } from './user_filter_list_page';

defineOptions({ name: 'UserFilterListPage' });

const { pageTitle, userFilterStore } = useUserFilterListPage({
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
