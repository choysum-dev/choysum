<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: uomActions.create, edit: uomActions.edit, copy: uomActions.copy, delete: uomActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Unit of Measure Information')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Symbol" /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="CategoryId" :search-view="UoMCategoryListView" :search-view-title="_t('Select Category')"
        /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3"><ChoyBooleanField :store="store" prop="IsReference" /></ChoyCol>
        <ChoyCol :span="3"><ChoyNumberField :store="store" prop="Factor" /></ChoyCol>
        <ChoyCol :span="3"><ChoyNumberField :store="store" prop="Rounding" /></ChoyCol>
        <ChoyCol :span="3"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoM from '@/base/service/models/uom';
import UoMCategoryListView from './UoMCategoryListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyNumberField, ChoyVarcharField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'UoMFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/UoMFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<UoM>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const uomActions = defineModelActions('base.UoM', { entityTitle: _lt('Unit of Measure') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'UoMFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
</style>
