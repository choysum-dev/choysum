<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="userFilterStore" :store="userFilterStore">
    <UserFilterFormView
      :store="userFilterStore"
      :initial-values="initialValues"
      create-action="/web/user-filters/new"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import UserFilterFormView from '../views/UserFilterFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useUserFilterDetailPage } from './user_filter_detail_page';

defineOptions({ name: 'UserFilterPage' });

const { userFilterStore } = useUserFilterDetailPage({
  scopeManager: useScopeManager().menuScopeManager,
});

// Technical admin create defaults: no default-favorite; empty condition/groupby.
const initialValues: Record<string, unknown> = {
  Name: null,
  ScopeKey: '',
  Application: null,
  ModelName: null,
  Condition: {},
  GroupBy: [],
  UserId: null,
  IsDefault: false,
};
</script>
