<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage
    :title="pageTitle"
    :store="sequenceStore"
    :action-import="true"
    :action-export="true"
  >
    <SequenceListView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import SequenceListView from '../views/SequenceListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Sequence from '@/base/service/models/sequence';
import { ChoyPage } from '@/web';

defineOptions({ name: 'SequenceListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/SequenceList' });
const pageTitle = _t('Sequence List');

const route = useRoute();
const sequenceStore = createStoreByModel<typeof Sequence>('base.Sequence', {
  storeId: `Sequence_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
