<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <RoleFieldRuleFormView
      :key="$route.fullPath"
      createAction="/auth/field-rules/new"
      :store="fieldRuleStore"
      :record-id="recordId"
      :view-mode="viewMode"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleFieldRuleFormView from '@/auth/web/views/RoleFieldRuleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string | undefined;
  }>(),
  {}
);

const route = useRoute();
const fieldRuleStore = createStoreByModel<typeof RoleFieldRule>('auth.RoleFieldRule', {
  storeId: `RoleFieldRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
