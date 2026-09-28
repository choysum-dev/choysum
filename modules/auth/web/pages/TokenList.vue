<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="tokenStore">
    <TokenListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import TokenListView from '@/auth/web/views/TokenListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Token from '@/auth/service/models/token';
import { ChoyPage } from '@/web';

const { _t } = createTranslate('auth', { scope: 'web/pages/TokenList' });
const pageTitle = _t('Token List');

// Reuse the shared list/kanban store id so both views keep one cache.
const tokenStore = createStoreByModel<typeof Token>('auth.Token', {
  storeId: 'Token_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
