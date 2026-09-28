<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="tokenStore">
    <TokenKanbanView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { ChoyPage } from '@/web';
import TokenKanbanView from '../views/TokenKanbanView.vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Token from '@/auth/service/models/token';

const { _t } = createTranslate('auth', { scope: 'web/pages/TokenKanban' });
const pageTitle = _t('Token Kanban');

// Reuse the list store id so kanban and list views share cache and filters.
const tokenStore = createStoreByModel<typeof Token>('auth.Token', {
  storeId: 'Token_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
