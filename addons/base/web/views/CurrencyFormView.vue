<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: currencyActions.create, edit: currencyActions.edit, copy: currencyActions.copy, delete: currencyActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>Currency Information</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" label="Name" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" label="Code" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Symbol" label="Symbol" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OIntField :store="store" prop="DecimalDigits" label="Decimal Digits" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><ODecimalField :store="store" prop="Rounding" label="Rounding Precision" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" label="Active" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Currency from '@/base/service/models/currency';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'CurrencyFormView', inheritAttrs: true });
const props = withDefaults(
  defineProps<{ store: WebModelStore<Currency>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const currencyActions = defineModelActions('base.Currency', { entityTitle: 'Currency' });
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
