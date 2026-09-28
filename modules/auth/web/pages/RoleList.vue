<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="roleStore"
    :action-import="true"
    :action-export="true"
  >
    <RoleListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import RoleListView from '@/auth/web/views/RoleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Role from '@/auth/service/models/role';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleList' });
const pageTitle = _t('Role List');

const route = useRoute();
const roleStore = createStoreByModel<typeof Role>('auth.Role', {
  storeId: `Role_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
