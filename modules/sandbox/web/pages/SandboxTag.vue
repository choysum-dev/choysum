<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="tagStore">
    <SandboxTagFormView :initial-values="initialValues" :create-action="'/sandbox/tags/new'" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { ChoyPage } from '@/web';
import SandboxTagFormView from '../views/SandboxTagFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type SandboxTag from '@/sandbox/service/models/sandbox_tag';

defineOptions({ name: 'SandboxTagPage' });

const route = useRoute();
const tagStore = createStoreByModel<typeof SandboxTag>('sandbox.SandboxTag', {
  storeId: `SandboxTag_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

const initialValues: Partial<SandboxTag> = {
  IsActive: true,
};
</script>
