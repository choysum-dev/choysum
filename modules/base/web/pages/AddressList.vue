<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="addressStore"
    :action-import="true"
    :action-export="true"
  >
    <AddressListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import AddressListView from '../views/AddressListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Address from '@/base/service/models/address';
import { ChoyPage } from '@/web';

defineOptions({ name: 'AddressListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/AddressList' });
const pageTitle = _t('Address List');

const route = useRoute();
const addressStore = createStoreByModel<typeof Address>('base.Address', {
  storeId: `Address_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
