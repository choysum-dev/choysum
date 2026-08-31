<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="methodAccessStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns LogicalModelName, Mode, Source.')"
  >
    <RoleMethodAccessListView selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleMethodAccessListView from '@/auth/web/views/RoleMethodAccessListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleMethodAccessList' });
const pageTitle = _t('Method Access List');

const route = useRoute();
const methodAccessStore = createStoreByModel<typeof RoleMethodAccess>('auth.RoleMethodAccess', {
  storeId: `RoleMethodAccess_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
