<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="addressStore">
    <AddressFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import AddressFormView from '../views/AddressFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Address from '@/base/service/models/address';
import { ChoyPage } from '@/web';

defineOptions({ name: 'AddressPage' });

const route = useRoute();
const addressStore = createStoreByModel<typeof Address>('base.Address', {
  storeId: `Address_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
