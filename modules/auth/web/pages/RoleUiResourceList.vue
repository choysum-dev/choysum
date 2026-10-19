<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="uiResourceGrantStore"
    :action-import="true"
    :action-export="true"
  >
    <RoleUiResourceListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import RoleUiResourceListView from '@/auth/web/views/RoleUiResourceListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/RoleUiResourceList' });
const pageTitle = _t('UI Resource Grant List');

const route = useRoute();
const uiResourceGrantStore = createStoreByModel<typeof RoleUiResource>('auth.RoleUiResource', {
  storeId: `RoleUiResource_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
