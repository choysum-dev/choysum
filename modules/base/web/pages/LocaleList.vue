<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <LocaleListView :store="localeStore" createAction="/base/locales/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import LocaleListView from '../views/LocaleListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Locale from '@/base/service/models/locale';

defineOptions({ name: 'LocaleListPage' });

const route = useRoute();
const localeStore = createStoreByModel<typeof Locale>('base.Locale', {
  storeId: `Locale_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
