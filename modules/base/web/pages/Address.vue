<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="addressStore">
    <AddressFormView :key="$route.fullPath" createAction="/base/addresses/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import AddressFormView from '../views/AddressFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Address from '@/base/service/models/address';

defineOptions({ name: 'AddressPage' });

const route = useRoute();
const addressStore = createStoreByModel<typeof Address>('base.Address', {
  storeId: `Address_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
