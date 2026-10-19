<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="languageStore">
    <LanguageFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import LanguageFormView from '../views/LanguageFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Language from '@/base/service/models/language';
import { ChoyPage } from '@/web';

defineOptions({ name: 'LanguagePage' });

const route = useRoute();
const languageStore = createStoreByModel<typeof Language>('base.Language', {
  storeId: `Language_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
