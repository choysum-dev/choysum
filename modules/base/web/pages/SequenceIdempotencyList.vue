<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <SequenceIdempotencyListView :store="sequenceIdempotencyStore" createAction="/base/sequence-idempotencies/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SequenceIdempotencyListView from '../views/SequenceIdempotencyListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';

defineOptions({ name: 'SequenceIdempotencyListPage' });

const route = useRoute();
const sequenceIdempotencyStore = createStoreByModel<typeof SequenceIdempotency>('base.SequenceIdempotency', {
  storeId: `SequenceIdempotency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
