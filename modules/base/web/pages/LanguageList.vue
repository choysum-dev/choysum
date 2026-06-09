<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <LanguageListView :store="languageStore" createAction="/base/languages/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import LanguageListView from '../views/LanguageListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Language from '@/base/service/models/language';

defineOptions({ name: 'LanguageListPage' });

const route = useRoute();
const languageStore = createStoreByModel<typeof Language>('base.Language', {
  storeId: `Language_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
