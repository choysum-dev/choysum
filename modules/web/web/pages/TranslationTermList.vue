<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="termStore" :title="pageTitle" :store="termStore">
    <TranslationTermListView :key="application" :application="application" :store="termStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import TranslationTermListView from '../views/TranslationTermListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useTerminologyListPage } from './terminology_list_page';

defineOptions({ name: 'TranslationTermListPage' });

const { application, pageTitle, termStore } = useTerminologyListPage({
  route: useRoute(),
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
