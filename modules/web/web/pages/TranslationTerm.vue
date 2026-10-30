<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="termStore" :store="termStore">
    <TranslationTermFormView :key="application" :application="application" :store="termStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import TranslationTermFormView from '../views/TranslationTermFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useTerminologyDetailPage } from './terminology_detail_page';

defineOptions({ name: 'TranslationTermPage' });

const { application, termStore } = useTerminologyDetailPage({
  route: useRoute(),
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
