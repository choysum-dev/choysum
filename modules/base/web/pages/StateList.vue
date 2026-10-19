<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="stateStore"
    :action-import="true"
    :action-export="true"
  >
    <StateListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import StateListView from '../views/StateListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type State from '@/base/service/models/state';
import { ChoyPage } from '@/web';

defineOptions({ name: 'StateListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/StateList' });
const pageTitle = _t('State List');

const route = useRoute();
const stateStore = createStoreByModel<typeof State>('base.State', {
  storeId: `State_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
