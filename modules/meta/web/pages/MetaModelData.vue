<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="metaModelDataStore" :store="metaModelDataStore">
    <MetaModelDataFormView :store="metaModelDataStore" view-mode="display" />
  </ChoyPage>
</template>

<script setup lang="ts">
import MetaModelDataFormView from '../views/MetaModelDataFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useMetaModelDataDetailPage } from './meta_model_data_detail_page';

defineOptions({ name: 'MetaModelDataPage' });

const { metaModelDataStore } = useMetaModelDataDetailPage({
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
