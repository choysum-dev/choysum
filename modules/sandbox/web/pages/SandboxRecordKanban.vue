<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle" :store="recordStore">
    <SandboxRecordKanbanView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { ChoyPage } from '@/web';
import SandboxRecordKanbanView from '../views/SandboxRecordKanbanView.vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';

defineOptions({ name: 'SandboxRecordKanbanPage' });

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxRecordKanban' });
const pageTitle = _t('Sandbox Record Kanban');

const recordStore = createStoreByModel<typeof SandboxRecord>('sandbox.SandboxRecord', {
  storeId: 'SandboxRecord_ListKanban',
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
