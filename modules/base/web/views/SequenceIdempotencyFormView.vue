<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{
      create: sequenceIdempotencyActions.create,
      edit: sequenceIdempotencyActions.edit,
      copy: sequenceIdempotencyActions.copy,
      delete: sequenceIdempotencyActions.delete,
    }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Sequence Idempotency Record')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="SequenceId" :search-view="SequenceListView" :search-view-title="_t('Select Sequence')"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="CompanyId" :search-view="CompanyListView" :search-view-title="_t('Select Company')"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="IdempotencyKey" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3"><ChoyNumberField :store="store" prop="Count" /></ChoyCol>
        <ChoyCol :span="3"><ChoyBooleanField :store="store" prop="DryRun" /></ChoyCol>
        <ChoyCol :span="3"><ChoyNumberField :store="store" prop="RangeStart" /></ChoyCol>
        <ChoyCol :span="3"><ChoyNumberField :store="store" prop="RangeEnd" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6"><ChoyVarcharField :store="store" prop="CodeSnapshot" /></ChoyCol>
        <ChoyCol :span="6"><ChoyVarcharField :store="store" prop="RequestHash" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="12"><ChoyJsonField :store="store" prop="FormatSnapshot" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyDatetimeField :store="store" prop="ExpiresAt" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import SequenceListView from './SequenceListView.vue';
import CompanyListView from './CompanyListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyJsonField, ChoyManyToOneField, ChoyNumberField, ChoyVarcharField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'SequenceIdempotencyFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceIdempotencyFormView' });
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SequenceIdempotency>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  { showHeader: true, createAction: undefined }
);
const sequenceIdempotencyActions = defineModelActions('base.SequenceIdempotency', { entityTitle: _lt('Sequence Idempotency Record') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'SequenceIdempotencyFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
</style>
