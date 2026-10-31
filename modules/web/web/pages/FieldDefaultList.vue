<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="fieldDefaultStore" :title="pageTitle" :store="fieldDefaultStore">
    <FieldDefaultListView
      :key="application"
      :application="application"
      :store="fieldDefaultStore"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import FieldDefaultListView from '../views/FieldDefaultListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useFieldDefaultListPage } from './field_default_list_page';

defineOptions({ name: 'FieldDefaultListPage' });

const { application, pageTitle, fieldDefaultStore } = useFieldDefaultListPage({
  route: useRoute(),
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
