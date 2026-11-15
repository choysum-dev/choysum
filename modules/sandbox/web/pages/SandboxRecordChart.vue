<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="recordStore">
    <SandboxRecordChartView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { ChoyPage } from '@/web';
import SandboxRecordChartView from '../views/SandboxRecordChartView.vue';
import { createModelStore } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';

defineOptions({ name: 'SandboxRecordChartPage' });

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxRecordChart' });
const pageTitle = _t('Sandbox Record Chart');

const recordStore = createModelStore<typeof SandboxRecord>('sandbox.SandboxRecord', {
  storeId: 'SandboxRecord_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
