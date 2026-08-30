<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="stateStore">
    <StateFormView :key="$route.fullPath" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import StateFormView from '../views/StateFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type State from '@/base/service/models/state';

defineOptions({ name: 'StatePage' });

const route = useRoute();
const stateStore = createStoreByModel<typeof State>('base.State', {
  storeId: `State_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
