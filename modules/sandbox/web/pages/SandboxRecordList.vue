<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="recordStore">
    <SandboxRecordListView selection-mode="multiple" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { createStoreByModel } from '@/web/web/stores/registry';
import SandboxRecordListView from '../views/SandboxRecordListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { ChoyPage } from '@/web';

defineOptions({ name: 'SandboxRecordListPage' });

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxRecordList' });
const pageTitle = _t('Field Kitchen Sink');

const recordStore = createStoreByModel<typeof SandboxRecord>('sandbox.SandboxRecord', {
  storeId: 'SandboxRecord_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
