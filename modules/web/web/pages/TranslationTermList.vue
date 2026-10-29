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
import { computed, shallowRef, watch } from 'vue';
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import TranslationTermListView from '../views/TranslationTermListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import { ChoyPage } from '@/web';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'TranslationTermListPage' });

const { _t } = createTranslate('web', { scope: 'web/pages/TranslationTermList' });
const route = useRoute();
const scopeManager = useScopeManager().menuScopeManager;

/** Concrete list routes use meta; detail-style params still accepted as fallback. */
const application = computed(() => {
  const fromMeta = String((route.meta as { terminologyApplication?: string })?.terminologyApplication || '').trim();
  if (fromMeta) return fromMeta;
  const fromParam = String(route.params.application || '').trim();
  if (fromParam) return fromParam;
  const m = String(route.path || '').match(/\/web\/terminology\/([^/]+)/);
  return String(m?.[1] || '').trim();
});
const pageTitle = computed(() => _t('Translations: %s', application.value || '—'));
const termStore = shallowRef<WebModelStore<any> | null>(null);

function bindStore(app: string) {
  if (!app) {
    termStore.value = null;
    return;
  }
  termStore.value = createStoreByModel(`${app}.TranslationTerm`, {
    storeId: `TranslationTerm_${app}_${route.fullPath}`,
    scopeManager,
  });
}

watch(application, (app) => bindStore(app), { immediate: true });
</script>
