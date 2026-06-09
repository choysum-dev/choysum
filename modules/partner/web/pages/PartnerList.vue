<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <PartnerListView :store="partnerStore" createAction="/partner/partners/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import PartnerListView from '../views/PartnerListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Partner from '@/partner/service/models/partner';

defineOptions({ name: 'PartnerListPage' });

const route = useRoute();

/**
 * Route-scoped store backing the partner list page.
 */
const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
