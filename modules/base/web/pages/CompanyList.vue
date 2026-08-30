<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="companyStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code.')"
  >
    <CompanyListView createAction="/base/companies/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import CompanyListView from '../views/CompanyListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Company from '@/base/service/models/company';

defineOptions({ name: 'CompanyListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/CompanyList' });
const pageTitle = _t('Company List');

const route = useRoute();
const companyStore = createStoreByModel<typeof Company>('base.Company', {
  storeId: `Company_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
