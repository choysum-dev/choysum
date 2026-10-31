<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="fieldDefaultStore" :store="fieldDefaultStore">
    <FieldDefaultFormView
      :key="application"
      :application="application"
      :store="fieldDefaultStore"
      :initial-values="initialValues"
      :create-action="createPath"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import FieldDefaultFormView from '../views/FieldDefaultFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useFieldDefaultDetailPage } from './field_default_detail_page';

defineOptions({ name: 'FieldDefaultPage' });

const route = useRoute();
const { application, fieldDefaultStore } = useFieldDefaultDetailPage({
  route,
  scopeManager: useScopeManager().menuScopeManager,
});

const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/field-defaults/${app}/new` : '';
});

// Empty UserId/CompanyId = global scope dimensions (align Set semantics).
const initialValues: Record<string, unknown> = {
  Model: null,
  Field: null,
  UserId: null,
  CompanyId: null,
  Value: null,
};
</script>
