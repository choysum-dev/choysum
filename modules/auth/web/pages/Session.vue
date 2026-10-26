<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="sessionStore">
    <SessionFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SessionFormView from '@/auth/web/views/SessionFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Session from '@/auth/service/models/session';
import { ChoyPage } from '@/web';

const route = useRoute();
const sessionStore = createStoreByModel<typeof Session>('auth.Session', {
  storeId: `Session_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
