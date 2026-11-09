<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="catalogStore">
    <p v-if="loading" class="p-4 text-sm text-muted-foreground">{{ _t('Loading…') }}</p>
    <p v-else-if="missing" class="p-4 text-sm text-muted-foreground" data-testid="module-catalog-missing">
      {{ _t('Module not found.') }}
    </p>
    <ModuleCatalogFormView
      v-else-if="recordId"
      :store="catalogStore"
      :operation-store="operationStore"
      :record-id="recordId"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import ModuleCatalogFormView from '../views/ModuleCatalogFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import { ChoyPage } from '@/web';
import { useModuleCatalogDetailPage } from './module_catalog_detail_page';

defineOptions({ name: 'MetaModuleCatalogPage' });

const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleCatalog' });
const route = useRoute();

const { catalogStore, operationStore, recordId, loading, missing, moduleName } = useModuleCatalogDetailPage({
  scopeManager: useScopeManager().menuScopeManager,
  moduleName: () => String(route.params.name || ''),
});

const pageTitle = computed(() => moduleName.value || _t('Module'));
</script>
