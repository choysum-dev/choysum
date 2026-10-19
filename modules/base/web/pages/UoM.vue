<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="uomStore">
    <UoMFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UoMFormView from '../views/UoMFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type UoM from '@/base/service/models/uom';
import { ChoyPage } from '@/web';

defineOptions({ name: 'UoMPage' });

const route = useRoute();
const uomStore = createStoreByModel<typeof UoM>('base.UoM', {
  storeId: `UoM_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
