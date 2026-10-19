<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="sequenceIdempotencyStore"
    :action-import="true"
    :action-export="true"
  >
    <SequenceIdempotencyListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SequenceIdempotencyListView from '../views/SequenceIdempotencyListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { ChoyPage } from '@/web';

defineOptions({ name: 'SequenceIdempotencyListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/SequenceIdempotencyList' });
const pageTitle = _t('Sequence Idempotency List');

const route = useRoute();
const sequenceIdempotencyStore = createStoreByModel<typeof SequenceIdempotency>('base.SequenceIdempotency', {
  storeId: `SequenceIdempotency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
