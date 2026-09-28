<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="tokenStore">
    <TokenFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import TokenFormView from '@/auth/web/views/TokenFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Token from '@/auth/service/models/token';
import { ChoyPage } from '@/web';

const route = useRoute();
const tokenStore = createStoreByModel<typeof Token>('auth.Token', {
  storeId: `Token_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
