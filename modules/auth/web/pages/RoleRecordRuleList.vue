<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleRecordRuleListView :store="recordRuleStore" createAction="/auth/record-rules/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleRecordRuleListView from '@/auth/web/views/RoleRecordRuleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';

const route = useRoute();
const recordRuleStore = createStoreByModel<typeof RoleRecordRule>('auth.RoleRecordRule', {
  storeId: `RoleRecordRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
