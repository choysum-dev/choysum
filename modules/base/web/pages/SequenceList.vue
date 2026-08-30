<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage
    :title="pageTitle"
    :store="sequenceStore"
    action-import
    action-export
    :action-import-upload-hint="_t('Upload a UTF-8 CSV with columns Name, Code, Prefix, Suffix, Padding, NextNumber, IsActive.')"
  >
    <SequenceListView createAction="/base/sequences/new" />
  </OPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { createStoreByModel } from '@/web/web/stores/registry';
import OPage from '@/web/web/components/page/OPage.vue';
import SequenceListView from '../views/SequenceListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createTranslate } from '@/web/web/i18n';
import type Sequence from '@/base/service/models/sequence';

defineOptions({ name: 'SequenceListPage' });

const { _t } = createTranslate('base', { scope: 'web/pages/SequenceList' });
const pageTitle = _t('Sequence List');

const route = useRoute();
const sequenceStore = createStoreByModel<typeof Sequence>('base.Sequence', {
  storeId: `Sequence_${route.fullPath}`,
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
