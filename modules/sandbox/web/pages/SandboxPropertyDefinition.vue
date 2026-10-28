<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="definitionStore">
    <SandboxPropertyDefinitionFormView
      :initial-values="initialValues"
      :create-action="'/sandbox/configuration/record-properties/new'"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import SandboxPropertyDefinitionFormView from '../views/SandboxPropertyDefinitionFormView.vue';

defineOptions({ name: 'SandboxPropertyDefinitionPage' });

const route = useRoute();
const definitionStore = createStoreByModel('sandbox.PropertyDefinition', {
  storeId: `SandboxPropertyDefinition_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

const initialValues: Record<string, unknown> = {
  TargetModel: 'SandboxRecord',
  PropertiesField: 'Properties',
  ContainerModel: null,
  ContainerId: null,
  Definition: [],
};
</script>
