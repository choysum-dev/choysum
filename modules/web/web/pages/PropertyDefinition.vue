<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="definitionStore" :store="definitionStore">
    <PropertyDefinitionFormView
      :key="application"
      :application="application"
      :store="definitionStore"
      :initial-values="initialValues"
      :create-action="createPath"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import PropertyDefinitionFormView from '../views/PropertyDefinitionFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { usePropertyDefinitionDetailPage } from './property_definition_detail_page';

defineOptions({ name: 'PropertyDefinitionPage' });

const route = useRoute();
const { application, definitionStore } = usePropertyDefinitionDetailPage({
  route,
  scopeManager: useScopeManager().menuScopeManager,
});

const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/property-definitions/${app}/new` : '';
});

// Leave scope empty on New: prefilling a concrete properties scope can collide
// with an existing unique (TargetModel, PropertiesField[, Container*]) row.
const initialValues: Record<string, unknown> = {
  TargetModel: null,
  PropertiesField: null,
  ContainerModel: null,
  ContainerId: null,
  Definition: [],
};
</script>
