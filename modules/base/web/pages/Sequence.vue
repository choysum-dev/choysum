<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :store="sequenceStore">
    <SequenceFormView :key="$route.fullPath" createAction="/base/sequences/new" :record-id="recordId" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SequenceFormView from '../views/SequenceFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Sequence from '@/base/service/models/sequence';

defineOptions({ name: 'SequencePage' });

withDefaults(
  defineProps<{
    recordId?: string;
  }>(),
  {}
);

const route = useRoute();
const sequenceStore = createStoreByModel<typeof Sequence>('base.Sequence', {
  storeId: `Sequence_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
