<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <TokenKanbanView :store="tokenStore" create-action="/auth/tokens/new" />
  </OPage>
</template>

<script setup lang="ts">
import OPage from '@/web/web/components/page/OPage.vue';
import TokenKanbanView from '../views/TokenKanbanView.vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Token from '@/auth/service/models/token';

// Reuse the list store id so kanban and list views share cache and filters.
const tokenStore = createStoreByModel<typeof Token>('auth.Token', {
  storeId: 'Token_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
