<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="metaModelDataStore" :title="pageTitle" :store="metaModelDataStore">
    <MetaModelDataListView :store="metaModelDataStore" />
  </ChoyPage>
</template>

<script setup lang="ts">
import MetaModelDataListView from '../views/MetaModelDataListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useMetaModelDataListPage } from './meta_model_data_list_page';

defineOptions({ name: 'MetaModelDataListPage' });

const { pageTitle, metaModelDataStore } = useMetaModelDataListPage({
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
