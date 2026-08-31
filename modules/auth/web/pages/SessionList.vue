<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="sessionStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns IpAddress, Status.')"
  >
    <SessionListView selection-mode="multiple" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SessionListView from '@/auth/web/views/SessionListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Session from '@/auth/service/models/session';

const { _t } = createTranslate('auth', { scope: 'web/pages/SessionList' });
const pageTitle = _t('Session List');

const route = useRoute();
const sessionStore = createStoreByModel<typeof Session>('auth.Session', {
  storeId: `Session_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
