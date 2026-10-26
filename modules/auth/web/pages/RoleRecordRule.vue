<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="recordRuleStore">
    <RoleRecordRuleFormView
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import RoleRecordRuleFormView from '@/auth/web/views/RoleRecordRuleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import { ChoyPage } from '@/web';

const route = useRoute();
const recordRuleStore = createStoreByModel<typeof RoleRecordRule>('auth.RoleRecordRule', {
  storeId: `RoleRecordRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
