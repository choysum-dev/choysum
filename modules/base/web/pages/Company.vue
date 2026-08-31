<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="companyStore">
    <CompanyFormView
      :initial-values="initialValues"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CompanyFormView from '../views/CompanyFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Company from '@/base/service/models/company';

defineOptions({ name: 'CompanyPage' });

const route = useRoute();
const companyStore = createStoreByModel<typeof Company>('base.Company', {
  storeId: `Company_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
const initialValues = {
  Timezone: browserTimezone,
} as const;
</script>
