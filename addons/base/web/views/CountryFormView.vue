<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: countryActions.create, edit: countryActions.edit, copy: countryActions.copy, delete: countryActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>Country Information</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" label="Name" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" label="Code" :rules="[{ required: true, message: 'Required' }]" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="PhonePrefix" label="Dialing Code" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="DefaultCurrencyId"
            label="Default Currency"
            :search-view="CurrencyListView"
            search-view-title="Select Currency"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="ZipRequired" label="ZIP Required" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="StateRequired" label="State/Province Required" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" label="Active" /></el-col>
        <el-col :xs="24"><OTextField :store="store" prop="AddressFormat" label="Address Format" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Country from '@/base/service/models/country';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import CurrencyListView from './CurrencyListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'CountryFormView', inheritAttrs: true });
const props = withDefaults(
  defineProps<{ store: WebModelStore<Country>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const countryActions = defineModelActions('base.Country', { entityTitle: 'Country' });
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
