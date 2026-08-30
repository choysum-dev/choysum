<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="fieldRuleStore">
    <RoleFieldRuleFormView
      :key="$route.fullPath"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleFieldRuleFormView from '@/auth/web/views/RoleFieldRuleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';

const route = useRoute();
const fieldRuleStore = createStoreByModel<typeof RoleFieldRule>('auth.RoleFieldRule', {
  storeId: `RoleFieldRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
