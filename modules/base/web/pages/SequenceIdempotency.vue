<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="sequenceIdempotencyStore">
    <SequenceIdempotencyFormView
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SequenceIdempotencyFormView from '../views/SequenceIdempotencyFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { ChoyPage } from '@/web';

defineOptions({ name: 'SequenceIdempotencyPage' });

const route = useRoute();
const sequenceIdempotencyStore = createStoreByModel<typeof SequenceIdempotency>('base.SequenceIdempotency', {
  storeId: `SequenceIdempotency_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
