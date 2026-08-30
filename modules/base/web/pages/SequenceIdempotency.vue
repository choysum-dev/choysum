<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="sequenceIdempotencyStore">
    <SequenceIdempotencyFormView
      :key="$route.fullPath"
      createAction="/base/sequence-idempotencies/new"
      :record-id="recordId"
      :view-mode="viewMode"
    />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SequenceIdempotencyFormView from '../views/SequenceIdempotencyFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';

defineOptions({ name: 'SequenceIdempotencyPage' });

withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const sequenceIdempotencyStore = createStoreByModel<typeof SequenceIdempotency>('base.SequenceIdempotency', {
  storeId: `SequenceIdempotency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
