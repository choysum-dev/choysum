<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :store="sequenceStore">
    <SequenceFormView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SequenceFormView from '../views/SequenceFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Sequence from '@/base/service/models/sequence';
import { ChoyPage } from '@/web';

defineOptions({ name: 'SequencePage' });

const route = useRoute();
const sequenceStore = createStoreByModel<typeof Sequence>('base.Sequence', {
  storeId: `Sequence_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
