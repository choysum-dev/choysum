<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="sessionStore"
    :action-import="true"
    :action-export="true"
  >
    <SessionListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SessionListView from '@/auth/web/views/SessionListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Session from '@/auth/service/models/session';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/SessionList' });
const pageTitle = _t('Session List');

const route = useRoute();
const sessionStore = createStoreByModel<typeof Session>('auth.Session', {
  storeId: `Session_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
