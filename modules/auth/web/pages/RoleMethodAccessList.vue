<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="methodAccessStore"
    :action-import="true"
    :action-export="true"
  >
    <RoleMethodAccessListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import RoleMethodAccessListView from '@/auth/web/views/RoleMethodAccessListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleMethodAccessList' });
const pageTitle = _t('Method Access List');

const route = useRoute();
const methodAccessStore = createStoreByModel<typeof RoleMethodAccess>('auth.RoleMethodAccess', {
  storeId: `RoleMethodAccess_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
