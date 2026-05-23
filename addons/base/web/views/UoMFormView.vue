<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: uomActions.create, edit: uomActions.edit, copy: uomActions.copy, delete: uomActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>Unit of Measure Information</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" label="Name" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Symbol" label="Symbol" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="CategoryId" label="Category" :search-view="UoMCategoryListView" search-view-title="Select Category"
        /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="6"><OBooleanField :store="store" prop="IsReference" label="Reference Unit" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><ODecimalField :store="store" prop="Factor" label="Conversion Factor" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><ODecimalField :store="store" prop="Rounding" label="Rounding" /></el-col>
        <el-col :xs="24" :sm="12" :md="6"><OBooleanField :store="store" prop="IsActive" label="Active" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoM from '@/base/service/models/uom';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import UoMCategoryListView from './UoMCategoryListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'UoMFormView', inheritAttrs: true });
const props = withDefaults(
  defineProps<{ store: WebModelStore<UoM>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const uomActions = defineModelActions('base.UoM', { entityTitle: 'Unit of Measure' });
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
