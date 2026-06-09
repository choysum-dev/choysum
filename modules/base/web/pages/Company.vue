<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <CompanyFormView
      :key="$route.fullPath"
      createAction="/base/companies/new"
      :store="companyStore"
      :record-id="recordId"
      :view-mode="viewMode"
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
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type Company from '@/base/service/models/company';

defineOptions({ name: 'CompanyPage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

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
