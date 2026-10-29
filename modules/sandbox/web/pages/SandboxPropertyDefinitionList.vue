<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="definitionStore">
    <SandboxPropertyDefinitionListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import { ChoyPage } from '@/web';
import SandboxPropertyDefinitionListView from '../views/SandboxPropertyDefinitionListView.vue';

defineOptions({ name: 'SandboxPropertyDefinitionListPage' });

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxPropertyDefinitionList' });
const pageTitle = _t('Record Properties');

const definitionStore = createStoreByModel('sandbox.PropertyDefinition', {
  storeId: 'SandboxPropertyDefinition_List',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
