<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="recordStore">
    <SandboxRecordFormView :initial-values="initialValues" />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import { ChoyPage } from '@/web';
import SandboxRecordFormView from '../views/SandboxRecordFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { useAuthStore } from '@/auth/web/stores/auth';

defineOptions({ name: 'SandboxRecordPage' });

const route = useRoute();
const authStore = useAuthStore();
const recordStore = createStoreByModel<typeof SandboxRecord>('sandbox.SandboxRecord', {
  storeId: `SandboxRecord_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});

const identityMeta = ((authStore.identity as any)?.metadata ?? {}) as {
  activeCompanyId?: string;
};
const activeCompanyId = String(identityMeta.activeCompanyId ?? '').trim();

const initialValues: Partial<SandboxRecord> = {
  IsActive: true,
  Sequence: 10,
  State: 'draft',
  Progress: 'draft',
  IntValue: 0,
  ...(activeCompanyId ? { CompanyId: activeCompanyId } : {}),
};
</script>
