<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="uiResourceGrantStore">
    <RoleUiResourceFormView
      :key="$route.fullPath"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import RoleUiResourceFormView from '@/auth/web/views/RoleUiResourceFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';

const route = useRoute();
const uiResourceGrantStore = createStoreByModel<typeof RoleUiResource>('auth.RoleUiResource', {
  storeId: `RoleUiResource_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
