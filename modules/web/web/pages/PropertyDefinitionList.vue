<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="definitionStore" :title="pageTitle" :store="definitionStore">
    <PropertyDefinitionListView
      :key="application"
      :application="application"
      :store="definitionStore"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import PropertyDefinitionListView from '../views/PropertyDefinitionListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { usePropertyDefinitionListPage } from './property_definition_list_page';

defineOptions({ name: 'PropertyDefinitionListPage' });

const { application, pageTitle, definitionStore } = usePropertyDefinitionListPage({
  route: useRoute(),
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
