<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="recordRuleStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Kind, PermRead, PermWrite, PermCreate, PermDelete.')"
  >
    <RoleRecordRuleListView selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleRecordRuleListView from '@/auth/web/views/RoleRecordRuleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleRecordRuleList' });
const pageTitle = _t('Record Rule List');

const route = useRoute();
const recordRuleStore = createStoreByModel<typeof RoleRecordRule>('auth.RoleRecordRule', {
  storeId: `RoleRecordRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
