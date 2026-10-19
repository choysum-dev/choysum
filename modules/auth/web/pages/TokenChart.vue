<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="tokenStore">
    <TokenChartView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { ChoyPage } from '@/web';
import TokenChartView from '../views/TokenChartView.vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Token from '@/auth/service/models/token';

const { _t } = createTranslate('auth', { scope: 'web/pages/TokenChart' });
const pageTitle = _t('Token Chart');

// Share the list/kanban store id so filters and grouping stay aligned across views.
const tokenStore = createStoreByModel<typeof Token>('auth.Token', {
  storeId: 'Token_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
