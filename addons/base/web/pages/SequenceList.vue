<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage>
    <SequenceListView :store="sequenceStore" createAction="/base/sequences/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SequenceListView from '../views/SequenceListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import type Sequence from '@/base/service/models/sequence';

defineOptions({ name: 'SequenceListPage' });

const route = useRoute();
const sequenceStore = createStoreByModel<typeof Sequence>('base.Sequence', {
  storeId: `Sequence_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
