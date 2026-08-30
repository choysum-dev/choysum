<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
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
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Sequence Idempotency Record') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="SequenceId" :search-view="SequenceListView" :search-view-title="_t('Select Sequence')"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="CompanyId" :search-view="CompanyListView" :search-view-title="_t('Select Company')"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="IdempotencyKey" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="6"><OIntField :store="store" prop="Count" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OBooleanField :store="store" prop="DryRun" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OBigintField :store="store" prop="RangeStart" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OBigintField :store="store" prop="RangeEnd" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12"><OVarCharField :store="store" prop="CodeSnapshot" /></el-col>
        <el-col :xs="24" :sm="12"><OVarCharField :store="store" prop="RequestHash" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24"><OJsonobjectField :store="store" prop="FormatSnapshot" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><ODateTimeField :store="store" prop="ExpiresAt" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SequenceIdempotency from '@/base/service/models/sequence_idempotency';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import SequenceListView from './SequenceListView.vue';
import CompanyListView from './CompanyListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';

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
.bfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
