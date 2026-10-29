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
import { computed, shallowRef, watch } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import TranslationTermFormView from '../views/TranslationTermFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'TranslationTermPage' });

const route = useRoute();
const scopeManager = useScopeManager().menuScopeManager;
const application = computed(() => String(route.params.application || '').trim());
const termStore = shallowRef<WebModelStore<any> | null>(null);

watch(
  application,
  (app) => {
    if (!app) {
      termStore.value = null;
      return;
    }
    termStore.value = createStoreByModel(`${app}.TranslationTerm`, {
      storeId: `TranslationTerm_${app}_${route.fullPath}`,
      scopeManager,
    });
  },
  { immediate: true }
);
</script>
