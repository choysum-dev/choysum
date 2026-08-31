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

defineOptions({ name: 'PartnerPage' });

const route = useRoute();
const authStore = useAuthStore();
const partnerStore = createStoreByModel<typeof Partner>('partner.Partner', {
  storeId: `Partner_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

/**
 * Auth metadata used to derive the default company for new partner records.
 */
const identityMeta = ((authStore.identity as any)?.metadata ?? {}) as {
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
};
const normalizedActiveCompanyId = String(identityMeta.activeCompanyId ?? '').trim();
const normalizedEnabledCompanyIds = Array.isArray(identityMeta.enabledCompanyIds)
  ? identityMeta.enabledCompanyIds.map(id => String(id ?? '').trim()).filter(Boolean)
  : [];
const defaultCompanyId =
  normalizedActiveCompanyId && normalizedEnabledCompanyIds.includes(normalizedActiveCompanyId) ? normalizedActiveCompanyId : normalizedEnabledCompanyIds[0];

/**
 * Seed values used when the page creates a new partner record.
 */
const initialValues: Partial<Partner> = {
  CompanyId: defaultCompanyId,
  IsActive: true,
  IsCompany: true,
  CustomerRank: 0,
  SupplierRank: 0,
  Contacts: [],
  Sequence: 10,
};
</script>
