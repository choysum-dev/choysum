<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="catalogStore">
    <p v-if="loading" class="p-4 text-sm text-muted-foreground">{{ _t('Loading…') }}</p>
    <p v-else-if="missing" class="p-4 text-sm text-muted-foreground" data-testid="module-show-missing">
      {{ _t('Module not found.') }}
    </p>
    <ModuleShowView
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
import ModuleShowView from '../views/ModuleShowView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import { ChoyPage } from '@/web';
import { useModuleShowPage } from './module_show_page';

defineOptions({ name: 'MetaModuleShowPage' });

const { _t } = createTranslate('meta', { scope: 'web/pages/ModuleShow' });
const route = useRoute();

const { catalogStore, operationStore, recordId, loading, missing, moduleName } = useModuleShowPage({
  scopeManager: useScopeManager().menuScopeManager,
  moduleName: () => String(route.params.name || ''),
});

const pageTitle = computed(() => moduleName.value || _t('Module'));
</script>
