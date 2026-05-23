<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: sequenceActions.create, edit: sequenceActions.edit, copy: sequenceActions.copy, delete: sequenceActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>Sequence Configuration</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" label="Name" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" label="Code" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="CompanyId" label="Company" :search-view="CompanyListView" search-view-title="Select Company"
        /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="6"><OVarCharField :store="store" prop="Prefix" label="Prefix" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OVarCharField :store="store" prop="Suffix" label="Suffix" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OIntField :store="store" prop="Padding" label="Padding Length" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OBigintField :store="store" prop="NextNumber" label="Next Number" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="6"><OBooleanField :store="store" prop="IsActive" label="Active" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Sequence from '@/base/service/models/sequence';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import CompanyListView from './CompanyListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'SequenceFormView', inheritAttrs: true });
const props = withDefaults(
  defineProps<{ store: WebModelStore<Sequence>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const sequenceActions = defineModelActions('base.Sequence', { entityTitle: 'Sequence' });
const { hasAction } = usePermission();
const { store, recordId, viewMode, showHeader, createAction } = props;
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
