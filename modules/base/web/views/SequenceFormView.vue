<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: sequenceActions.create, edit: sequenceActions.edit, copy: sequenceActions.copy, delete: sequenceActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Sequence Configuration')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="CompanyId" :search-view="CompanyListView" :search-view-title="_t('Select Company')"
        /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3"><ChoyVarcharField :store="store" prop="Prefix" /></ChoyCol>
        <ChoyCol :span="3"><ChoyVarcharField :store="store" prop="Suffix" /></ChoyCol>
        <ChoyCol :span="3"><ChoyIntField :store="store" prop="Padding" /></ChoyCol>
        <ChoyCol :span="3"><ChoyBigintField :store="store" prop="NextNumber" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Sequence from '@/base/service/models/sequence';
import CompanyListView from './CompanyListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyNumberField, ChoyVarcharField, ChoyIntField, ChoyBigintField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'SequenceFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/SequenceFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<Sequence>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const sequenceActions = defineModelActions('base.Sequence', { entityTitle: _lt('Sequence') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'SequenceFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

