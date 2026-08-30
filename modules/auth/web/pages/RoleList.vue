<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="roleStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, Description, IsActive, IsSystem.')"
  >
    <RoleListView createAction="/auth/roles/new" selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Role from '@/auth/service/models/role';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleList' });
const pageTitle = _t('Role List');

const route = useRoute();
const roleStore = createStoreByModel<typeof Role>('auth.Role', {
  storeId: `Role_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
