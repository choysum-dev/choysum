<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="fieldRuleStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns LogicalModelName, PermRead, PermWrite.')"
  >
    <RoleFieldRuleListView selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleFieldRuleListView from '@/auth/web/views/RoleFieldRuleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleFieldRuleList' });
const pageTitle = _t('Field Rule List');

const route = useRoute();
const fieldRuleStore = createStoreByModel<typeof RoleFieldRule>('auth.RoleFieldRule', {
  storeId: `RoleFieldRule_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
