<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleRecordRuleFormView
      :key="$route.fullPath"
      createAction="/auth/record-rules/new"
      :store="recordRuleStore"
      :record-id="recordId"
      :view-mode="viewMode"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleRecordRuleFormView from '@/auth/web/views/RoleRecordRuleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const recordRuleStore = createStoreByModel<typeof RoleRecordRule>('auth.RoleRecordRule', {
  storeId: `RoleRecordRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
