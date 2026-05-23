<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <LocaleFormView :key="$route.fullPath" createAction="/base/locales/new" :store="localeStore" :record-id="recordId" :view-mode="viewMode" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import LocaleFormView from '../views/LocaleFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type Locale from '@/base/service/models/locale';

defineOptions({ name: 'LocalePage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const localeStore = createStoreByModel<typeof Locale>('base.Locale', {
  storeId: `Locale_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
