<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="languageStore"
    :action-import="true"
    :action-export="true"
  >
    <LanguageListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import LanguageListView from '../views/LanguageListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Language from '@/base/service/models/language';
import { ChoyPage } from '@/web';

defineOptions({ name: 'LanguageListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/LanguageList' });
const pageTitle = _t('Language List');

const route = useRoute();
const languageStore = createStoreByModel<typeof Language>('base.Language', {
  storeId: `Language_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
