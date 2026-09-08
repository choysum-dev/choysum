<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="partnerStore">
    <PartnerFormView
      :initial-values="initialValues"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import PartnerFormView from '../views/PartnerFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Partner from '@/partner/service/models/partner';
import { useAuthStore } from '@/auth/web/stores/auth';
import { buildPartnerPageInitialValues } from './partner_page_initial_values';

defineOptions({ name: 'PartnerPage' });

const route = useRoute();
const authStore = useAuthStore();
const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

/**
 * Seed values used when the page creates a new partner record.
 */
const identityMeta = ((authStore.identity as any)?.metadata ?? {}) as {
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
};
const initialValues: Partial<Partner> = buildPartnerPageInitialValues(identityMeta);
</script>
