<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="uomStore"
    :action-import="true"
    :action-export="true"
  >
    <UoMListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import UoMListView from '../views/UoMListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type UoM from '@/base/service/models/uom';
import { ChoyPage } from '@/web';

defineOptions({ name: 'UoMListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/UoMList' });
const pageTitle = _t('Unit of Measure List');

const route = useRoute();
const uomStore = createStoreByModel<typeof UoM>('base.UoM', {
  storeId: `UoM_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
